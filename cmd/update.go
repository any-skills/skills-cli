package cmd

import (
	stderrors "errors"
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/rushteam/skills-cli/internal/config"
	"github.com/rushteam/skills-cli/internal/registry"
	"github.com/rushteam/skills-cli/internal/skill"
	"github.com/rushteam/skills-cli/internal/source"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [skills...]",
	Short: "Update installed skills to latest versions",
	Long: `Update installed skills to their latest versions.

With no arguments, all tracked skills are checked. Pass one or more skill
names to update only those.`,
	RunE: runUpdate,
}

func runUpdate(cmd *cobra.Command, args []string) error {
	lock, err := config.LoadLock()
	if err != nil {
		return err
	}

	if len(lock.Skills) == 0 {
		fmt.Println(dimStyle.Render("No skills tracked in lock file."))
		return nil
	}

	// When skill names are given, restrict the update set to those.
	var only map[string]bool
	if len(args) > 0 {
		only = make(map[string]bool, len(args))
		for _, a := range args {
			if _, ok := lock.Skills[a]; !ok {
				fmt.Println(dimStyle.Render(fmt.Sprintf("  Skipping %q: not tracked in lock file", a)))
				continue
			}
			only[a] = true
		}
		if len(only) == 0 {
			return nil
		}
	}

	showLogo()
	fmt.Println()
	fmt.Println(textStyle.Render("Checking for skill updates..."))
	fmt.Println()

	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	type updateInfo struct {
		name  string
		entry config.SkillLockEntry
	}
	var updates []updateInfo

	for name, entry := range lock.Skills {
		if only != nil && !only[name] {
			continue
		}
		if entry.SkillFolderHash == "" || entry.SkillPath == "" || entry.Source == "" {
			continue
		}
		latestHash, err := registry.FetchSkillFolderHash(entry.Source, entry.SkillPath, entry.Ref, "")
		if err != nil {
			if stderrors.Is(err, registry.ErrRateLimited) {
				fmt.Println(errStyle.Render("GitHub API rate limit exceeded."))
				fmt.Println(dimStyle.Render("Set GITHUB_TOKEN or GH_TOKEN to raise the limit, then retry."))
				return nil
			}
			continue
		}
		if latestHash != entry.SkillFolderHash {
			updates = append(updates, updateInfo{name: name, entry: entry})
		}
	}

	if len(updates) == 0 {
		fmt.Println(okStyle.Render("✓ All skills are up to date"))
		return nil
	}

	fmt.Println(textStyle.Render(fmt.Sprintf("Found %d update(s)", len(updates))))
	fmt.Println()

	var successCount, failCount int
	centralDir := config.SkillsHome()

	for _, u := range updates {
		fmt.Println(textStyle.Render(fmt.Sprintf("  Updating %s...", u.name)))

		ps := source.Parse(u.entry.SourceURL)
		if ps.Type == source.SourceGit || ps.Type == source.SourceGitHub || ps.Type == source.SourceGitLab {
			if u.entry.SkillPath != "" {
				ps.Subpath = filepath.Dir(u.entry.SkillPath)
			}
		}

		tmpDir, skills, err := source.FetchSkills(ps)
		if err != nil {
			fmt.Println(errStyle.Render(fmt.Sprintf("  ✗ %s: %v", u.name, err)))
			failCount++
			continue
		}

		var found *skill.Skill
		for _, s := range skills {
			if s.Name == u.name {
				found = s
				break
			}
		}

		if found == nil {
			source.Cleanup(tmpDir, ps)
			fmt.Println(errStyle.Render(fmt.Sprintf("  ✗ %s: skill not found in source", u.name)))
			failCount++
			continue
		}

		dstDir := filepath.Join(centralDir, u.name)
		if err := skill.ReplaceDir(found.Path, dstDir); err != nil {
			source.Cleanup(tmpDir, ps)
			fmt.Println(errStyle.Render(fmt.Sprintf("  ✗ %s: %v", u.name, err)))
			failCount++
			continue
		}

		newHash, _ := registry.FetchSkillFolderHash(u.entry.Source, u.entry.SkillPath, u.entry.Ref, "")
		entry := u.entry
		if newHash != "" {
			entry.SkillFolderHash = newHash
		}
		lock.AddSkill(u.name, entry)

		source.Cleanup(tmpDir, ps)
		fmt.Println(okStyle.Render(fmt.Sprintf("  ✓ %s", u.name)))
		successCount++
	}

	config.SaveLock(lock)

	fmt.Println()
	if successCount > 0 {
		fmt.Println(okStyle.Render(fmt.Sprintf("✓ Updated %d skill(s)", successCount)))
	}
	if failCount > 0 {
		fmt.Println(errStyle.Render(fmt.Sprintf("Failed to update %d skill(s)", failCount)))
	}
	fmt.Println()
	return nil
}

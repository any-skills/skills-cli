package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/rushteam/skills-cli/internal/config"
	"github.com/rushteam/skills-cli/internal/registry"
	"github.com/rushteam/skills-cli/internal/skill"
	"github.com/rushteam/skills-cli/internal/source"
	"github.com/spf13/cobra"
)

var (
	addSkill []string
	addYes   bool
	addAll   bool
)

var addCmd = &cobra.Command{
	Use:   "add <source>",
	Short: "Install a skill from GitHub, GitLab, or local path",
	Long: `Install skills from various sources:
  skills-cli add owner/repo              # GitHub shorthand
  skills-cli add https://github.com/o/r  # GitHub URL
  skills-cli add ./local-skills          # Local path
  skills-cli add owner/repo --skill name # Specific skill`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringSliceVarP(&addSkill, "skill", "s", nil, "Specific skill name(s) to install")
	addCmd.Flags().BoolVarP(&addYes, "yes", "y", false, "Skip confirmation prompts")
	addCmd.Flags().BoolVar(&addAll, "all", false, "Install all skills found in the source")
}

func runAdd(cmd *cobra.Command, args []string) error {
	return runAddWithArgs(args[0], "")
}

func runAddWithArgs(src string, skillFilter string) error {
	showLogo()
	fmt.Println()

	ps := source.Parse(src)
	ownerRepo := source.GetOwnerRepo(ps)

	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	fmt.Println(titleStyle.Render("  Installing skills"))
	if ownerRepo != "" {
		fmt.Println(dimStyle.Render(fmt.Sprintf("  Source: %s", ownerRepo)))
	}
	fmt.Println()

	if skillFilter != "" && len(addSkill) == 0 {
		addSkill = []string{skillFilter}
	}

	if ps.SkillFilter != "" && len(addSkill) == 0 {
		addSkill = []string{ps.SkillFilter}
	}

	tmpDir, skills, err := source.FetchSkills(ps)
	if err != nil {
		return fmt.Errorf("failed to fetch skills: %w", err)
	}
	defer source.Cleanup(tmpDir, ps)

	if len(skills) == 0 {
		fmt.Println(dimStyle.Render("  No skills found in source"))
		return nil
	}

	if len(addSkill) > 0 {
		filterSet := make(map[string]bool)
		for _, s := range addSkill {
			filterSet[s] = true
		}
		var filtered []*skill.Skill
		for _, s := range skills {
			if filterSet[s.Name] {
				filtered = append(filtered, s)
			}
		}
		skills = filtered
		if len(skills) == 0 {
			fmt.Println(dimStyle.Render("  No matching skills found for the specified filter"))
			return nil
		}
	}

	fmt.Printf("  Found %d skill(s):\n", len(skills))
	for _, s := range skills {
		fmt.Printf("    %s - %s\n", titleStyle.Render(s.Name), dimStyle.Render(s.Description))
	}
	fmt.Println()

	if !addYes && !addAll && len(skills) > 1 {
		var selectedNames []string
		var options []huh.Option[string]
		for _, s := range skills {
			options = append(options, huh.NewOption(fmt.Sprintf("%s - %s", s.Name, s.Description), s.Name))
		}

		err := huh.NewMultiSelect[string]().
			Title("Select skills to install:").
			Options(options...).
			Value(&selectedNames).
			Run()
		if err != nil {
			return nil
		}

		nameSet := make(map[string]bool)
		for _, n := range selectedNames {
			nameSet[n] = true
		}
		var selected []*skill.Skill
		for _, s := range skills {
			if nameSet[s.Name] {
				selected = append(selected, s)
			}
		}
		skills = selected
	}

	centralDir := config.SkillsHome()
	lock, _ := config.LoadLock()

	for _, s := range skills {
		dstDir := filepath.Join(centralDir, s.Name)
		if err := skill.ReplaceDir(s.Path, dstDir); err != nil {
			fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(
				fmt.Sprintf("  ✗ %s: %v", s.Name, err)))
			continue
		}

		skillPath := getRelSkillPath(tmpDir, s.Path)

		// Record the folder hash so `check`/`update` can detect upstream changes.
		// Only GitHub-hosted skills support this; failures are non-fatal.
		var folderHash string
		if ownerRepo != "" && (ps.Type == source.SourceGitHub || ps.Type == source.SourceGit) {
			if h, err := registry.FetchSkillFolderHash(ownerRepo, skillPath, ps.Ref, ""); err == nil {
				folderHash = h
			}
		}

		lock.AddSkill(s.Name, config.SkillLockEntry{
			Source:          ownerRepo,
			SourceType:      string(ps.Type),
			SourceURL:       ps.URL,
			Ref:             ps.Ref,
			SkillPath:       skillPath,
			SkillFolderHash: folderHash,
		})
		fmt.Println(okStyle.Render(fmt.Sprintf("  ✓ %s", s.Name)))
	}

	if err := config.SaveLock(lock); err != nil {
		fmt.Println(dimStyle.Render(fmt.Sprintf("  Warning: failed to save lock file: %v", err)))
	}

	fmt.Println()
	fmt.Println(okStyle.Render("  Done!"))
	fmt.Println(dimStyle.Render("  Review skills before use; they run with full agent permissions."))
	fmt.Println()
	return nil
}

func getRelSkillPath(base, skillPath string) string {
	rel, err := filepath.Rel(base, skillPath)
	if err != nil {
		return skillPath
	}
	return rel
}

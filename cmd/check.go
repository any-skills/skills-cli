package cmd

import (
	"encoding/json"
	stderrors "errors"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/rushteam/skills-cli/internal/config"
	"github.com/rushteam/skills-cli/internal/registry"
	"github.com/spf13/cobra"
)

var checkJSON bool

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check for available skill updates",
	RunE:  runCheck,
}

func init() {
	checkCmd.Flags().BoolVar(&checkJSON, "json", false, "Output as JSON")
}

func runCheck(cmd *cobra.Command, args []string) error {
	lock, err := config.LoadLock()
	if err != nil {
		return err
	}

	if len(lock.Skills) == 0 {
		if checkJSON {
			fmt.Println(`{"updates":[],"skipped":[],"errors":[]}`)
			return nil
		}
		fmt.Println(dimStyle.Render("No skills tracked in lock file."))
		fmt.Println(dimStyle.Render("Install skills with: skills-cli add <source>"))
		return nil
	}

	if !checkJSON {
		fmt.Println(textStyle.Render("Checking for skill updates..."))
		fmt.Println()
	}

	cyanStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	var updates []string
	var skipped []string
	var errors []string
	rateLimited := false

	for name, entry := range lock.Skills {
		if entry.SkillFolderHash == "" || entry.SkillPath == "" || entry.Source == "" {
			skipped = append(skipped, name)
			continue
		}

		latestHash, err := registry.FetchSkillFolderHash(entry.Source, entry.SkillPath, entry.Ref, "")
		if err != nil {
			if stderrors.Is(err, registry.ErrRateLimited) {
				rateLimited = true
				break
			}
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))
			continue
		}

		if latestHash != entry.SkillFolderHash {
			updates = append(updates, name)
		}
	}

	if checkJSON {
		out := map[string]any{
			"updates":      updates,
			"skipped":      skipped,
			"errors":       errors,
			"rate_limited": rateLimited,
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if rateLimited {
		fmt.Println(errStyle.Render("GitHub API rate limit exceeded."))
		fmt.Println(dimStyle.Render("Set GITHUB_TOKEN or GH_TOKEN to raise the limit, then retry."))
		fmt.Println()
		return nil
	}

	if len(updates) == 0 {
		fmt.Println(okStyle.Render("✓ All skills are up to date"))
	} else {
		fmt.Println(textStyle.Render(fmt.Sprintf("%d update(s) available:", len(updates))))
		fmt.Println()
		for _, name := range updates {
			fmt.Printf("  %s %s\n", cyanStyle.Render("↑"), name)
		}
		fmt.Println()
		fmt.Println(dimStyle.Render("Run ") + textStyle.Render("skills-cli update") + dimStyle.Render(" to update all skills"))
	}

	if len(skipped) > 0 {
		fmt.Println()
		fmt.Println(dimStyle.Render(fmt.Sprintf("%d skill(s) cannot be checked automatically:", len(skipped))))
		for _, name := range skipped {
			fmt.Printf("  %s %s\n", dimStyle.Render("•"), name)
		}
	}

	if len(errors) > 0 {
		fmt.Println()
		fmt.Println(dimStyle.Render(fmt.Sprintf("Could not check %d skill(s):", len(errors))))
		for _, e := range errors {
			fmt.Printf("  %s %s\n", dimStyle.Render("✗"), e)
		}
	}
	fmt.Println()
	return nil
}

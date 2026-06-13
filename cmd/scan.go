package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
	"github.com/rushteam/skills-cli/internal/agent"
	"github.com/rushteam/skills-cli/internal/config"
	"github.com/rushteam/skills-cli/internal/skill"
	syncer "github.com/rushteam/skills-cli/internal/sync"
	"github.com/spf13/cobra"
)

var (
	scanAgent   []string
	scanProject []string
	scanAll     bool
	scanDiff    bool
	scanJSON    bool
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan agent directories for skills without pulling",
	Long: `Scan global agent directories and/or project-level agent directories,
showing discovered skills and their sync status against the central store.
No files are copied or modified.`,
	RunE: runScan,
}

func init() {
	scanCmd.Flags().StringSliceVarP(&scanAgent, "agent", "a", nil, "Agent(s) to scan")
	scanCmd.Flags().StringSliceVarP(&scanProject, "project", "p", nil, "Project path(s) to scan")
	scanCmd.Flags().BoolVar(&scanAll, "all", false, "Scan all registered projects and global agents")
	scanCmd.Flags().BoolVar(&scanDiff, "diff", false, "Show diff details for modified skills")
	scanCmd.Flags().BoolVar(&scanJSON, "json", false, "Output as JSON")
}

func runScan(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	targets := syncer.ResolveTargets(cfg, scanAgent, scanProject, scanAll)
	if len(targets) == 0 {
		if scanJSON {
			fmt.Println("[]")
			return nil
		}
		fmt.Println(dimStyle.Render("No targets to scan."))
		fmt.Println(dimStyle.Render("Use --agent, --project, or --all to specify sources."))
		fmt.Println(dimStyle.Render("Or configure default_agents in: ") + textStyle.Render(config.ConfigPath()))
		return nil
	}

	if scanJSON {
		return scanJSONOutput(targets)
	}

	showLogo()
	fmt.Println()
	fmt.Println(titleStyle.Render("  Scanning agent directories"))
	fmt.Println()

	centralDir := config.SkillsHome()
	newStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	modifiedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)
	syncedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))

	var totalSkills, totalNew, totalModified, totalSynced int

	for _, t := range targets {
		skillNames, err := agent.ScanSkillsInDir(t.Dir)
		if err != nil {
			fmt.Println(warnStyle.Render(fmt.Sprintf("  Skipping %s (%s): %v", t.AgentName, t.Dir, err)))
			continue
		}
		if len(skillNames) == 0 {
			continue
		}

		label := t.AgentName
		if t.Scope == "project" {
			label = fmt.Sprintf("%s @ %s", t.AgentName, agent.ShortenPath(t.Project))
		}

		fmt.Println(titleStyle.Render(fmt.Sprintf("  [%s] %s", t.Scope, label)))
		fmt.Println(dimStyle.Render(fmt.Sprintf("  %s", agent.ShortenPath(t.Dir))))
		fmt.Println()

		for _, name := range skillNames {
			totalSkills++
			srcDir := filepath.Join(t.Dir, name)
			dstDir := filepath.Join(centralDir, name)

			if !dirExists(dstDir) {
				totalNew++
				fmt.Printf("    %s  %s\n", newStyle.Render("NEW"), textStyle.Render(name))
				continue
			}

			hasDiff, _ := skill.HasDifferences(srcDir, dstDir)
			if hasDiff {
				totalModified++
				fmt.Printf("    %s  %s\n", modifiedStyle.Render("MOD"), textStyle.Render(name))
				if scanDiff {
					diffs, _ := skill.CompareSkillDirs(srcDir, dstDir)
					fmt.Print(skill.FormatDiff(diffs))
				}
			} else {
				totalSynced++
				fmt.Printf("    %s   %s\n", syncedStyle.Render("OK "), textStyle.Render(name))
			}
		}
		fmt.Println()
	}

	fmt.Println(dimStyle.Render("  ─────────────────────────────"))
	fmt.Printf("  Total: %s  ", textStyle.Render(fmt.Sprintf("%d skill(s)", totalSkills)))
	if totalNew > 0 {
		fmt.Printf("%s  ", newStyle.Render(fmt.Sprintf("%d new", totalNew)))
	}
	if totalModified > 0 {
		fmt.Printf("%s  ", modifiedStyle.Render(fmt.Sprintf("%d modified", totalModified)))
	}
	if totalSynced > 0 {
		fmt.Printf("%s", syncedStyle.Render(fmt.Sprintf("%d synced", totalSynced)))
	}
	fmt.Println()

	if totalNew > 0 || totalModified > 0 {
		fmt.Println()
		fmt.Println(dimStyle.Render("  Run ") + textStyle.Render("skills-cli pull") + dimStyle.Render(" to pull these skills to the central store."))
	}
	fmt.Println()

	return nil
}

type scanSkillJSON struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "new", "modified", "synced"
}

type scanTargetJSON struct {
	Agent   string          `json:"agent"`
	Scope   string          `json:"scope"`
	Project string          `json:"project,omitempty"`
	Dir     string          `json:"dir"`
	Skills  []scanSkillJSON `json:"skills"`
}

func scanJSONOutput(targets []syncer.SyncTarget) error {
	centralDir := config.SkillsHome()
	var out []scanTargetJSON

	for _, t := range targets {
		skillNames, err := agent.ScanSkillsInDir(t.Dir)
		if err != nil || len(skillNames) == 0 {
			continue
		}
		entry := scanTargetJSON{
			Agent:   t.AgentName,
			Scope:   t.Scope,
			Project: t.Project,
			Dir:     t.Dir,
		}
		for _, name := range skillNames {
			srcDir := filepath.Join(t.Dir, name)
			dstDir := filepath.Join(centralDir, name)
			status := "synced"
			if !dirExists(dstDir) {
				status = "new"
			} else if hasDiff, _ := skill.HasDifferences(srcDir, dstDir); hasDiff {
				status = "modified"
			}
			entry.Skills = append(entry.Skills, scanSkillJSON{Name: name, Status: status})
		}
		out = append(out, entry)
	}

	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
	return nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

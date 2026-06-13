package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/rushteam/skills-cli/internal/agent"
	"github.com/rushteam/skills-cli/internal/config"
	sk "github.com/rushteam/skills-cli/internal/skill"
	syncer "github.com/rushteam/skills-cli/internal/sync"
	"github.com/spf13/cobra"
)

var (
	removeAgent       []string
	removeYes         bool
	removeAll         bool
	removeCentralOnly bool
)

var removeCmd = &cobra.Command{
	Use:     "remove [skills...]",
	Aliases: []string{"rm"},
	Short:   "Remove installed skills",
	RunE:    runRemove,
}

func init() {
	removeCmd.Flags().StringSliceVarP(&removeAgent, "agent", "a", nil, "Only remove agent copies from these agent(s)")
	removeCmd.Flags().BoolVarP(&removeYes, "yes", "y", false, "Skip confirmation")
	removeCmd.Flags().BoolVar(&removeAll, "all", false, "Remove all skills")
	removeCmd.Flags().BoolVar(&removeCentralOnly, "central-only", false, "Only remove from the central store, leave agent copies in place")
}

func runRemove(cmd *cobra.Command, args []string) error {
	centralDir := config.SkillsHome()
	okStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	skillNames, err := agent.ScanSkillsInDir(centralDir)
	if err != nil || len(skillNames) == 0 {
		fmt.Println(dimStyle.Render("No skills installed."))
		return nil
	}

	var toRemove []string

	if removeAll {
		toRemove = skillNames
	} else if len(args) > 0 {
		toRemove = args
	} else {
		var options []huh.Option[string]
		for _, n := range skillNames {
			options = append(options, huh.NewOption(n, n))
		}

		var selected []string
		err := huh.NewMultiSelect[string]().
			Title("Select skills to remove:").
			Options(options...).
			Value(&selected).
			Run()
		if err != nil || len(selected) == 0 {
			fmt.Println(dimStyle.Render("Cancelled"))
			return nil
		}
		toRemove = selected
	}

	if !removeYes && !removeAll {
		var confirm bool
		err := huh.NewConfirm().
			Title(fmt.Sprintf("Remove %d skill(s)?", len(toRemove))).
			Value(&confirm).
			Run()
		if err != nil || !confirm {
			fmt.Println(dimStyle.Render("Cancelled"))
			return nil
		}
	}

	lock, _ := config.LoadLock()

	// Agent copies to clean up alongside the central store, unless --central-only.
	var targets []syncer.SyncTarget
	if !removeCentralOnly {
		cfg, err := config.Load()
		if err == nil {
			if len(removeAgent) > 0 {
				targets = syncer.ResolveTargets(cfg, removeAgent, nil, false)
			} else {
				targets = syncer.ResolveTargets(cfg, nil, nil, true)
			}
		}
	}

	for _, name := range toRemove {
		dir := filepath.Join(centralDir, name)
		if err := sk.RemoveSkillDir(dir); err != nil && !os.IsNotExist(err) {
			fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(
				fmt.Sprintf("  ✗ %s: %v", name, err)))
			continue
		}
		lock.RemoveSkill(name)
		fmt.Println(okStyle.Render(fmt.Sprintf("  ✓ removed %s", name)))

		for _, t := range targets {
			agentCopy := filepath.Join(t.Dir, name)
			if _, statErr := os.Lstat(agentCopy); statErr != nil {
				continue
			}
			if err := sk.RemoveSkillDir(agentCopy); err != nil && !os.IsNotExist(err) {
				fmt.Println(dimStyle.Render(fmt.Sprintf("      ! %s: %v", agent.ShortenPath(agentCopy), err)))
				continue
			}
			fmt.Println(dimStyle.Render(fmt.Sprintf("      - %s", agent.ShortenPath(agentCopy))))
		}
	}

	config.SaveLock(lock)
	fmt.Println()
	return nil
}

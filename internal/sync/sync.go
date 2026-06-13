package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/rushteam/skills-cli/internal/agent"
	"github.com/rushteam/skills-cli/internal/config"
	"github.com/rushteam/skills-cli/internal/skill"
)

var (
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type SyncTarget struct {
	AgentName string
	Dir       string
	Scope     string // "global" or "project"
	Project   string // project path if scope is "project"
}

type SyncOptions struct {
	Force    bool
	DiffOnly bool
}

func ResolveTargets(cfg *config.Config, agentNames []string, projectPaths []string, all bool) []SyncTarget {
	var targets []SyncTarget

	if all {
		for name, ag := range cfg.Agents {
			globalDir := config.ResolveGlobalPath(ag)
			targets = append(targets, SyncTarget{AgentName: name, Dir: globalDir, Scope: "global"})
		}
		for _, proj := range cfg.Projects {
			agents := proj.Agents
			if len(agents) == 0 {
				agents = agent.DetectProjectAgents(proj.Path, cfg.Agents)
			}
			for _, agName := range agents {
				dir := agent.ResolveProjectSkillsDir(proj.Path, agName, cfg.Agents)
				if dir != "" {
					targets = append(targets, SyncTarget{AgentName: agName, Dir: dir, Scope: "project", Project: proj.Path})
				}
			}
		}
		return targets
	}

	if len(projectPaths) > 0 {
		for _, projPath := range projectPaths {
			absPath, _ := filepath.Abs(projPath)
			agentsToUse := agentNames
			if len(agentsToUse) == 0 {
				agentsToUse = agent.DetectProjectAgents(absPath, cfg.Agents)
			}
			for _, agName := range agentsToUse {
				dir := agent.ResolveProjectSkillsDir(absPath, agName, cfg.Agents)
				if dir != "" {
					targets = append(targets, SyncTarget{AgentName: agName, Dir: dir, Scope: "project", Project: absPath})
				}
			}
		}
		return targets
	}

	if len(agentNames) > 0 {
		for _, name := range agentNames {
			resolved := resolveAgentName(name, cfg.Agents)
			if resolved == "" {
				fmt.Fprintf(os.Stderr, "  Warning: unknown agent %q, skipping\n", name)
				continue
			}
			ag := cfg.Agents[resolved]
			globalDir := config.ResolveGlobalPath(ag)
			targets = append(targets, SyncTarget{AgentName: resolved, Dir: globalDir, Scope: "global"})
		}
		return targets
	}

	if len(cfg.Sync.DefaultAgents) > 0 {
		for _, name := range cfg.Sync.DefaultAgents {
			ag, ok := cfg.Agents[name]
			if !ok {
				continue
			}
			globalDir := config.ResolveGlobalPath(ag)
			targets = append(targets, SyncTarget{AgentName: name, Dir: globalDir, Scope: "global"})
		}
	}

	return targets
}

func Pull(targets []SyncTarget, opts SyncOptions) error {
	centralDir := config.SkillsHome()
	if err := os.MkdirAll(centralDir, 0o755); err != nil {
		return err
	}

	for _, t := range targets {
		skillNames, err := agent.ScanSkillsInDir(t.Dir)
		if err != nil {
			fmt.Println(warnStyle.Render(fmt.Sprintf("  Skipping %s (%s): %v", t.AgentName, t.Dir, err)))
			continue
		}
		if len(skillNames) == 0 {
			continue
		}

		fmt.Println(infoStyle.Render(fmt.Sprintf("  Pulling from %s (%d skills)", targetLabel(t), len(skillNames))))
		for _, name := range skillNames {
			syncSkill(name, filepath.Join(t.Dir, name), filepath.Join(centralDir, name), opts)
		}
	}
	return nil
}

func Push(targets []SyncTarget, opts SyncOptions) error {
	centralDir := config.SkillsHome()
	skillNames, err := agent.ScanSkillsInDir(centralDir)
	if err != nil {
		return fmt.Errorf("failed to scan central skills: %w", err)
	}
	if len(skillNames) == 0 {
		fmt.Println(dimStyle.Render("  No skills in central directory to push"))
		return nil
	}

	for _, t := range targets {
		if !dirExists(t.Dir) {
			fmt.Println(dimStyle.Render(fmt.Sprintf("  Skipping %s: directory does not exist (%s)", targetLabel(t), t.Dir)))
			continue
		}

		fmt.Println(infoStyle.Render(fmt.Sprintf("  Pushing to %s (%d skills)", targetLabel(t), len(skillNames))))
		for _, name := range skillNames {
			syncSkill(name, filepath.Join(centralDir, name), filepath.Join(t.Dir, name), opts)
		}
	}
	return nil
}

func targetLabel(t SyncTarget) string {
	if t.Scope == "project" {
		return fmt.Sprintf("%s @ %s", t.AgentName, agent.ShortenPath(t.Project))
	}
	return t.AgentName
}

// syncSkill copies one skill from srcDir to dstDir, handling conflict detection,
// diff display, prompting and dry-run. It is shared by both push and pull.
func syncSkill(name, srcDir, dstDir string, opts SyncOptions) {
	if dirExists(dstDir) {
		hasDiff, _ := skill.HasDifferences(srcDir, dstDir)
		if !hasDiff {
			fmt.Println(dimStyle.Render(fmt.Sprintf("    %s (up to date)", name)))
			return
		}

		diffs, _ := skill.CompareSkillDirs(srcDir, dstDir)
		fmt.Println(warnStyle.Render(fmt.Sprintf("    %s (conflict)", name)))
		fmt.Print(skill.FormatDiff(diffs))

		if opts.DiffOnly {
			return
		}

		if !opts.Force {
			action, err := promptConflict(name)
			if err != nil || action == "skip" {
				fmt.Println(dimStyle.Render("    Skipped"))
				return
			}
			if action == "keep" {
				fmt.Println(dimStyle.Render("    Kept existing"))
				return
			}
		}
	}

	if opts.DiffOnly {
		fmt.Println(infoStyle.Render(fmt.Sprintf("    %s (would copy)", name)))
		return
	}

	if err := skill.ReplaceDir(srcDir, dstDir); err != nil {
		fmt.Println(errorStyle.Render(fmt.Sprintf("    %s: copy failed: %v", name, err)))
		return
	}
	fmt.Println(successStyle.Render(fmt.Sprintf("    %s ✓", name)))
}

// resolveAgentName returns the exact key in agents that matches name, trying
// exact match first, then prefix match (e.g. "claude" → "claude-code").
func resolveAgentName(name string, agents map[string]config.AgentConfig) string {
	if _, ok := agents[name]; ok {
		return name
	}
	for key := range agents {
		if strings.HasPrefix(key, name) {
			return key
		}
	}
	return ""
}

func promptConflict(skillName string) (string, error) {
	var action string
	err := huh.NewSelect[string]().
		Title(fmt.Sprintf("Conflict: %s already exists. What to do?", skillName)).
		Options(
			huh.NewOption("Overwrite with source", "overwrite"),
			huh.NewOption("Keep existing", "keep"),
			huh.NewOption("Skip", "skip"),
		).
		Value(&action).
		Run()
	return action, err
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

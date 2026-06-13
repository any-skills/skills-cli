package cmd

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/charmbracelet/lipgloss"
	"github.com/rushteam/skills-cli/internal/config"
	"github.com/spf13/cobra"
)

// Version is injected at build time via -ldflags. When the binary is built
// without that flag (e.g. `go install`), it falls back to the module version
// recorded in the build info so `--version` still reports something useful.
var Version = "dev"

func resolveVersion() string {
	if Version != "dev" && Version != "" {
		return Version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return Version
}

var (
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	textStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
)

var logoLines = []string{
	"███████╗██╗  ██╗██╗██╗     ██╗     ███████╗",
	"██╔════╝██║ ██╔╝██║██║     ██║     ██╔════╝",
	"███████╗█████╔╝ ██║██║     ██║     ███████╗",
	"╚════██║██╔═██╗ ██║██║     ██║     ╚════██║",
	"███████║██║  ██╗██║███████╗███████╗███████║",
	"╚══════╝╚═╝  ╚═╝╚═╝╚══════╝╚══════╝╚══════╝",
}

var grays = []lipgloss.Color{"250", "248", "245", "243", "240", "238"}

func showLogo() {
	fmt.Println()
	for i, line := range logoLines {
		style := lipgloss.NewStyle().Foreground(grays[i])
		fmt.Println(style.Render(line))
	}
}

func showBanner() {
	showLogo()
	fmt.Println()
	fmt.Println(dimStyle.Render("The open agent skills management tool"))
	fmt.Println()
	fmt.Printf("  %s %s  %s\n", dimStyle.Render("$"), textStyle.Render("skills-cli add <source>"), dimStyle.Render("Add a skill from GitHub"))
	fmt.Printf("  %s %s      %s\n", dimStyle.Render("$"), textStyle.Render("skills-cli find"), dimStyle.Render("Search for skills"))
	fmt.Printf("  %s %s      %s\n", dimStyle.Render("$"), textStyle.Render("skills-cli list"), dimStyle.Render("List installed skills"))
	fmt.Printf("  %s %s      %s\n", dimStyle.Render("$"), textStyle.Render("skills-cli scan"), dimStyle.Render("Scan agents (dry-run)"))
	fmt.Printf("  %s %s      %s\n", dimStyle.Render("$"), textStyle.Render("skills-cli pull"), dimStyle.Render("Pull skills from agents"))
	fmt.Printf("  %s %s      %s\n", dimStyle.Render("$"), textStyle.Render("skills-cli push"), dimStyle.Render("Push skills to agents"))
	fmt.Println()
	fmt.Printf("  %s %s     %s\n", dimStyle.Render("$"), textStyle.Render("skills-cli check"), dimStyle.Render("Check for updates"))
	fmt.Printf("  %s %s    %s\n", dimStyle.Render("$"), textStyle.Render("skills-cli update"), dimStyle.Render("Update all skills"))
	fmt.Printf("  %s %s     %s\n", dimStyle.Render("$"), textStyle.Render("skills-cli watch"), dimStyle.Render("Watch & auto-sync"))
	fmt.Println()
	fmt.Printf("  Discover skills at %s\n", textStyle.Render("https://skills.sh/"))
	fmt.Println()
}

var rootCmd = &cobra.Command{
	Use:     "skills-cli",
	Short:   "The open agent skills management tool",
	Version: Version,
	Run: func(cmd *cobra.Command, args []string) {
		showBanner()
	},
}

func Execute() {
	rootCmd.Version = resolveVersion()
	rootCmd.InitDefaultVersionFlag()
	if f := rootCmd.Flags().Lookup("version"); f != nil {
		f.Shorthand = "v"
	}
	if err := config.EnsureDirs(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(pushCmd)
	rootCmd.AddCommand(findCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(initSkillCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(configCmd)
}

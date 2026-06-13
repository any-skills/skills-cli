package cmd

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/rushteam/skills-cli/internal/registry"
	"github.com/spf13/cobra"
)

var findCmd = &cobra.Command{
	Use:   "find [query]",
	Short: "Search for skills interactively or by keyword",
	Long:  `Search for skills on skills.sh. Provide a keyword to search directly, or run without arguments for interactive search.`,
	RunE:  runFind,
}

func runFind(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	if query != "" {
		return findNonInteractive(query)
	}
	return findInteractive()
}

func findNonInteractive(query string) error {
	results, err := registry.SearchSkills(query, 10)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Println(dimStyle.Render(fmt.Sprintf("No skills found for %q", query)))
		if hasNonASCII(query) {
			fmt.Println(dimStyle.Render("Tip: skills.sh only indexes English skill names. Try searching in English."))
		}
		return nil
	}

	cyanStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	fmt.Println()

	for _, s := range results {
		pkg := s.Source
		if pkg == "" {
			pkg = s.Slug
		}
		installs := registry.FormatInstalls(s.Installs)
		installBadge := ""
		if installs != "" {
			installBadge = " " + cyanStyle.Render(installs)
		}
		fmt.Printf("  %s@%s%s\n", textStyle.Render(pkg), s.Name, installBadge)
		fmt.Printf("  %s\n\n", dimStyle.Render("└ https://skills.sh/"+s.Slug))
	}

	return selectAndInstall(results)
}

// selectAndInstall lets the user pick one of the search results to install.
// If the prompt is cancelled (e.g. a non-interactive terminal), it returns nil.
func selectAndInstall(results []registry.SearchResult) error {
	var options []huh.Option[string]
	for _, s := range results {
		pkg := s.Source
		if pkg == "" {
			pkg = s.Slug
		}
		label := fmt.Sprintf("%s (%s)", s.Name, pkg)
		if installs := registry.FormatInstalls(s.Installs); installs != "" {
			label += " - " + installs
		}
		options = append(options, huh.NewOption(label, pkg+"@"+s.Name))
	}

	var selected string
	err := huh.NewSelect[string]().
		Title("Select a skill to install:").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil || selected == "" {
		fmt.Println(dimStyle.Render("Cancelled"))
		return nil
	}

	parts := strings.SplitN(selected, "@", 2)
	if len(parts) == 2 {
		fmt.Println()
		fmt.Printf("  Installing %s from %s...\n", titleStyle.Render(parts[1]), dimStyle.Render(parts[0]))
		fmt.Println()
		return runAddWithArgs(parts[0], parts[1])
	}
	return nil
}

func findInteractive() error {
	var query string
	err := huh.NewInput().
		Title("Search skills (English only):").
		Placeholder("e.g. git, react, testing...").
		Value(&query).
		Run()
	if err != nil {
		return nil
	}
	if len(query) < 2 {
		fmt.Println(dimStyle.Render("Query too short (min 2 chars)"))
		return nil
	}

	results, err := registry.SearchSkills(query, 10)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Println(dimStyle.Render("No skills found"))
		if hasNonASCII(query) {
			fmt.Println(dimStyle.Render("Tip: skills.sh only indexes English skill names. Try searching in English."))
		}
		return nil
	}

	return selectAndInstall(results)
}

func hasNonASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}

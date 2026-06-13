package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	difflib "github.com/sergi/go-diff/diffmatchpatch"
)

var (
	addStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	removeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	headerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type FileDiff struct {
	RelPath    string
	Status     string // "modified", "added", "deleted"
	OldContent string
	NewContent string
	Binary     bool
}

// isBinary reports whether content looks like a binary blob. A NUL byte is a
// reliable signal that line-based diffing would produce garbage.
func isBinary(content string) bool {
	return strings.IndexByte(content, 0) != -1
}

func CompareSkillDirs(srcDir, dstDir string) ([]FileDiff, error) {
	srcFiles, err := collectFiles(srcDir)
	if err != nil {
		return nil, err
	}
	dstFiles, err := collectFiles(dstDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	allFiles := make(map[string]bool)
	for f := range srcFiles {
		allFiles[f] = true
	}
	for f := range dstFiles {
		allFiles[f] = true
	}

	var diffs []FileDiff
	for f := range allFiles {
		srcContent, hasSrc := srcFiles[f]
		dstContent, hasDst := dstFiles[f]

		if hasSrc && !hasDst {
			diffs = append(diffs, FileDiff{
				RelPath:    f,
				Status:     "added",
				NewContent: srcContent,
				Binary:     isBinary(srcContent),
			})
		} else if !hasSrc && hasDst {
			diffs = append(diffs, FileDiff{
				RelPath:    f,
				Status:     "deleted",
				OldContent: dstContent,
				Binary:     isBinary(dstContent),
			})
		} else if srcContent != dstContent {
			diffs = append(diffs, FileDiff{
				RelPath:    f,
				Status:     "modified",
				OldContent: dstContent,
				NewContent: srcContent,
				Binary:     isBinary(srcContent) || isBinary(dstContent),
			})
		}
	}
	return diffs, nil
}

func HasDifferences(srcDir, dstDir string) (bool, error) {
	diffs, err := CompareSkillDirs(srcDir, dstDir)
	if err != nil {
		return false, err
	}
	return len(diffs) > 0, nil
}

func FormatDiff(diffs []FileDiff) string {
	if len(diffs) == 0 {
		return dimStyle.Render("  No differences")
	}

	var sb strings.Builder
	for _, d := range diffs {
		if d.Binary {
			label := map[string]string{"added": "new binary file", "deleted": "binary file deleted", "modified": "binary file differs"}[d.Status]
			sb.WriteString(headerStyle.Render(fmt.Sprintf("  ~ %s (%s)", d.RelPath, label)))
			sb.WriteByte('\n')
			continue
		}
		switch d.Status {
		case "added":
			sb.WriteString(headerStyle.Render(fmt.Sprintf("  + %s (new file)", d.RelPath)))
			sb.WriteByte('\n')
			for _, line := range strings.Split(d.NewContent, "\n") {
				sb.WriteString(addStyle.Render(fmt.Sprintf("    + %s", line)))
				sb.WriteByte('\n')
			}
		case "deleted":
			sb.WriteString(headerStyle.Render(fmt.Sprintf("  - %s (deleted)", d.RelPath)))
			sb.WriteByte('\n')
			for _, line := range strings.Split(d.OldContent, "\n") {
				sb.WriteString(removeStyle.Render(fmt.Sprintf("    - %s", line)))
				sb.WriteByte('\n')
			}
		case "modified":
			sb.WriteString(headerStyle.Render(fmt.Sprintf("  ~ %s", d.RelPath)))
			sb.WriteByte('\n')
			sb.WriteString(renderUnifiedDiff(d.OldContent, d.NewContent))
		}
	}
	return sb.String()
}

func renderUnifiedDiff(oldText, newText string) string {
	dmp := difflib.New()
	diffs := dmp.DiffMain(oldText, newText, true)
	diffs = dmp.DiffCleanupSemantic(diffs)

	var sb strings.Builder
	for _, diff := range diffs {
		lines := strings.Split(diff.Text, "\n")
		for i, line := range lines {
			if i == len(lines)-1 && line == "" {
				continue
			}
			switch diff.Type {
			case difflib.DiffInsert:
				sb.WriteString(addStyle.Render(fmt.Sprintf("    + %s", line)))
			case difflib.DiffDelete:
				sb.WriteString(removeStyle.Render(fmt.Sprintf("    - %s", line)))
			case difflib.DiffEqual:
				sb.WriteString(dimStyle.Render(fmt.Sprintf("      %s", line)))
			}
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func collectFiles(dir string) (map[string]string, error) {
	files := make(map[string]string)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[rel] = string(data)
		return nil
	})
	if err != nil && os.IsNotExist(err) {
		return files, nil
	}
	return files, err
}

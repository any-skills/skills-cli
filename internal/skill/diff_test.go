package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCompareSkillDirs(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	writeFile(t, filepath.Join(src, "SKILL.md"), "v2")   // modified
	writeFile(t, filepath.Join(dst, "SKILL.md"), "v1")   // modified
	writeFile(t, filepath.Join(src, "added.txt"), "new") // added in src
	writeFile(t, filepath.Join(dst, "gone.txt"), "old")  // only in dst -> deleted
	writeFile(t, filepath.Join(src, "same.txt"), "same") // unchanged
	writeFile(t, filepath.Join(dst, "same.txt"), "same") // unchanged

	diffs, err := CompareSkillDirs(src, dst)
	if err != nil {
		t.Fatal(err)
	}

	status := map[string]string{}
	for _, d := range diffs {
		status[d.RelPath] = d.Status
	}

	if status["SKILL.md"] != "modified" {
		t.Errorf("SKILL.md status = %q, want modified", status["SKILL.md"])
	}
	if status["added.txt"] != "added" {
		t.Errorf("added.txt status = %q, want added", status["added.txt"])
	}
	if status["gone.txt"] != "deleted" {
		t.Errorf("gone.txt status = %q, want deleted", status["gone.txt"])
	}
	if _, ok := status["same.txt"]; ok {
		t.Errorf("same.txt should not appear in diffs")
	}
}

func TestHasDifferences(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, filepath.Join(src, "SKILL.md"), "x")
	writeFile(t, filepath.Join(dst, "SKILL.md"), "x")

	if has, _ := HasDifferences(src, dst); has {
		t.Errorf("identical dirs reported as differing")
	}

	writeFile(t, filepath.Join(dst, "SKILL.md"), "y")
	if has, _ := HasDifferences(src, dst); !has {
		t.Errorf("differing dirs reported as identical")
	}
}

func TestIsBinary(t *testing.T) {
	if isBinary("plain text") {
		t.Errorf("text classified as binary")
	}
	if !isBinary("has\x00nul") {
		t.Errorf("NUL content not classified as binary")
	}
}

func TestFormatDiffBinary(t *testing.T) {
	diffs := []FileDiff{{RelPath: "logo.png", Status: "modified", Binary: true}}
	out := FormatDiff(diffs)
	if out == "" {
		t.Fatal("expected output")
	}
	// The raw (binary) content must not be dumped into the diff.
	if want := "binary file differs"; !contains(out, want) {
		t.Errorf("FormatDiff output missing %q: %q", want, out)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (indexOf(haystack, needle) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

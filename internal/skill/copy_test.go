package skill

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCopyDirPreservesMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes differ on windows")
	}
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "out")

	script := filepath.Join(src, "run.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(src, "SKILL.md"), "x")

	if err := CopyDir(src, dst); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(filepath.Join(dst, "run.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("executable bit not preserved: mode = %v", info.Mode().Perm())
	}
}

func TestReplaceDir(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "skill")

	writeFile(t, filepath.Join(src, "SKILL.md"), "new content")

	// Pre-existing dst with stale content should be fully replaced.
	writeFile(t, filepath.Join(dst, "SKILL.md"), "old content")
	writeFile(t, filepath.Join(dst, "stale.txt"), "remove me")

	if err := ReplaceDir(src, dst); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(dst, "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new content" {
		t.Errorf("SKILL.md = %q, want %q", got, "new content")
	}
	if _, err := os.Stat(filepath.Join(dst, "stale.txt")); !os.IsNotExist(err) {
		t.Errorf("stale file not removed by ReplaceDir")
	}
	// No temp/old scratch dirs left behind.
	for _, suffix := range []string{".skills-cli-tmp", ".skills-cli-old"} {
		if _, err := os.Stat(dst + suffix); !os.IsNotExist(err) {
			t.Errorf("leftover scratch dir %s%s", dst, suffix)
		}
	}
}

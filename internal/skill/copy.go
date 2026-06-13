package skill

import (
	"io"
	"os"
	"path/filepath"
)

// CopyDir recursively copies src into dst, preserving file mode bits (so
// executable scripts inside a skill keep their +x). Symlinks are recreated as
// symlinks rather than followed.
func CopyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			os.Remove(target)
			return os.Symlink(link, target)
		}
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

// ReplaceDir atomically replaces dst with a copy of src. It copies into a
// temporary sibling directory first, then swaps it into place with rename, so a
// failure mid-copy never leaves dst partially written.
func ReplaceDir(src, dst string) error {
	tmp := dst + ".skills-cli-tmp"
	old := dst + ".skills-cli-old"
	os.RemoveAll(tmp)
	os.RemoveAll(old)

	if err := CopyDir(src, tmp); err != nil {
		os.RemoveAll(tmp)
		return err
	}

	dstExists := false
	if _, err := os.Lstat(dst); err == nil {
		dstExists = true
		if err := os.Rename(dst, old); err != nil {
			os.RemoveAll(tmp)
			return err
		}
	}

	if err := os.Rename(tmp, dst); err != nil {
		// Roll back to the previous contents on failure.
		if dstExists {
			os.Rename(old, dst)
		}
		os.RemoveAll(tmp)
		return err
	}

	os.RemoveAll(old)
	return nil
}

func SymlinkDir(src, dst string) error {
	parent := filepath.Dir(dst)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	os.Remove(dst)
	return os.Symlink(src, dst)
}

func RemoveSkillDir(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return os.Remove(path)
	}
	return os.RemoveAll(path)
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Chmod(perm)
}

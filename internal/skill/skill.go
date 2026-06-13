package skill

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/rushteam/skills-cli/internal/config"
	"gopkg.in/yaml.v3"
)

type Skill struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Path        string `yaml:"-" json:"path"`
	RawContent  string `yaml:"-" json:"-"`
}

func ParseSkillMd(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	frontmatter, err := extractFrontmatter(content)
	if err != nil {
		return nil, err
	}

	var skill Skill
	if err := yaml.Unmarshal([]byte(frontmatter), &skill); err != nil {
		return nil, err
	}
	if skill.Name == "" || skill.Description == "" {
		return nil, nil
	}
	skill.Path = filepath.Dir(path)
	skill.RawContent = content
	return &skill, nil
}

func extractFrontmatter(content string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var lines []string
	inFrontmatter := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			}
			break
		}
		if inFrontmatter {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n"), nil
}

func DiscoverSkills(basePath string) ([]*Skill, error) {
	var skills []*Skill
	seen := make(map[string]bool)

	skillMdPath := filepath.Join(basePath, "SKILL.md")
	if fileExists(skillMdPath) {
		if s, err := ParseSkillMd(skillMdPath); err == nil && s != nil && !seen[s.Name] {
			skills = append(skills, s)
			seen[s.Name] = true
		}
	}

	for _, dir := range discoverSearchDirs(basePath) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			mdPath := filepath.Join(dir, entry.Name(), "SKILL.md")
			if !fileExists(mdPath) {
				continue
			}
			s, err := ParseSkillMd(mdPath)
			if err != nil || s == nil || seen[s.Name] {
				continue
			}
			skills = append(skills, s)
			seen[s.Name] = true
		}
	}

	return skills, nil
}

// discoverSearchDirs returns the directories under basePath that may contain
// skills. The per-agent skills directories are derived from the agent config so
// the list stays in sync with the supported agents instead of being hardcoded.
func discoverSearchDirs(basePath string) []string {
	dirs := []string{
		basePath,
		filepath.Join(basePath, "skills"),
		filepath.Join(basePath, "skills", ".curated"),
		filepath.Join(basePath, "skills", ".experimental"),
		filepath.Join(basePath, "skills", ".system"),
	}
	seen := make(map[string]bool, len(dirs))
	for _, d := range dirs {
		seen[d] = true
	}
	for _, ag := range config.DefaultAgents() {
		if ag.ProjectPath == "" {
			continue
		}
		dir := filepath.Join(basePath, ag.ProjectPath)
		if !seen[dir] {
			dirs = append(dirs, dir)
			seen[dir] = true
		}
	}
	return dirs
}

func ListSkillFiles(skillDir string) ([]string, error) {
	var files []string
	err := filepath.Walk(skillDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(skillDir, path)
			files = append(files, rel)
		}
		return nil
	})
	return files, err
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

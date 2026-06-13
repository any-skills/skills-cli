package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	AppDir      = ".skills-cli"
	ConfigFile  = "config.yaml"
	LockFile    = "skill-lock.json"
	SkillsDir   = "skills"
	LockVersion = 1

	WatchCentralToAgents = "push"
	WatchAgentsToCentral = "pull"
	WatchBidirectional   = "both"
)

var watchDirectionAliases = map[string]string{
	"central_to_agents": WatchCentralToAgents,
	"agents_to_central": WatchAgentsToCentral,
	"bidirectional":     WatchBidirectional,
}

func NormalizeWatchDirection(dir string) string {
	if v, ok := watchDirectionAliases[dir]; ok {
		return v
	}
	return dir
}

func ValidWatchDirections() []string {
	return []string{WatchCentralToAgents, WatchAgentsToCentral, WatchBidirectional}
}

type AgentConfig struct {
	ProjectPath string `yaml:"project_path" json:"project_path"`
	GlobalPath  string `yaml:"global_path" json:"global_path"`
}

type ProjectConfig struct {
	Path   string   `yaml:"path" json:"path"`
	Agents []string `yaml:"agents,omitempty" json:"agents,omitempty"`
}

type SyncConfig struct {
	DefaultAgents []string    `yaml:"default_agents" json:"default_agents"`
	Watch         WatchConfig `yaml:"watch" json:"watch"`
}

type WatchConfig struct {
	Direction string `yaml:"direction" json:"direction"`
}

type Config struct {
	Agents   map[string]AgentConfig `yaml:"agents" json:"agents"`
	Projects []ProjectConfig        `yaml:"projects,omitempty" json:"projects,omitempty"`
	Sync     SyncConfig             `yaml:"sync" json:"sync"`
}

type SkillLockEntry struct {
	Source          string `json:"source"`
	SourceType      string `json:"source_type"`
	SourceURL       string `json:"source_url"`
	Ref             string `json:"ref,omitempty"`
	SkillPath       string `json:"skill_path"`
	SkillFolderHash string `json:"skill_folder_hash"`
	InstalledAt     string `json:"installed_at"`
	UpdatedAt       string `json:"updated_at"`
}

type SkillLock struct {
	Version int                       `json:"version"`
	Skills  map[string]SkillLockEntry `json:"skills"`
}

var (
	instance *Config
	once     sync.Once
)

func AppHome() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, AppDir)
}

func SkillsHome() string {
	return filepath.Join(AppHome(), SkillsDir)
}

func ConfigPath() string {
	return filepath.Join(AppHome(), ConfigFile)
}

func LockPath() string {
	return filepath.Join(AppHome(), LockFile)
}

func expandPath(p string) string {
	if len(p) > 1 && p[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

func ResolveGlobalPath(agent AgentConfig) string {
	return expandPath(agent.GlobalPath)
}

// DeriveDefaultGlobalPath maps a project-relative skills dir to the usual home path,
// e.g. ".cursor/skills" -> "~/.cursor/skills".
func DeriveDefaultGlobalPath(projectPath string) string {
	p := strings.TrimSpace(projectPath)
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return ""
	}
	return "~/" + p
}

func DefaultAgents() map[string]AgentConfig {
	home := "~"
	return map[string]AgentConfig{
		"amp":            {ProjectPath: ".agents/skills", GlobalPath: home + "/.config/agents/skills"},
		"antigravity":    {ProjectPath: ".agents/skills", GlobalPath: home + "/.gemini/antigravity/skills"},
		"augment":        {ProjectPath: ".augment/skills", GlobalPath: home + "/.augment/skills"},
		"claude-code":    {ProjectPath: ".claude/skills", GlobalPath: home + "/.claude/skills"},
		"openclaw":       {ProjectPath: "skills", GlobalPath: home + "/.openclaw/skills"},
		"cline":          {ProjectPath: ".agents/skills", GlobalPath: home + "/.agents/skills"},
		"codebuddy":      {ProjectPath: ".codebuddy/skills", GlobalPath: home + "/.codebuddy/skills"},
		"codex":          {ProjectPath: ".agents/skills", GlobalPath: home + "/.codex/skills"},
		"command-code":   {ProjectPath: ".commandcode/skills", GlobalPath: home + "/.commandcode/skills"},
		"continue":       {ProjectPath: ".continue/skills", GlobalPath: home + "/.continue/skills"},
		"cortex":         {ProjectPath: ".cortex/skills", GlobalPath: home + "/.snowflake/cortex/skills"},
		"crush":          {ProjectPath: ".crush/skills", GlobalPath: home + "/.config/crush/skills"},
		"cursor":         {ProjectPath: ".cursor/skills", GlobalPath: home + "/.cursor/skills"},
		"deepagents":     {ProjectPath: ".agents/skills", GlobalPath: home + "/.deepagents/agent/skills"},
		"droid":          {ProjectPath: ".factory/skills", GlobalPath: home + "/.factory/skills"},
		"firebender":     {ProjectPath: ".agents/skills", GlobalPath: home + "/.firebender/skills"},
		"gemini-cli":     {ProjectPath: ".agents/skills", GlobalPath: home + "/.gemini/skills"},
		"github-copilot": {ProjectPath: ".agents/skills", GlobalPath: home + "/.copilot/skills"},
		"goose":          {ProjectPath: ".goose/skills", GlobalPath: home + "/.config/goose/skills"},
		"junie":          {ProjectPath: ".junie/skills", GlobalPath: home + "/.junie/skills"},
		"iflow-cli":      {ProjectPath: ".iflow/skills", GlobalPath: home + "/.iflow/skills"},
		"kilo":           {ProjectPath: ".kilocode/skills", GlobalPath: home + "/.kilocode/skills"},
		"kimi-cli":       {ProjectPath: ".agents/skills", GlobalPath: home + "/.config/agents/skills"},
		"kiro-cli":       {ProjectPath: ".kiro/skills", GlobalPath: home + "/.kiro/skills"},
		"kode":           {ProjectPath: ".kode/skills", GlobalPath: home + "/.kode/skills"},
		"mcpjam":         {ProjectPath: ".mcpjam/skills", GlobalPath: home + "/.mcpjam/skills"},
		"mistral-vibe":   {ProjectPath: ".vibe/skills", GlobalPath: home + "/.vibe/skills"},
		"mux":            {ProjectPath: ".mux/skills", GlobalPath: home + "/.mux/skills"},
		"opencode":       {ProjectPath: ".agents/skills", GlobalPath: home + "/.config/opencode/skills"},
		"openhands":      {ProjectPath: ".openhands/skills", GlobalPath: home + "/.openhands/skills"},
		"pi":             {ProjectPath: ".pi/skills", GlobalPath: home + "/.pi/agent/skills"},
		"qoder":          {ProjectPath: ".qoder/skills", GlobalPath: home + "/.qoder/skills"},
		"qwen-code":      {ProjectPath: ".qwen/skills", GlobalPath: home + "/.qwen/skills"},
		"replit":         {ProjectPath: ".agents/skills", GlobalPath: home + "/.config/agents/skills"},
		"roo":            {ProjectPath: ".roo/skills", GlobalPath: home + "/.roo/skills"},
		"trae":           {ProjectPath: ".trae/skills", GlobalPath: home + "/.trae/skills"},
		"trae-cn":        {ProjectPath: ".trae/skills", GlobalPath: home + "/.trae-cn/skills"},
		"warp":           {ProjectPath: ".agents/skills", GlobalPath: home + "/.agents/skills"},
		"windsurf":       {ProjectPath: ".windsurf/skills", GlobalPath: home + "/.codeium/windsurf/skills"},
		"zencoder":       {ProjectPath: ".zencoder/skills", GlobalPath: home + "/.zencoder/skills"},
		"neovate":        {ProjectPath: ".neovate/skills", GlobalPath: home + "/.neovate/skills"},
		"pochi":          {ProjectPath: ".pochi/skills", GlobalPath: home + "/.pochi/skills"},
		"adal":           {ProjectPath: ".adal/skills", GlobalPath: home + "/.adal/skills"},
	}
}

func DefaultConfig() *Config {
	return &Config{
		Agents:   DefaultAgents(),
		Projects: []ProjectConfig{},
		Sync: SyncConfig{
			DefaultAgents: []string{},
			Watch: WatchConfig{
				Direction: WatchCentralToAgents,
			},
		},
	}
}

func Load() (*Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}
	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	// Merge in any missing default agents
	defaults := DefaultAgents()
	for name, agent := range defaults {
		if _, ok := cfg.Agents[name]; !ok {
			cfg.Agents[name] = agent
		}
	}
	return cfg, nil
}

func Get() *Config {
	once.Do(func() {
		cfg, err := Load()
		if err != nil {
			cfg = DefaultConfig()
		}
		instance = cfg
	})
	return instance
}

func Reload() (*Config, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}
	instance = cfg
	once = sync.Once{}
	return cfg, nil
}

func (c *Config) Save() error {
	dir := AppHome()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0o644)
}

func (c *Config) AddProject(path string, agents []string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	for i, p := range c.Projects {
		if p.Path == absPath {
			c.Projects[i].Agents = agents
			return
		}
	}
	c.Projects = append(c.Projects, ProjectConfig{Path: absPath, Agents: agents})
}

func (c *Config) RemoveProject(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	for i, p := range c.Projects {
		if p.Path == absPath {
			c.Projects = append(c.Projects[:i], c.Projects[i+1:]...)
			return true
		}
	}
	return false
}

func (c *Config) AddAgent(name string, agent AgentConfig) {
	c.Agents[name] = agent
}

func (c *Config) RemoveAgent(name string) bool {
	if _, ok := c.Agents[name]; ok {
		delete(c.Agents, name)
		return true
	}
	return false
}

// Lock file operations

func LoadLock() (*SkillLock, error) {
	path := LockPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SkillLock{Version: LockVersion, Skills: map[string]SkillLockEntry{}}, nil
		}
		return nil, err
	}
	var lock SkillLock
	if err := json.Unmarshal(data, &lock); err != nil {
		return &SkillLock{Version: LockVersion, Skills: map[string]SkillLockEntry{}}, nil
	}
	if lock.Version < LockVersion {
		return &SkillLock{Version: LockVersion, Skills: map[string]SkillLockEntry{}}, nil
	}
	if lock.Skills == nil {
		lock.Skills = map[string]SkillLockEntry{}
	}
	return &lock, nil
}

func SaveLock(lock *SkillLock) error {
	dir := AppHome()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(LockPath(), data, 0o644)
}

func (l *SkillLock) AddSkill(name string, entry SkillLockEntry) {
	now := time.Now().UTC().Format(time.RFC3339)
	if entry.InstalledAt == "" {
		entry.InstalledAt = now
	}
	entry.UpdatedAt = now
	l.Skills[name] = entry
}

func (l *SkillLock) RemoveSkill(name string) {
	delete(l.Skills, name)
}

func EnsureDirs() error {
	dirs := []string{AppHome(), SkillsHome()}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

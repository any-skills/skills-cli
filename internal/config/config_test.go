package config

import "testing"

func TestDeriveDefaultGlobalPath(t *testing.T) {
	tests := map[string]string{
		".cursor/skills":   "~/.cursor/skills",
		"./.claude/skills": "~/.claude/skills",
		"/.foo/skills":     "~/.foo/skills",
		"":                 "",
	}
	for in, want := range tests {
		if got := DeriveDefaultGlobalPath(in); got != want {
			t.Errorf("DeriveDefaultGlobalPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeWatchDirection(t *testing.T) {
	tests := map[string]string{
		"central_to_agents": WatchCentralToAgents,
		"agents_to_central": WatchAgentsToCentral,
		"bidirectional":     WatchBidirectional,
		"push":              "push",
		"unknown":           "unknown",
	}
	for in, want := range tests {
		if got := NormalizeWatchDirection(in); got != want {
			t.Errorf("NormalizeWatchDirection(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLockAddRemove(t *testing.T) {
	lock := &SkillLock{Version: LockVersion, Skills: map[string]SkillLockEntry{}}

	lock.AddSkill("foo", SkillLockEntry{Source: "o/r", Ref: "main"})
	entry, ok := lock.Skills["foo"]
	if !ok {
		t.Fatal("skill not added")
	}
	if entry.InstalledAt == "" || entry.UpdatedAt == "" {
		t.Errorf("timestamps not set: %+v", entry)
	}
	if entry.Ref != "main" {
		t.Errorf("Ref = %q, want main", entry.Ref)
	}

	// Re-adding preserves InstalledAt.
	first := entry.InstalledAt
	lock.AddSkill("foo", SkillLockEntry{Source: "o/r", InstalledAt: first})
	if lock.Skills["foo"].InstalledAt != first {
		t.Errorf("InstalledAt changed on update")
	}

	lock.RemoveSkill("foo")
	if _, ok := lock.Skills["foo"]; ok {
		t.Errorf("skill not removed")
	}
}

func TestAddRemoveProject(t *testing.T) {
	c := &Config{Agents: map[string]AgentConfig{}}

	c.AddProject("/tmp/proj", []string{"cursor"})
	if len(c.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(c.Projects))
	}

	// Adding the same path again updates agents instead of duplicating.
	c.AddProject("/tmp/proj", []string{"claude-code"})
	if len(c.Projects) != 1 {
		t.Fatalf("duplicate project added, got %d", len(c.Projects))
	}
	if c.Projects[0].Agents[0] != "claude-code" {
		t.Errorf("agents not updated: %v", c.Projects[0].Agents)
	}

	if !c.RemoveProject("/tmp/proj") {
		t.Errorf("RemoveProject returned false")
	}
	if len(c.Projects) != 0 {
		t.Errorf("project not removed")
	}
}

func TestDefaultAgentsResolvable(t *testing.T) {
	for name, ag := range DefaultAgents() {
		if ag.ProjectPath == "" {
			t.Errorf("agent %q has empty ProjectPath", name)
		}
		if ResolveGlobalPath(ag) == "" {
			t.Errorf("agent %q resolves to empty global path", name)
		}
	}
}

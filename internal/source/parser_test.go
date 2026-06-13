package source

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantType    SourceType
		wantURL     string
		wantRef     string
		wantSubpath string
		wantFilter  string
	}{
		{
			name:     "github shorthand",
			input:    "vercel-labs/agent-skills",
			wantType: SourceGitHub,
			wantURL:  "https://github.com/vercel-labs/agent-skills.git",
		},
		{
			name:        "github shorthand with subpath",
			input:       "vercel-labs/agent-skills/frontend",
			wantType:    SourceGitHub,
			wantURL:     "https://github.com/vercel-labs/agent-skills.git",
			wantSubpath: "frontend",
		},
		{
			name:     "github url",
			input:    "https://github.com/vercel-labs/agent-skills",
			wantType: SourceGitHub,
			wantURL:  "https://github.com/vercel-labs/agent-skills.git",
		},
		{
			name:     "github url with .git suffix",
			input:    "https://github.com/vercel-labs/agent-skills.git",
			wantType: SourceGitHub,
			wantURL:  "https://github.com/vercel-labs/agent-skills.git",
		},
		{
			name:        "github tree path",
			input:       "https://github.com/vercel-labs/agent-skills/tree/main/skills/frontend",
			wantType:    SourceGitHub,
			wantURL:     "https://github.com/vercel-labs/agent-skills.git",
			wantRef:     "main",
			wantSubpath: "skills/frontend",
		},
		{
			name:     "github tree no subpath",
			input:    "https://github.com/vercel-labs/agent-skills/tree/dev",
			wantType: SourceGitHub,
			wantURL:  "https://github.com/vercel-labs/agent-skills.git",
			wantRef:  "dev",
		},
		{
			name:       "owner/repo@skill",
			input:      "vercel-labs/agent-skills@frontend-design",
			wantType:   SourceGitHub,
			wantURL:    "https://github.com/vercel-labs/agent-skills.git",
			wantFilter: "frontend-design",
		},
		{
			name:     "gitlab url",
			input:    "https://gitlab.com/org/repo",
			wantType: SourceGitLab,
			wantURL:  "https://gitlab.com/org/repo.git",
		},
		{
			name:        "gitlab tree path",
			input:       "https://gitlab.com/org/repo/-/tree/main/skills/x",
			wantType:    SourceGitLab,
			wantURL:     "https://gitlab.com/org/repo.git",
			wantRef:     "main",
			wantSubpath: "skills/x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if got.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", got.Type, tt.wantType)
			}
			if got.URL != tt.wantURL {
				t.Errorf("URL = %q, want %q", got.URL, tt.wantURL)
			}
			if got.Ref != tt.wantRef {
				t.Errorf("Ref = %q, want %q", got.Ref, tt.wantRef)
			}
			if got.Subpath != tt.wantSubpath {
				t.Errorf("Subpath = %q, want %q", got.Subpath, tt.wantSubpath)
			}
			if got.SkillFilter != tt.wantFilter {
				t.Errorf("SkillFilter = %q, want %q", got.SkillFilter, tt.wantFilter)
			}
		})
	}
}

func TestParseLocalPath(t *testing.T) {
	for _, in := range []string{"./local", "../up", ".", "/abs/path"} {
		if got := Parse(in); got.Type != SourceLocal {
			t.Errorf("Parse(%q).Type = %q, want local", in, got.Type)
		}
	}
}

func TestGetOwnerRepo(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/vercel-labs/agent-skills.git", "vercel-labs/agent-skills"},
		{"https://gitlab.com/org/repo.git", "org/repo"},
		{"git@github.com:owner/repo.git", "owner/repo"},
	}
	for _, tt := range tests {
		got := GetOwnerRepo(ParsedSource{Type: SourceGitHub, URL: tt.url})
		if got != tt.want {
			t.Errorf("GetOwnerRepo(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}

package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// ErrRateLimited is returned when the GitHub API rejects the request because of
// rate limiting. Callers can detect it with errors.Is to hint about GITHUB_TOKEN.
var ErrRateLimited = errors.New("GitHub API rate limit exceeded")

type TreeEntry struct {
	Path string `json:"path"`
	SHA  string `json:"sha"`
	Type string `json:"type"`
}

type TreeResponse struct {
	SHA       string      `json:"sha"`
	Tree      []TreeEntry `json:"tree"`
	Truncated bool        `json:"truncated"`
}

func getGitHubToken() string {
	for _, key := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
		if token := os.Getenv(key); token != "" {
			return token
		}
	}
	return ""
}

// FetchSkillFolderHash returns the git tree SHA of the skill's folder. When ref
// is empty it tries the repository's common default branches in turn so repos
// using "master" (or any non-"main" default) still resolve correctly.
func FetchSkillFolderHash(ownerRepo string, skillPath string, ref string, token string) (string, error) {
	if token == "" {
		token = getGitHubToken()
	}

	refs := []string{ref}
	if ref == "" {
		refs = []string{"HEAD", "main", "master"}
	}

	var lastErr error
	for _, r := range refs {
		hash, err := fetchTreeHash(ownerRepo, skillPath, r, token)
		if err == nil {
			return hash, nil
		}
		if errors.Is(err, ErrRateLimited) {
			return "", err
		}
		lastErr = err
	}
	return "", lastErr
}

func fetchTreeHash(ownerRepo, skillPath, ref, token string) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/git/trees/%s?recursive=1", ownerRepo, ref)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		if resp.Header.Get("X-RateLimit-Remaining") == "0" {
			return "", ErrRateLimited
		}
		return "", ErrRateLimited
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var tree TreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&tree); err != nil {
		return "", fmt.Errorf("failed to decode GitHub tree response: %w", err)
	}

	skillPath = strings.TrimSuffix(skillPath, "/SKILL.md")
	skillPath = strings.TrimSuffix(skillPath, "/")

	// A skill living at the repository root has no dedicated tree entry; use the
	// root tree SHA as its folder hash.
	if skillPath == "" || skillPath == "." {
		return tree.SHA, nil
	}

	for _, entry := range tree.Tree {
		if entry.Type == "tree" && entry.Path == skillPath {
			return entry.SHA, nil
		}
	}

	return "", fmt.Errorf("skill path %q not found in repository tree", skillPath)
}

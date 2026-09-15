// project_gen.go — builds the suggested project.json when
// context-trace analyzes a remote repository (or an unconfigured
// local folder). Shape and key order follow the STRICT contract
// (see docs/i18n/{es,en}/PROJECT.md § "Plantilla generada por
// context-trace"): project/description/repo/lang/adapter/
// default_task/agents/skills/tasks/llm_profile/budget/budget_path/
// token_history_path/jobs. Go's json.MarshalIndent preserves a
// struct's field DECLARATION order (unlike map[string]any), which is
// why every section below is its own named struct rather than a
// generic map. The "debug" key (see core.Project.Debug) is
// intentionally NEVER written here - it defaults to false and is
// documentation-only until a person adds it by hand.
package trace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mova.local/core"
)

type suggestedProject struct {
	Project          string          `json:"project"`
	Author           string          `json:"author"`
	Description      string          `json:"description"`
	Repo             string          `json:"repo"`
	Adapter          string          `json:"adapter"`
	DefaultTask      string          `json:"default_task"`
	Agents           suggestedRef    `json:"agents"`
	Skills           suggestedRef    `json:"skills"`
	Tasks            map[string]task `json:"tasks"`
	LLMProfile       llmProfile      `json:"llm_profile"`
	Budget           suggestedBudget `json:"budget"`
	TokenHistoryPath string          `json:"token_history_path"`
	BudgetPath       string          `json:"budget_path"`
	MemoryPath       string          `json:"memory_path"`
	Tools            toolsConfig     `json:"tools"`
}

type suggestedRef struct {
	Domain string   `json:"domain"`
	Use    []string `json:"use"`
	Custom []string `json:"custom"`
}

type task struct {
	Prompt    string            `json:"prompt"`
	Variables map[string]string `json:"variables"`
	Focus     []string          `json:"focus"`
	Exclude   []string          `json:"exclude"`
}

type llmProfile struct {
	Config string `json:"config"`
}

type suggestedBudget struct {
	MaxTokens  int             `json:"max_tokens"`
	Sanitize   sanitizeOptions `json:"sanitize"`
	CacheHint  bool            `json:"cache_hint"`
	PIIMasking piiMasking      `json:"pii_masking"`
}

type sanitizeOptions struct {
	Enabled       bool `json:"enabled"`
	DedupeLogs    bool `json:"dedupe_logs"`
	StripBlank    bool `json:"strip_blank"`
	StripComments bool `json:"strip_comments"`
}

type piiMasking struct {
	Enabled bool `json:"enabled"`
}

type toolsConfig struct {
	Enabled bool `json:"enabled"`
}

// BuildSuggestedProjectJSON builds the suggested JSON from an already
// completed remote analysis matching the exact contract requested.
func BuildSuggestedProjectJSON(d *Data, projectName string) (string, error) {
	repo := d.RepoDir
	if repo == "" {
		repo = "."
	}
	taskName := taskSlug(d.TaskName)
	focus, exclude := smartFocus(d), smartExclude(d)

	sp := suggestedProject{
		Project:     projectName,
		Author:      "system:default",
		Description: fmt.Sprintf("Auto-generated project profile for %s from remote analysis.", projectName),
		Repo:        repo,
		Adapter:     "file",
		DefaultTask: taskName,
		Agents:      suggestedRef{Domain: "base", Use: []string{}, Custom: []string{}},
		Skills:      suggestedRef{Domain: "base", Use: []string{}, Custom: []string{}},
		Tasks: map[string]task{
			taskName: {
				Prompt:    "",
				Variables: map[string]string{"PROJECT": projectName, "REVIEW_TYPE": "completa"},
				Focus:     focus,
				Exclude:   exclude,
			},
		},
		LLMProfile: llmProfile{Config: defaultLLMConfig(d.RepoDir)},
		Budget: suggestedBudget{
			MaxTokens:  20000,
			Sanitize:   sanitizeOptions{Enabled: true, DedupeLogs: true, StripBlank: true, StripComments: false},
			CacheHint:  true,
			PIIMasking: piiMasking{Enabled: true},
		},
		TokenHistoryPath: filepath.ToSlash(filepath.Join("/projects", projectName, "mova-token-history.json")),
		BudgetPath:       filepath.ToSlash(filepath.Join("/projects", projectName, "mova-budget-report.md")),
		MemoryPath:       filepath.ToSlash(filepath.Join("/projects", projectName, "memory.md")),
		Tools:            toolsConfig{Enabled: false},
	}

	data, err := json.MarshalIndent(sp, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// taskSlug turns a free-text --task into a filesystem/JSON-key-safe slug.
func taskSlug(taskText string) string {
	if taskText == "" {
		return "revisar-backend"
	}
	lower := strings.ToLower(taskText)
	var b strings.Builder
	lastDash := false
	for _, r := range lower {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
		} else if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// smartFocus extracts top relevant files or defaults to root.
func smartFocus(d *Data) []string {
	if len(d.RelevanceTop) == 0 {
		return []string{"."}
	}
	max := 10
	if len(d.RelevanceTop) < max {
		max = len(d.RelevanceTop)
	}
	focus := make([]string, 0, max)
	for _, r := range d.RelevanceTop[:max] {
		focus = append(focus, r.Path)
	}
	return focus
}

// smartExclude defaults to an empty slice.
func smartExclude(d *Data) []string {
	return []string{}
}

// defaultLLMConfig picks the first local model configuration found.
func defaultLLMConfig(root string) string {
	for _, dir := range []string{"google", "ollama", "lmstudio", "vllm"} {
		entries, err := os.ReadDir(filepath.Join(root, "config", "models", dir))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
				return strings.TrimSuffix(e.Name(), ".json")
			}
		}
	}
	return "gemini-2.5-flash"
}

// SetSuggestedProjectRepo unmarshals into suggestedProject to maintain exact field order.
func SetSuggestedProjectRepo(content, newRepoPath string) (string, error) {
	var sp suggestedProject
	if err := json.Unmarshal([]byte(content), &sp); err != nil {
		return "", err
	}
	sp.Repo = newRepoPath
	data, err := json.MarshalIndent(sp, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteSuggestedProjectJSON writes the JSON payload to projects/<name>/project.json.
func WriteSuggestedProjectJSON(root, projectName, content string) (string, error) {
	path := core.ProjectJSONPath(root, projectName)
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("a project.json already exists at %s - not overwriting it", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

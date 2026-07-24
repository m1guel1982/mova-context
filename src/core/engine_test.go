package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildFixtureProject creates a minimal, valid Mova Context tree in a temp
// dir: one agent, one skill, one prompt, one project with a single task
// that uses all three plus a memory.md entry — enough to exercise every
// section BuildContextSections produces (Header/Agents/Skills/Prompt/
// Memory; Focus is exercised separately where needed).
func buildFixtureProject(t *testing.T) (root, projectName string) {
	t.Helper()
	root = t.TempDir()
	write := func(rel, content string) {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("agents/base/dev.md", "# Dev agent\nBe helpful.\n")
	write("skills/base/kiss.md", "# KISS\nKeep it simple.\n")
	write("prompts/base/greet.md", "# Greet\nSay hello to {{PROJECT}}.\n")
	write("projects/fixture/memory.md", "## 2024-01-01\nFirst session.\n")
	write("projects/fixture/project.json", `{
		"project": "fixture",
		"repo": ".",
		"lang": "en",
		"default_task": "say-hi",
		"agents": {"domain": "base", "use": ["dev"]},
		"skills": {"domain": "base", "use": ["kiss"]},
		"tasks": {
			"say-hi": {"prompt": "greet"}
		}
	}`)
	return root, "fixture"
}

func TestBuildContextSections_FullMatchesBuildContext(t *testing.T) {
	root, projectName := buildFixtureProject(t)
	adapter := NewFileAdapter(root)

	fromSections, err := BuildContextSections(adapter, root, projectName, "")
	if err != nil {
		t.Fatalf("BuildContextSections: %v", err)
	}
	fromBuildContext, err := BuildContext(adapter, root, projectName, "")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	if fromSections.Full() != fromBuildContext {
		t.Fatalf("BuildContextSections().Full() must match BuildContext() exactly.\n--- sections.Full() ---\n%s\n--- BuildContext() ---\n%s",
			fromSections.Full(), fromBuildContext)
	}
}

func TestBuildContextSections_PopulatesEachSection(t *testing.T) {
	root, projectName := buildFixtureProject(t)
	adapter := NewFileAdapter(root)

	sections, err := BuildContextSections(adapter, root, projectName, "")
	if err != nil {
		t.Fatalf("BuildContextSections: %v", err)
	}

	checks := map[string]string{
		"Header":      sections.Header,
		"Agents":      sections.Agents,
		"Skills":      sections.Skills,
		"Prompt":      sections.Prompt,
		"Memory":      sections.Memory,
		"Instruction": sections.Instruction,
	}
	for name, content := range checks {
		if content == "" {
			t.Errorf("expected section %s to be non-empty for this fixture", name)
		}
	}
	if sections.Focus != "" {
		t.Errorf("expected Focus to be empty — fixture project has no \"focus\" configured, got: %q", sections.Focus)
	}
}

func TestBuildContextSections_DedupsAcrossAgentsAndSkills(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	shared := "Always validate input before processing it further."
	write("agents/base/dev.md", "# Dev agent\n\n"+shared+"\n\nAgent-only paragraph.")
	write("skills/base/kiss.md", "# KISS\n\n"+shared+"\n\nSkill-only paragraph.")
	write("prompts/base/greet.md", "# Greet\nSay hello to {{PROJECT}}.\n")
	write("projects/dup/memory.md", "")
	write("projects/dup/project.json", `{
		"project": "dup",
		"repo": ".",
		"lang": "en",
		"default_task": "say-hi",
		"agents": {"domain": "base", "use": ["dev"]},
		"skills": {"domain": "base", "use": ["kiss"]},
		"tasks": {"say-hi": {"prompt": "greet"}}
	}`)

	adapter := NewFileAdapter(root)
	sections, err := BuildContextSections(adapter, root, "dup", "")
	if err != nil {
		t.Fatalf("BuildContextSections: %v", err)
	}

	occurrences := strings.Count(sections.Agents+sections.Skills, shared)
	if occurrences != 1 {
		t.Fatalf("expected the shared paragraph to survive exactly once across Agents+Skills, found %d times", occurrences)
	}
	if sections.DuplicatesRemoved != 1 {
		t.Fatalf("expected DuplicatesRemoved=1, got %d", sections.DuplicatesRemoved)
	}
	if !strings.Contains(sections.Agents, "Agent-only paragraph.") || !strings.Contains(sections.Skills, "Skill-only paragraph.") {
		t.Fatal("expected non-duplicate paragraphs to survive untouched")
	}
}

func TestResolveFocus_TaskOverridesProject(t *testing.T) {
	proj := &Project{Focus: []string{"project-level"}}
	task := &Task{Focus: []string{"task-level"}}

	got := ResolveFocus(proj, task)
	if len(got) != 1 || got[0] != "task-level" {
		t.Fatalf("expected task focus to replace project focus, got: %v", got)
	}

	taskNoFocus := &Task{}
	got2 := ResolveFocus(proj, taskNoFocus)
	if len(got2) != 1 || got2[0] != "project-level" {
		t.Fatalf("expected project focus when task has none, got: %v", got2)
	}
}

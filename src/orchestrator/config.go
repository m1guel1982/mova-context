// Package orchestrator implements the multiagent architecture (spec
// section 7-9): a "group" is a directory under projects/ whose own
// config.json lists several AGENTS — each agent being an ORDINARY Mova
// project living at projects/<group>/<agent>/project.json, with its own
// memory, budget, focus, tasks, jobs, everything a top-level project has.
//
// No new storage concept was needed for this: core.Adapter's
// GetProject/ListProjects already resolve nested paths as project names
// (see core/file_adapter.go's ListProjects, which uses the path relative
// to projects/ as the name) — "projects/ventas_online/vendedor" is
// already addressable today as project "ventas_online/vendedor" by
// every existing command (`mova run`, `mova budget`, `mova chat`,
// `mova jobs run`...). This package only adds the ONE missing piece:
// reading the parent's config.json to know which agents belong to a
// group and running them — one, several, or all — through the exact
// same budget.BuildGatedContext every single-project run already uses
// (see run.go).
package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GroupConfig maps projects/<group>/config.json — the orchestrator file
// (spec section 7: "El archivo padre ... será el orquestador de todos
// los agentes").
type GroupConfig struct {
	Group       string   `json:"group"`       // display name, e.g. "ventas_online"
	Description string   `json:"description"` // optional
	Agents      []string `json:"agents"`      // subdirectory names, each with its own project.json
}

// ConfigPath returns projects/<group>/config.json under root.
func ConfigPath(root, group string) string {
	return filepath.Join(root, "projects", group, "config.json")
}

// LoadGroupConfig reads and parses a group's config.json. If "agents" is
// empty/omitted, every subdirectory of projects/<group>/ that itself
// contains a project.json is used instead (auto-discovery — same
// "never hardcode, detect automatically" principle core.ListProjects
// already follows), so a config.json with just {"group": "..."} still
// works.
func LoadGroupConfig(root, group string) (*GroupConfig, error) {
	path := ConfigPath(root, group)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("orchestrator config not found: %s", path)
	}
	var cfg GroupConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("%s invalid: %w", path, err)
	}
	if cfg.Group == "" {
		cfg.Group = group
	}
	if len(cfg.Agents) == 0 {
		cfg.Agents = discoverAgents(root, group)
	}
	return &cfg, nil
}

// discoverAgents lists every immediate subdirectory of projects/<group>/
// that contains its own project.json.
func discoverAgents(root, group string) []string {
	dir := filepath.Join(root, "projects", group)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var agents []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "project.json")); err == nil {
			agents = append(agents, e.Name())
		}
	}
	return agents
}

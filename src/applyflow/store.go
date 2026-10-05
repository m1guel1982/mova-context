package applyflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"mova.local/core"
)

// PendingTTL: una propuesta sin responder vence (el código pudo cambiar).
const PendingTTL = 60 * time.Minute

func projectDir(root, project string) string {
	return filepath.Dir(core.ProjectJSONPath(root, project))
}

func pendingPath(root, project string) string {
	return filepath.Join(projectDir(root, project), "pending-changes.json")
}

// Save deja la propuesta pendiente junto a project.json: así la confirmación
// puede llegar en OTRA llamada MCP/HTTP (que no comparte memoria de proceso).
func Save(root string, p *Proposal) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	path := pendingPath(root, p.Project)
	tmp := fmt.Sprintf("%s.tmp-%d", path, os.Getpid())
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(path)
		if err = os.Rename(tmp, path); err != nil {
			_ = os.Remove(tmp)
			return err
		}
	}
	return nil
}

// Load devuelve la propuesta pendiente (nil si no hay) y si estaba vencida
// (en ese caso también se borra).
func Load(root, project string) (p *Proposal, expired bool) {
	data, err := os.ReadFile(pendingPath(root, project))
	if err != nil {
		return nil, false
	}
	var q Proposal
	if json.Unmarshal(data, &q) != nil || len(q.Changes) == 0 {
		return nil, false
	}
	if t, err := time.Parse(time.RFC3339, q.Created); err == nil && time.Since(t) > PendingTTL {
		Clear(root, project)
		return nil, true
	}
	return &q, false
}

func Clear(root, project string) { _ = os.Remove(pendingPath(root, project)) }

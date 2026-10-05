// Package applyflow es el flujo ÚNICO "el modelo propone cambios → Mova
// pregunta → se modifican los archivos elegidos" que comparten Chat, MCP y
// HTTP (HTTP es MCP). Cada puerta solo decide CÓMO preguntar (terminal
// interactiva, o respuesta de la siguiente llamada); detectar, validar,
// mostrar, preguntar, respaldar y escribir vive aquí, una sola vez.
//
// Contrato de seguridad: nada se escribe sin una respuesta afirmativa
// explícita; todo se valida antes (dentro del repo, fuera de "exclude");
// todo archivo existente se respalda; un fragmento nunca pisa un archivo.
package applyflow

import (
	"fmt"
	"time"

	"mova.local/core"
	focusrender "mova.local/core/focus/render"
	"mova.local/documents"
	"mova.local/i18n"
	"mova.local/patcher"
)

// Change es un cambio propuesto (un bloque ```lang:ruta[::función()]```).
type Change struct {
	Index   int    `json:"index"`
	Lang    string `json:"lang"`
	Path    string `json:"path"`
	Symbol  string `json:"symbol,omitempty"`
	Content string `json:"content"`
	Action  string `json:"action"`         // create | modify
	Added   int    `json:"added"`          // líneas nuevas
	Removed int    `json:"removed"`        // líneas reemplazadas
	Skip    string `json:"skip,omitempty"` // motivo por el que NO se aplicará
}

// Proposal agrupa todos los cambios de UNA respuesta del modelo.
type Proposal struct {
	Project string   `json:"project"`
	Task    string   `json:"task"`
	Lang    string   `json:"lang"`
	Repo    string   `json:"repo"`
	Created string   `json:"created"`
	Backup  bool     `json:"backup"` // copiar cada archivo existente al lado antes de modificarlo
	Exclude []string `json:"exclude,omitempty"`
	Changes []Change `json:"changes"`
}

func (p *Proposal) options() patcher.Options {
	return patcher.Options{Excluded: func(path, sym string) (bool, string) {
		return core.ExcludedTarget(p.Exclude, path, sym, p.Lang)
	}}
}

func (c Change) block() documents.LabeledCodeBlock {
	return documents.LabeledCodeBlock{Lang: c.Lang, Path: c.Path, Symbol: c.Symbol, Content: c.Content}
}

// Build detecta los bloques etiquetados de reply y arma la propuesta.
// Devuelve nil si la tarea no tiene "apply" activo o no hay bloques: en ese
// caso Mova se comporta exactamente como antes.
func Build(root, project, task string, proj *core.Project, reply string) *Proposal {
	if !core.ApplyEnabled(proj, task) {
		return nil
	}
	return build(root, project, task, proj, reply, core.ApplyBackup(proj, task))
}

// BuildForced arma la propuesta aunque "apply" no esté activo: es el camino
// del Auto-Apply histórico (la persona escribe «sí» a una propuesta que el
// modelo ya hizo). Respalda siempre, el valor seguro por defecto.
func BuildForced(root, project, task string, proj *core.Project, reply string) *Proposal {
	backup := true
	if core.ApplyEnabled(proj, task) {
		backup = core.ApplyBackup(proj, task)
	}
	return build(root, project, task, proj, reply, backup)
}

func build(root, project, task string, proj *core.Project, reply string, backup bool) *Proposal {
	blocks := documents.ExtractLabeledCodeBlocks(reply)
	if len(blocks) == 0 || proj == nil {
		return nil
	}
	p := &Proposal{
		Project: project, Task: task, Lang: proj.Lang, Backup: backup,
		Repo:    focusrender.ResolveRepoPath(root, proj.Repo),
		Created: time.Now().UTC().Format(time.RFC3339),
		Exclude: core.ApplyExcludes(proj, task),
	}
	opt := p.options()
	for i, b := range blocks {
		plan := patcher.Preflight(p.Repo, b, opt)
		skip := plan.Skip
		switch skip {
		case "symbol-not-found":
			skip = i18n.TIn(p.Lang, "apply.skip_not_found", map[string]any{"symbol": b.Symbol})
		case "unbalanced":
			skip = i18n.TIn(p.Lang, "apply.skip_unbalanced")
		}
		p.Changes = append(p.Changes, Change{Index: i + 1, Lang: b.Lang, Path: b.Path, Symbol: b.Symbol,
			Content: b.Content, Action: plan.Action, Added: plan.Added, Removed: plan.Removed, Skip: skip})
	}
	return p
}

func (c Change) target() string {
	if c.Symbol != "" {
		return c.Path + "::" + c.Symbol + "()"
	}
	return c.Path
}

// Files: cuántos archivos distintos toca la propuesta.
func (p *Proposal) Files() int {
	seen := map[string]bool{}
	for _, c := range p.Changes {
		seen[c.Path] = true
	}
	return len(seen)
}

// Render arma la lista numerada y la PREGUNTA de confirmación, ya
// traducida. remote=false: pregunta interactiva de terminal; remote=true:
// instrucciones para responder en la siguiente llamada (MCP/HTTP).
func (p *Proposal) Render(remote bool) string {
	t := func(k string, a ...map[string]any) string { return i18n.TIn(p.Lang, k, a...) }
	out := t("apply.header", map[string]any{"changes": len(p.Changes), "files": p.Files()}) + "\n"
	for _, c := range p.Changes {
		action := "tools.file_changes.action_modify"
		if c.Action == "create" {
			action = "tools.file_changes.action_create"
		}
		out += t("apply.item", map[string]any{"n": c.Index, "action": t(action), "target": c.target(),
			"added": c.Added, "removed": c.Removed}) + "\n"
		if c.Skip != "" {
			out += t("apply.item_skip", map[string]any{"reason": c.Skip}) + "\n"
		}
	}
	if p.Backup {
		out += t("apply.backup_on") + "\n"
	} else {
		out += t("apply.backup_off") + "\n"
	}
	if remote {
		return out + "\n" + t("apply.question_remote", map[string]any{"count": len(p.Changes)})
	}
	return out + "\n" + t("apply.question_chat", map[string]any{"count": len(p.Changes)})
}

func (p *Proposal) String() string {
	return fmt.Sprintf("%s/%s: %d cambio(s)", p.Project, p.Task, len(p.Changes))
}

// apply_policy.go — cuándo una tarea puede modificar archivos y qué le
// está vedado. "apply" en project.json (bool) y en cada tarea (anula al
// proyecto): ausente/false = solo lectura, como siempre; true = el modelo
// recibe cómo proponer cambios y Mova PREGUNTA antes de escribir (ver
// mova.local/applyflow). Lo declarado en "exclude" nunca se modifica.
package core

import (
	"bytes"
	"encoding/json"
	"strings"

	"mova.local/i18n"
)

// ApplySetting es la política "apply" ya interpretada.
type ApplySetting struct {
	Set     bool // ¿el campo estaba declarado?
	Enabled bool
	Backup  bool
}

// parseApply interpreta el campo "apply": ausente = no declarado; bool =
// habilitar/deshabilitar (con respaldo, el valor seguro por defecto);
// objeto = {"enabled": bool, "backup": bool}. Ausente "enabled" en un
// objeto = true (declarar el objeto ya es querer aplicar); ausente
// "backup" = true.
func parseApply(raw json.RawMessage) ApplySetting {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return ApplySetting{}
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		return ApplySetting{Set: true, Enabled: b, Backup: true}
	}
	var o struct {
		Enabled *bool `json:"enabled"`
		Backup  *bool `json:"backup"`
	}
	if json.Unmarshal(raw, &o) != nil {
		return ApplySetting{}
	}
	st := ApplySetting{Set: true, Enabled: true, Backup: true}
	if o.Enabled != nil {
		st.Enabled = *o.Enabled
	}
	if o.Backup != nil {
		st.Backup = *o.Backup
	}
	return st
}

// ApplyFor resuelve la política de UNA tarea: la de la tarea si la declara,
// si no la del proyecto.
func ApplyFor(proj *Project, taskName string) ApplySetting {
	if proj == nil {
		return ApplySetting{}
	}
	if t, ok := proj.Tasks[taskName]; ok {
		if st := parseApply(t.Apply); st.Set {
			return st
		}
	}
	return parseApply(proj.Apply)
}

// ApplyEnabled: ¿alguna tarea en alcance puede modificar archivos?
func ApplyEnabled(proj *Project, taskName string) bool {
	if proj == nil {
		return false
	}
	for _, n := range tasksInScope(proj, taskName) {
		if ApplyFor(proj, n).Enabled {
			return true
		}
	}
	return false
}

// ApplyBackup: ¿se respalda cada archivo antes de modificarlo? Con varias
// tareas en alcance (modo «todas») basta que UNA con apply activo lo pida
// para respaldar; el valor por defecto es true (el seguro).
func ApplyBackup(proj *Project, taskName string) bool {
	any, backup := false, false
	for _, n := range tasksInScope(proj, taskName) {
		if st := ApplyFor(proj, n); st.Enabled {
			any = true
			backup = backup || st.Backup
		}
	}
	return !any || backup
}

// ApplyExcludes: unión de los "exclude" de las tareas en alcance.
func ApplyExcludes(proj *Project, taskName string) []string {
	var out []string
	for _, n := range tasksInScope(proj, taskName) {
		t := proj.Tasks[n]
		out = append(out, resolveTaskExclude(proj, &t)...)
	}
	return dedupe(out)
}

func normPath(p string) string {
	return strings.TrimPrefix(strings.ReplaceAll(strings.TrimSpace(p), `\`, "/"), "./")
}

// ExcludedTarget dice si un bloque (archivo + símbolo opcional) cae en
// "exclude" ("ruta" o "ruta::func=a,b") y devuelve el motivo ya traducido.
// Un archivo COMPLETO no se reescribe si alguna de sus funciones está
// excluida: se pide solo las funciones a cambiar.
func ExcludedTarget(patterns []string, path, symbol, lang string) (bool, string) {
	p := normPath(path)
	for _, pat := range patterns {
		file, spec, hasSpec := strings.Cut(pat, "::")
		f := normPath(file)
		if f == "" || !(p == f || strings.HasSuffix(p, "/"+f) || strings.HasSuffix(f, "/"+p)) {
			continue
		}
		if !hasSpec {
			return true, i18n.TIn(lang, "apply.excluded_file")
		}
		if i := strings.Index(spec, "="); i >= 0 {
			spec = spec[i+1:]
		}
		names := strings.Split(spec, ",")
		if symbol == "" {
			return true, i18n.TIn(lang, "apply.excluded_whole", map[string]any{"symbols": strings.Join(names, ", ")})
		}
		for _, n := range names {
			if strings.TrimSpace(n) == symbol {
				return true, i18n.TIn(lang, "apply.excluded_symbol", map[string]any{"symbol": symbol})
			}
		}
	}
	return false, ""
}

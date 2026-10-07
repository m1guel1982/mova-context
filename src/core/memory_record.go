// memory_record.go — registro de memoria tras una respuesta del LLM. UNA
// sola función para las tres puertas (chat, MCP y HTTP, que es MCP), para
// `mova run`/`mova memory` y para el flujo de Auto-Apply: mismo formato,
// misma deduplicación, misma regla del campo "memory" de project.json.
//
// Lo que se guarda es el bloque de síntesis ```memory del modelo (preciso y
// barato en tokens, no la respuesta completa). Si el modelo no lo entregó,
// un resumen automático con las líneas técnicas de la respuesta.
package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// MinMemoryReplyChars: respuestas más cortas ("ok", un sí/no) no son
// trabajo y no se registran (salvo Force).
const MinMemoryReplyChars = 200

// RecordOptions: Force ignora el campo "memory" y el largo mínimo (acción
// explícita del usuario: /memory, `mova memory`). Raw indica que el texto
// YA es la síntesis (no una respuesta completa del LLM): se guarda tal
// cual, sin resumen automático ni largo mínimo — lo usa MCP save_memory,
// con el que un LLM anfitrión (Cursor, Claude Desktop…) registra su propia
// síntesis cuando Mova no llama al modelo (modo delegado). Respeta el
// campo "memory" del proyecto.
type RecordOptions struct {
	Force, Raw bool
	// RunID: the evidence run whose model reply produced this entry
	// (observed by Mova). Empty with Source "host" means the entry was
	// reported by an MCP host (save_memory): Mova did not see the model.
	RunID  string
	Source string // "observed" (Mova called the model) | "host" (save_memory) | "user" (/memory, mova memory)
}

// RecordResult describe qué pasó.
type RecordResult struct {
	Saved   int    // entradas escritas
	Skipped int    // entradas ya existentes (idénticas)
	Off     bool   // "memory" ausente o false
	Short   bool   // respuesta demasiado corta
	Path    string // memory.md usado
	Tasks   []string
}

// Message: línea de estado para mostrar al usuario ("" = nada que decir).
func (r RecordResult) Message() string {
	switch {
	case r.Saved > 0:
		where := "memory.md"
		if r.Path != "" {
			where = r.Path
		}
		return fmt.Sprintf("[Memory] %d entrada(s) guardada(s) en %s (tarea: %s)", r.Saved, where, strings.Join(r.Tasks, ", "))
	case r.Skipped > 0:
		return "[Memory] sin cambios: la síntesis ya estaba registrada"
	}
	return ""
}

var (
	reMemBlocks = regexp.MustCompile("(?s)```memory\\s*(.*?)\\s*```")
	reMemTask   = regexp.MustCompile(`(?mi)^\*\*(?:tarea|task):\*\*\s*(.+?)\s*$`)
	reMemHead   = regexp.MustCompile(`^##\s+[^\n]*\n?`)
	reHRule     = regexp.MustCompile(`(?m)^\s*(?:---+|___+)\s*$`)
)

// ExtractMemoryBlocks devuelve TODOS los bloques ```memory de la respuesta.
func ExtractMemoryBlocks(response string) []string {
	var out []string
	for _, m := range reMemBlocks.FindAllStringSubmatch(response, -1) {
		if b := strings.TrimSpace(m[1]); b != "" {
			out = append(out, b)
		}
	}
	return out
}

func tagFor(task string) string {
	switch {
	case IsAllTasks(task):
		return "todas"
	case strings.TrimSpace(task) == "":
		return "general"
	}
	return strings.Join(strings.Fields(task), "_")
}

func shortSha(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:12]
}

// digestReply: resumen de respaldo cuando no hay bloque ```memory — las
// líneas con contenido técnico (títulos, archivo::función, `código`,
// viñetas con identificadores), acotado.
func digestReply(reply string, maxChars int) string {
	var b strings.Builder
	b.WriteString("**Resumen automático** (el modelo no entregó bloque de síntesis):\n")
	inFence := false
	for _, ln := range strings.Split(reply, "\n") {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "```") {
			inFence = !inFence
			continue
		}
		if inFence || t == "" {
			continue
		}
		tech := strings.HasPrefix(t, "#") || strings.Contains(t, "::") || strings.Contains(t, "`") ||
			((strings.HasPrefix(t, "-") || strings.HasPrefix(t, "*") || (len(t) > 2 && t[1] == '.')) && len(t) > 12)
		if !tech {
			continue
		}
		if b.Len()+len(t)+1 > maxChars {
			b.WriteString("…\n")
			break
		}
		b.WriteString(t + "\n")
	}
	return strings.TrimSpace(b.String())
}

// RecordMemory registra la síntesis de una respuesta del LLM en memory.md.
func RecordMemory(adapter Adapter, root, project, task, reply string, opt RecordOptions) (RecordResult, error) {
	var res RecordResult
	if adapter == nil || project == "" {
		res.Off = true
		return res, nil
	}
	proj, err := adapter.GetProject(project)
	if err != nil {
		return res, err
	}
	if !opt.Force && !MemoryEnabled(proj) {
		res.Off = true
		return res, nil
	}
	reply = strings.TrimSpace(reply)
	if reply == "" || (!opt.Force && !opt.Raw && len(reply) < MinMemoryReplyChars) {
		res.Short = true
		return res, nil
	}
	blocks := ExtractMemoryBlocks(reply)
	if len(blocks) == 0 && opt.Raw {
		blocks = []string{reply}
	}
	if len(blocks) == 0 {
		blocks = []string{digestReply(reply, 1800)}
	}
	if root != "" {
		res.Path = MemoryPath(root, project)
	}
	existing, _ := adapter.GetMemory(project)
	now := time.Now().Format("2006-01-02 15:04")
	for _, body := range blocks {
		tag := tagFor(task)
		if m := reMemTask.FindStringSubmatch(body); m != nil {
			tag = tagFor(m[1])
		}
		body = strings.TrimSpace(reMemHead.ReplaceAllString(body, "")) // el encabezado lo pone Mova
		body = reHRule.ReplaceAllString(body, "***")                   // "---" separa entradas en memory.md
		if body == "" {
			continue
		}
		sha := shortSha(tag + "\n" + body)
		if strings.Contains(existing, "sha="+sha+" ") {
			res.Skipped++
			continue
		}
		prov := "source=" + orDefault(opt.Source, "user")
		if opt.RunID != "" {
			prov += " run=" + opt.RunID
		}
		entry := fmt.Sprintf("## %s — tarea: %s\n<!-- mova:entry task=%s sha=%s %s -->\n%s", now, tag, tag, sha, prov, body)
		if err := adapter.AppendMemory(project, entry); err != nil {
			return res, err
		}
		existing = entry + memorySep + existing
		res.Saved++
		res.Tasks = append(res.Tasks, tag)
	}
	return res, nil
}

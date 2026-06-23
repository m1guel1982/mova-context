// mova — CLI para Mova Context
// Empaqueta agents + skills + prompts + memory.md en un único bloque
// listo para pegar en cualquier LLM web (Claude, GPT, Gemini, etc.)
//
// Uso:
//   mova run [proyecto] [tarea]
//   mova memory [proyecto] "texto de respuesta del LLM"
//   mova list
//   mova init [nombre]
//
// Sin dependencias externas. Compila con: go build -o mova .

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ── Estructuras del project.json ──────────────────────────────────────────────

type AgentSkillBlock struct {
	Base   []string `json:"base"`
	Custom []string `json:"custom"`
}

type PromptBlock struct {
	Base   string `json:"base"`
	Custom string `json:"custom"`
}

type Task struct {
	Prompt    PromptBlock       `json:"prompt"`
	Agents    AgentSkillBlock   `json:"agents"`
	Skills    AgentSkillBlock   `json:"skills"`
	Variables map[string]string `json:"variables"`
	Focus     []string          `json:"focus"`
}

type Project struct {
	Project     string            `json:"project"`
	Description string            `json:"description"`
	Repo        string            `json:"repo"`
	DefaultTask string            `json:"default_task"`
	Variables   map[string]string `json:"variables"`
	Focus       []string          `json:"focus"`
	Agents      AgentSkillBlock   `json:"agents"`
	Skills      AgentSkillBlock   `json:"skills"`
	Tasks       map[string]Task   `json:"tasks"`
}

// ── Rutas ─────────────────────────────────────────────────────────────────────

func movaRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "workflow.md")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no se encontró workflow.md — ejecuta mova desde dentro de mova-context")
		}
		dir = parent
	}
}

// ── Archivos ──────────────────────────────────────────────────────────────────

// readFile lee el archivo y garantiza que el resultado es UTF-8 válido.
// Si el archivo está en Latin-1 / CP1252 (tildes como bytes > 0x7F sueltos),
// convierte cada byte al rune equivalente — que es exactamente lo que hace
// la tabla Latin-1: codepoint == byte value.
func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if isValidUTF8(data) {
		return string(data)
	}
	// Convertir Latin-1/CP1252 → UTF-8: cada byte es su propio codepoint.
	runes := make([]rune, len(data))
	for i, b := range data {
		runes[i] = rune(b)
	}
	return string(runes)
}

// isValidUTF8 verifica si el slice de bytes es UTF-8 válido.
func isValidUTF8(data []byte) bool {
	for i := 0; i < len(data); {
		b := data[i]
		var size int
		switch {
		case b < 0x80:
			size = 1
		case b < 0xC2:
			return false // byte de continuación huérfano o sobrecodificación
		case b < 0xE0:
			size = 2
		case b < 0xF0:
			size = 3
		case b < 0xF5:
			size = 4
		default:
			return false
		}
		if i+size > len(data) {
			return false
		}
		for j := 1; j < size; j++ {
			if data[i+j]&0xC0 != 0x80 {
				return false
			}
		}
		i += size
	}
	return true
}

func walkFind(root, filename string) string {
	var result string
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || result != "" {
			return nil
		}
		if !info.IsDir() && info.Name() == filename {
			result = path
		}
		return nil
	})
	return result
}

func resolveFile(root, kind, name string) string {
	for _, subdir := range []string{"base", "custom"} {
		found := walkFind(filepath.Join(root, kind, subdir), name+".md")
		if found != "" {
			return readFile(found)
		}
	}
	return ""
}

// ── Variables ─────────────────────────────────────────────────────────────────

func mergeVars(global, task map[string]string) map[string]string {
	merged := make(map[string]string)
	for k, v := range global {
		merged[k] = v
	}
	for k, v := range task {
		merged[k] = v
	}
	return merged
}

func injectVars(text string, vars map[string]string) string {
	for k, v := range vars {
		text = strings.ReplaceAll(text, "{{"+strings.ToUpper(k)+"}}", v)
	}
	return text
}

// ── Construcción del contexto ─────────────────────────────────────────────────

func loadSection(root, kind string, block AgentSkillBlock, vars map[string]string, label string) string {
	var sb strings.Builder
	for _, name := range append(block.Base, block.Custom...) {
		content := resolveFile(root, kind, name)
		if content == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("\n\n<!-- %s: %s -->\n%s", label, name, injectVars(content, vars)))
	}
	return sb.String()
}

func loadPrompt(root string, prompt PromptBlock, vars map[string]string) string {
	var sb strings.Builder
	for _, name := range []string{prompt.Base, prompt.Custom} {
		if name == "" {
			continue
		}
		content := resolveFile(root, "prompts", name)
		if content == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("\n\n<!-- prompt: %s -->\n%s", name, injectVars(content, vars)))
	}
	return sb.String()
}

// ── Comandos ──────────────────────────────────────────────────────────────────

func cmdRun(root, projectName, taskName string) {
	data, err := os.ReadFile(filepath.Join(root, "projects", projectName, "project.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: no se encontró projects/%s/project.json\n", projectName)
		os.Exit(1)
	}

	var proj Project
	if err := json.Unmarshal(data, &proj); err != nil {
		fmt.Fprintf(os.Stderr, "Error leyendo project.json: %v\n", err)
		os.Exit(1)
	}

	if taskName == "" {
		taskName = proj.DefaultTask
	}
	task, ok := proj.Tasks[taskName]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: task '%s' no existe.\nTasks disponibles: %s\n",
			taskName, strings.Join(taskNames(proj), ", "))
		os.Exit(1)
	}

	vars := mergeVars(proj.Variables, task.Variables)
	vars["project"] = proj.Project
	vars["repo"] = proj.Repo
	vars["task"] = taskName

	focus := proj.Focus
	if len(task.Focus) > 0 {
		focus = task.Focus
	}
	if len(focus) > 0 {
		vars["focus"] = "- " + strings.Join(focus, "\n- ")
	}

	agents := AgentSkillBlock{
		Base:   append(proj.Agents.Base, task.Agents.Base...),
		Custom: append(proj.Agents.Custom, task.Agents.Custom...),
	}
	skills := AgentSkillBlock{
		Base:   append(proj.Skills.Base, task.Skills.Base...),
		Custom: append(proj.Skills.Custom, task.Skills.Custom...),
	}

	var out strings.Builder
	out.WriteString(fmt.Sprintf("# Contexto Mova Context — %s / %s\n", proj.Project, taskName))
	out.WriteString(fmt.Sprintf("Generado: %s\nRepo de trabajo: %s\n", time.Now().Format("2006-01-02 15:04"), proj.Repo))

	if len(focus) > 0 {
		out.WriteString("\n\n---\n## FOCUS (trabajar solo sobre esto)\n")
		for _, f := range focus {
			out.WriteString(fmt.Sprintf("- `%s`\n", f))
		}
		out.WriteString("\nSi un elemento no tiene ruta completa, buscarlo dentro de `" + proj.Repo + "`.\n")
		out.WriteString("Si aparece en más de un lugar, informar las coincidencias antes de continuar.\n")
	}

	out.WriteString("\n\n---\n## AGENTS\n")
	out.WriteString(loadSection(root, "agents", agents, vars, "agent"))
	out.WriteString("\n\n---\n## SKILLS\n")
	out.WriteString(loadSection(root, "skills", skills, vars, "skill"))
	out.WriteString("\n\n---\n## PROMPT\n")
	out.WriteString(loadPrompt(root, task.Prompt, vars))

	if mem := readFile(filepath.Join(root, "projects", projectName, "memory.md")); mem != "" {
		out.WriteString("\n\n---\n## MEMORIA (sesiones anteriores)\n")
		out.WriteString(mem)
	}

	out.WriteString("\n\n---\n## INSTRUCCIÓN\n")
	out.WriteString(fmt.Sprintf("Eres un asistente trabajando en el proyecto **%s**.\n", proj.Project))
	out.WriteString(fmt.Sprintf("Directorio de trabajo: `%s`\n", proj.Repo))
	out.WriteString("Aplica los agents, skills y prompt cargados arriba.\n")
	out.WriteString("Al finalizar, responde con un bloque:\n\n")
	out.WriteString("```memory\n## YYYY-MM-DD — sesión\n**Hecho:**\n**Resuelto:**\n**Pendiente:**\n**Decisiones:**\n**Errores LLM:**\n```\n")
	out.WriteString("\nEse bloque se usará para actualizar memory.md.\n")

	consolePrint(out.String())
}

func cmdMemory(root, projectName, response string) {
	start := strings.Index(response, "```memory")
	end := strings.LastIndex(response, "```")
	if start == -1 || end <= start {
		fmt.Fprintln(os.Stderr, "No se encontró bloque ```memory en la respuesta.")
		os.Exit(1)
	}
	block := strings.TrimSpace(response[start+len("```memory") : end])
	memPath := filepath.Join(root, "projects", projectName, "memory.md")
	updated := block + "\n\n---\n\n" + readFile(memPath)
	if err := os.WriteFile(memPath, []byte(updated), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error escribiendo memory.md: %v\n", err)
		os.Exit(1)
	}
	consolePrint("memory.md actualizado en " + memPath + "\n")
}

func cmdList(root string) {
	entries, err := os.ReadDir(filepath.Join(root, "projects"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "No se encontró el directorio projects/")
		os.Exit(1)
	}
	consolePrint("Proyectos disponibles:\n")
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, "projects", e.Name(), "project.json"))
		if err != nil {
			continue
		}
		var proj Project
		if err := json.Unmarshal(data, &proj); err != nil {
			continue
		}
		consolePrint(fmt.Sprintf("  %s — %s\n    Tasks: %s\n", e.Name(), proj.Description, strings.Join(taskNames(proj), ", ")))
	}
}

func cmdInit(root, name string) {
	dir := filepath.Join(root, "projects", name)
	os.MkdirAll(dir, 0755)
	template := `{
  "project": "` + name + `",
  "description": "",
  "repo": ".",
  "default_task": "",
  "variables": {},
  "agents": { "base": [], "custom": [] },
  "skills":  { "base": [], "custom": [] },
  "tasks": {}
}`
	os.WriteFile(filepath.Join(dir, "project.json"), []byte(template), 0644)
	consolePrint("Creado projects/" + name + "/project.json\n")
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func taskNames(proj Project) []string {
	names := make([]string, 0, len(proj.Tasks))
	for k := range proj.Tasks {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

func autoDetectProject(root string) string {
	entries, err := os.ReadDir(filepath.Join(root, "projects"))
	if err != nil {
		return ""
	}
	var valid []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, "projects", e.Name(), "project.json")); err == nil {
			valid = append(valid, e.Name())
		}
	}
	if len(valid) == 1 {
		return valid[0]
	}
	return ""
}

func usage() {
	consolePrint(`mova — Mova Context CLI

Uso:
  mova run [proyecto] [tarea]     Genera el contexto completo para pegar en un LLM
  mova memory [proyecto] "texto"  Actualiza memory.md con la respuesta del LLM
  mova list                       Lista proyectos y tareas disponibles
  mova init [nombre]              Crea un nuevo proyecto

Ejecuta mova desde dentro del repositorio mova-context (donde está workflow.md).
`)
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(0)
	}

	root, err := movaRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Uso: mova init [nombre-proyecto]")
			os.Exit(1)
		}
		cmdInit(root, os.Args[2])

	case "run":
		projectName, taskName := "", ""
		if len(os.Args) >= 3 {
			projectName = os.Args[2]
		}
		if len(os.Args) >= 4 {
			taskName = os.Args[3]
		}
		if projectName == "" {
			projectName = autoDetectProject(root)
			if projectName == "" {
				fmt.Fprintln(os.Stderr, "Especifica el proyecto: mova run [proyecto] [tarea]")
				cmdList(root)
				os.Exit(1)
			}
		}
		cmdRun(root, projectName, taskName)

	case "memory":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Uso: mova memory [proyecto] \"respuesta del LLM\"")
			os.Exit(1)
		}
		cmdMemory(root, os.Args[2], os.Args[3])

	case "list":
		cmdList(root)

	default:
		usage()
		os.Exit(1)
	}
}
// main.go — Mova Context CLI v3 (Unified Engine)
//
// Build: go build -o mova ./cli   (desde la raíz del repo, módulo "mova")

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mova.local/core"
	httptransport "mova.local/http"
	"mova.local/mcp"
	"mova.local/runtime"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	root, err := runtime.FindRoot()
	if err != nil {
		die(err.Error())
	}

	// El adaptador se genera tras leer la configuración del proyecto (vía project.json)
	// Para comandos globales o basales, interactúa usando el fileAdapter directo.
	getAdapter := func(projectName string) core.Adapter {
		if projectName == "" {
			return core.NewFileAdapter(root)
		}
		fa := core.NewFileAdapter(root)
		proj, _ := fa.GetProject(projectName)
		return newAdapter(root, proj)
	}

	switch os.Args[1] {

	case "run":
		project, task := arg(2, ""), arg(3, "")
		if project == "" {
			project = runtime.AutoDetect(root)
		}
		runProject(root, getAdapter(project), project, task)

	case "memory":
		project, response := needArg(2, "project"), needArg(3, "response")
		block, err := core.ExtractMemoryBlock(response)
		must(err)
		must(getAdapter(project).AppendMemory(project, block))
		consolePrint("memory updated: " + project + "\n")

	case "memory-read":
		project := needArg(2, "project")
		all := flagBool("--all")
		month := flagStr("--month", "")
		adapter := getAdapter(project)
		if month != "" {
			path := filepath.Join(root, "projects", project, "memory-archive", month+".md")
			data, err := os.ReadFile(path)
			must(err)
			consolePrint(string(data))
		} else if all {
			c, err := adapter.GetMemoryAll(project)
			must(err)
			consolePrint(c)
		} else {
			c, err := adapter.GetMemory(project)
			must(err)
			consolePrint(c)
		}

	case "memory-archive":
		project := needArg(2, "project")
		days := flagInt("--days", 30)
		must(getAdapter(project).ArchiveMemory(project, days))
		consolePrint(fmt.Sprintf("archived: %s (entries older than %d days)\n", project, days))

	case "list":
		projects, err := core.NewFileAdapter(root).ListProjects()
		must(err)
		for _, p := range projects {
			lang := p.Lang
			if lang == "" {
				lang = "legacy"
			}
			consolePrint(fmt.Sprintf("  %-22s [%s] %s\n    tasks: %s\n",
				p.Name, lang, p.Description, strings.Join(p.Tasks, ", ")))
		}

	case "init":
		name := needArg(2, "name")
		dir := filepath.Join(root, "projects", name)
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, "project.json"), []byte(projectTemplate(name)), 0644)
		os.WriteFile(filepath.Join(dir, "memory.md"), []byte(""), 0644)
		consolePrint("created: projects/" + name + "/\n")

	case "search":
		query := needArg(2, "query")
		domain := arg(3, "")
		results, err := core.NewFileAdapter(root).Search(query, domain)
		must(err)
		if len(results) == 0 {
			consolePrint("no results for: " + query + "\n")
			return
		}
		for _, r := range results {
			consolePrint(fmt.Sprintf("  [%s/%s/%s] %s\n  %s\n\n",
				r.Kind, r.Domain, r.Lang, r.Name, r.Excerpt))
		}

	case "mcp":
		if arg(2, "") != "start" {
			die("usage: mova mcp start [--port 3000] [--stdio]")
		}

		adapter := core.NewFileAdapter(root)

		// Flag --stdio determina si se levanta por Entrada/Salida estándar o por HTTP
		if flagBool("--stdio") {
			must(mcp.StartStdio(adapter, root))
		} else {
			port := flagInt("--port", 3000)
			must(httptransport.StartServer(adapter, root, port))
		}

	case "memory-clear":
		project := needArg(2, "project")
		runMemoryClear(getAdapter(project), root, project)

	case "memory-config":
		project := needArg(2, "project")
		action := needArg(3, "action (enable|disable|days|confirm|keep-memory-only)")
		value := arg(4, "")
		runMemoryConfig(root, project, action, value)

	// ── modelos locales (Ollama, LM Studio, vLLM...) ────────────────────
	// ver models_cmd.go / chat_cmd.go / mova.local/models

	case "config":
		provider := needArg(2, "provider (ollama, lmstudio, ...)")
		runConfigProvider(root, provider)

	case "show":
		if arg(2, "") != "config" {
			die("usage: mova show config [model]")
		}
		runShowConfig(root, arg(3, ""))

	case "install":
		runInstall(root, needArg(2, "models (comma-separated: llama3.1,mistral)"))

	case "model-list":
		runModelList(root)

	case "remove":
		runRemove(root, needArg(2, "models (comma-separated: llama3.1,mistral)"))

	case "chat":
		project, task := arg(2, ""), arg(3, "")
		runChat(root, project, task)

	case "budget":
		project, task := positionalArgs(2)
		if project == "" {
			project = runtime.AutoDetect(root)
		}
		runBudget(root, project, task, flagBool("--focus"))

	default:
		usage()
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func projectTemplate(name string) string {
	return `{
  "project": "` + name + `",
  "description": "",
  "repo": ".",
  "lang": "en",
  "adapter": "file",
  "llm": "claude",
  "default_task": "",
  "variables": {},
  "agents": { "domain": "software", "use": [] },
  "skills": { "domain": "software", "use": [] },
  "tasks": {}
}`
}

func arg(i int, def string) string {
	if i < len(os.Args) {
		return os.Args[i]
	}
	return def
}

// positionalArgs returns the first two non-flag arguments starting at
// os.Args[startIdx] — used by commands like `mova budget [project] [task]
// --focus` that combine positional args with boolean flags, so a flag
// occupying what would otherwise be a positional slot is never mistaken
// for it (e.g. "mova budget my-project --focus" must not read "--focus"
// as the task name).
func positionalArgs(startIdx int) (first, second string) {
	var found []string
	for i := startIdx; i < len(os.Args); i++ {
		if !strings.HasPrefix(os.Args[i], "--") {
			found = append(found, os.Args[i])
		}
	}
	if len(found) > 0 {
		first = found[0]
	}
	if len(found) > 1 {
		second = found[1]
	}
	return first, second
}

func needArg(i int, label string) string {
	if i < len(os.Args) {
		return os.Args[i]
	}
	die("missing argument: " + label)
	return ""
}

func flagBool(flag string) bool {
	for _, a := range os.Args {
		if a == flag {
			return true
		}
	}
	return false
}

func flagStr(flag, def string) string {
	for i, a := range os.Args {
		if a == flag && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return def
}

func flagInt(flag string, def int) int {
	s := flagStr(flag, "")
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func must(err error) {
	if err != nil {
		die(err.Error())
	}
}

func die(msg string) {
	fmt.Fprintln(os.Stderr, "error: "+msg)
	os.Exit(1)
}

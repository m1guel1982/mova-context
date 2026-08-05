// dispatch.go — the `switch os.Args[1]` command table, split out of
// main.go purely to keep that file under 300 lines. main() calls
// dispatch(root) exactly once; this is still the single place every
// command is routed from — no second dispatch table anywhere.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"mova.local/core"
	httptransport "mova.local/http"
	"mova.local/mcp"
	"mova.local/orchestrator"
	"mova.local/runtime"
)

func dispatch(root string) {
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
		project, task := positionalArgs(2)
		if project == "" {
			project = runtime.AutoDetect(root)
		}
		if flagBool("--count") {
			// See orchestrator/count.go: IsGroup decides which adapter
			// to use exactly the way `mova agents run` already does for
			// actually executing a group (plain file adapter — a group
			// has no project.json/adapter setting of its own), while an
			// ordinary project keeps using its own configured adapter
			// via getAdapter, unchanged from a normal `mova run`.
			adapter := getAdapter(project)
			if orchestrator.IsGroup(root, project) {
				adapter = core.NewFileAdapter(root)
			}
			runProjectCount(root, adapter, project, task, flagBool("--focus"))
		} else {
			runProject(root, getAdapter(project), project, task)
		}

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

	case "jobs":
		sub := needArg(2, "action (list|run|start)")
		switch sub {
		case "list":
			runJobsList(root, needArg(3, "project"))
		case "run":
			runJobsRun(root, needArg(3, "project"), arg(4, ""))
		case "start":
			runJobsStart(root)
		default:
			die("usage: mova jobs list <project> | mova jobs run <project> [index|--all] | mova jobs start")
		}

	case "agents":
		sub := needArg(2, "action (list|run)")
		group := needArg(3, "group")
		switch sub {
		case "list":
			runAgentsList(root, group)
		case "run":
			runAgentsRun(root, group, arg(4, ""))
		default:
			die("usage: mova agents list <group> | mova agents run <group> [agent|--all] (for token counts, use: mova run --count <group>)")
		}

	case "ui":
		runUI(root, arg(2, ""))

	default:
		usage()
	}
}

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

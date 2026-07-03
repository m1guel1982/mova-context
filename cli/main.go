// main.go — Mova Context CLI v3 (Unified Engine)
//
// Build: cd cli && go build -o mova .
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	var (
		root string
		err  error
	)

	// Si el usuario indicó explícitamente el proyecto (ideal para MCP),
	// úsalo directamente.
	if p := os.Getenv("MOVA_PROJECT_PATH"); p != "" {
		root = filepath.Clean(p)
	} else {
		root, err = findRoot()
		if err != nil {
			die(err.Error())
		}
	}
	 

	// El adaptador se genera tras leer la configuración del proyecto (vía project.json)
	// Para comandos globales o basales, interactúa usando el fileAdapter directo.
	getAdapter := func(projectName string) Adapter {
		if projectName == "" {
			return newFileAdapter(root)
		}
		fa := newFileAdapter(root)
		proj, _ := fa.GetProject(projectName)
		return newAdapter(root, proj)
	}
	switch os.Args[1] {

	case "run":
		project, task := arg(2, ""), arg(3, "")
		if project == "" {
			project = autoDetect(root)
		}
		ctx, err := resolveContext(getAdapter(project), root, project, task)
		must(err)
		consolePrint(ctx)

	case "compile":
		project, task := arg(2, ""), arg(3, "")
		if project == "" {
			project = autoDetect(root)
		}
		ctx, err := buildCompiledContext(getAdapter(project), root, project, task)
		must(err)
		outPath := filepath.Join(root, "projects", project, "contexto.txt")
		must(os.WriteFile(outPath, []byte(ctx), 0644))
		consolePrint("compiled: projects/" + project + "/contexto.txt\n")

	case "memory":
		project, response := needArg(2, "project"), needArg(3, "response")
		block, err := extractMemoryBlock(response)
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
		projects, err := newFileAdapter(root).ListProjects()
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
		results, err := newFileAdapter(root).Search(query, domain)
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

	adapter := newFileAdapter(root)

	// Claude Desktop / Cursor
	if flagBool("--stdio") {
		must(startMCPStdio(adapter, root))
		return
	}

	// HTTP (por defecto)
	port := flagInt("--port", 3000)
	must(startMCPHttp(adapter, root, port))

	case "memory-clear":
		project := needArg(2, "project")
		runMemoryClear(getAdapter(project), root, project)

	case "memory-config":
		project := needArg(2, "project")
		action := needArg(3, "action (enable|disable|days|confirm|keep-memory-only)")
		value := arg(4, "")
		runMemoryConfig(root, project, action, value)

	default:
		usage()
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func findRoot() (string, error) {
	// Lugares desde donde intentaremos buscar el proyecto.
	var candidates []string

	// 1. Variable de entorno (ideal para MCP)
	if root := os.Getenv("MOVA_PROJECT_ROOT"); root != "" {
		candidates = append(candidates, filepath.Clean(root))
	}

	// 2. Directorio actual (CLI)
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Clean(cwd))
	}

	// 3. Directorio del ejecutable
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(filepath.Clean(exe)))
	}

	visited := make(map[string]struct{})

	for _, start := range candidates {
		if start == "" {
			continue
		}

		dir := start

		for {
			if _, err := os.Stat(filepath.Join(dir, "workflow.md")); err == nil {
				return dir, nil
			}

			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}

			dir = parent
		}

		visited[start] = struct{}{}
	}

	return "", fmt.Errorf(
		"workflow.md not found.\n"+
			"Search locations:\n"+
			"  • MOVA_PROJECT_ROOT\n"+
			"  • Current working directory\n"+
			"  • Executable directory\n\n"+
			"Open Claude/Cursor inside the project or define MOVA_PROJECT_ROOT.",
	)
}

func autoDetect(root string) string {
	entries, _ := os.ReadDir(filepath.Join(root, "projects"))
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 1 {
		return names[0]
	}
	return ""
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

func arg(i int, def string) string {
	if i < len(os.Args) {
		return os.Args[i]
	}
	return def
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

func usage() {
	consolePrint(`mova — Mova Context v3

  mova run           [project] [task]        generate context for LLM
  mova compile       [project] [task]        distill + prune context → contexto.txt
  mova memory        [project] "response"    save session to memory.md
  mova memory-read   [project]               print active memory
    --all                                    include archives
    --month 2024-01                          specific archive month
  mova memory-archive [project]              archive old entries
    --days N                                 keep N days active (default 30)
  mova list                                  list all projects
  mova init          [name]                  create project
  mova search        "query" [domain]        search knowledge
  mova mcp start                             start MCP server
    --port 3000                              run as HTTP server (default)
    --stdio                                  run as Stdio server (for Claude/Cursor)
  mova memory-clear  [project]               delete ALL memory
    --archived                               delete only archived months
    --keep-active                            delete archives, keep memory.md
    --date 2024-06-15                        delete a specific day
    --from 2024-06-01 --to 2024-06-30        delete date range
    --yes                                    skip confirmation
  mova memory-config [project] [action] [value]
    enable | disable                         toggle auto-archive
    days N                                   set retention days (1, 10, 30, 90...)
    confirm true|false                       toggle confirmation on delete

  MOVA_ADAPTER=db  MOVA_DSN=postgres://... mova run project task
`)
}
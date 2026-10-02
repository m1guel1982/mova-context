// graph_hook.go — punto de enganche entre el motor de contexto y el
// generador de grafos (paquete graph). Va por inversión de dependencia
// porque graph necesita diagram, y diagram ya importa core: core no
// puede importar graph sin crear un ciclo. graph se registra a sí mismo
// (init) y el binario lo enlaza con un import en blanco (cli/main.go).
package core

import (
	"bytes"
	"encoding/json"
	"strings"
)

// DefaultGraphFile es el nombre cuando "graph": true.
const DefaultGraphFile = "graph.png"

// GraphJob es el grafo de UNA tarea que declara "graph": su focus/exclude
// ya resueltos (con la misma herencia proyecto→tarea que el contexto).
type GraphJob struct {
	Task, OutSpec  string
	Focus, Exclude []string
}

// GraphRequest agrupa los grafos de las tareas en alcance (una sola, o
// todas con TaskAll) que declaran "graph".
type GraphRequest struct {
	Root, ProjectName, ActiveTask, Lang, RepoPath string
	Jobs                                          []GraphJob
}

// GraphHook lo registra el paquete graph. Devuelve las líneas de estado
// ya traducidas (una por tarea, "" si no hay nada que decir). nil = no
// hay generador.
var GraphHook func(GraphRequest) string

// GraphAsync = true en los procesos de larga vida (chat, MCP, HTTP): el
// grafo se genera en segundo plano y GraphHook vuelve al instante, para
// que ni el arranque ni `exit` esperen un render. Con false (mova run,
// trace, budget...) se genera en línea y el comando termina con el archivo
// ya escrito.
var GraphAsync bool

// GraphNotify, si no es nil, recibe (desde otra goroutine) el estado de
// cada grafo que termina de generarse en segundo plano.
var GraphNotify func(msg string)

// GraphTarget interpreta Task.Graph: "archivo" / true -> (spec, true);
// ""/false/null/ausente/otro tipo -> ("", false).
func (t Task) GraphTarget() (string, bool) {
	raw := bytes.TrimSpace(t.Graph)
	if len(raw) == 0 {
		return "", false
	}
	var asBool bool
	if json.Unmarshal(raw, &asBool) == nil {
		if asBool {
			return DefaultGraphFile, true
		}
		return "", false
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if s = strings.TrimSpace(s); s != "" {
			return s, true
		}
	}
	return "", false
}

// graphJobs devuelve un job por cada tarea EN ALCANCE que declare
// "graph": con una tarea concreta solo el de esa tarea; con TaskAll el de
// todas (orden estable por nombre). Ya no se generan los grafos de tareas
// que no se pidieron.
func graphJobs(proj *Project, active string) []GraphJob {
	var jobs []GraphJob
	for _, n := range tasksInScope(proj, active) {
		t, ok := proj.Tasks[n]
		if !ok {
			continue
		}
		if spec, on := t.GraphTarget(); on {
			jobs = append(jobs, GraphJob{Task: n, OutSpec: spec, Focus: resolveTaskFocus(proj, &t), Exclude: resolveTaskExclude(proj, &t)})
		}
	}
	return jobs
}

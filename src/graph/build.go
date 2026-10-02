package graph

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"mova.local/core/focus"
	"mova.local/core/focus/astfilter"
	"mova.local/core/focus/resolvers"
	"mova.local/diagram"
)

// spec es el pedido de UN grafo (una tarea): lo que build necesita.
type spec struct {
	RepoPath, ProjectName, TaskName, Lang string
	Focus, Exclude                        []string
}

// selector es una entrada de focus/exclude ya interpretada para un archivo.
type selector struct {
	kind  string // vacío = archivo completo
	names map[string]bool
	style string // focus | excluded
}

type located struct {
	path string
	sels []selector
}

type fileInfo struct {
	abs, rel string
	facts    *astfilter.Facts
	decls    map[string]astfilter.Decl
}

type buildStats struct{ warnings []string }

var wholeFileKinds = map[string]bool{"func": true, "class": true, "struct": true, "interface": true, "trait": true}

// locate resuelve focus/exclude a archivos reales (mismo resolvedor que el
// contexto: ruta absoluta, relativa al repo, o nombre/ruta parcial) y
// conserva qué símbolos pidió cada entrada. Orden determinista.
func locate(req spec) []located {
	ctx := focus.Context{RepoPath: req.RepoPath, Index: &focus.FileIndex{}}
	idx := map[string]int{}
	var out []located
	add := func(entries []string, style string) {
		for _, e := range entries {
			file, kind, names, ok := resolvers.ParseSymbolTarget(e)
			if !ok {
				file, kind, names = e, "", nil
				if i := strings.Index(file, "::"); i >= 0 {
					file = file[:i]
				}
			}
			set := map[string]bool{}
			for _, n := range names {
				set[strings.TrimSuffix(strings.TrimSpace(n), "()")] = true
			}
			for _, p := range resolvers.LocateTarget(ctx, strings.TrimSpace(file)) {
				i, seen := idx[p]
				if !seen {
					i = len(out)
					idx[p] = i
					out = append(out, located{path: p})
				}
				out[i].sels = append(out[i].sels, selector{kind: kind, names: set, style: style})
			}
		}
	}
	add(req.Focus, "focus")
	add(req.Exclude, "excluded")
	return out
}

func slash(p string) string { return strings.ReplaceAll(p, `\`, "/") }

func stemOf(p string) string {
	p = slash(p)
	return strings.TrimSuffix(p, path.Ext(p))
}

type nodeInfo struct {
	id, name string
	file     *fileInfo
	start    uint32
	style    string
}

func nodeKind(k string) string {
	switch k {
	case "func":
		return "func"
	case "const", "var":
		return "var"
	case "class", "struct", "interface", "trait", "enum", "impl", "namespace", "type":
		return "class"
	}
	return "other"
}

func build(req spec, files []located) (*diagram.GraphData, buildStats) {
	var st buildStats
	var infos []*fileInfo
	for _, lf := range files {
		data, err := os.ReadFile(lf.path)
		if err != nil {
			continue
		}
		facts, ok := astfilter.Analyze(data, lf.path)
		if !ok {
			continue
		}
		rel := slash(lf.path)
		if r, err := filepath.Rel(req.RepoPath, lf.path); err == nil && !strings.HasPrefix(r, "..") {
			rel = slash(r)
		}
		fi := &fileInfo{abs: lf.path, rel: rel, facts: facts, decls: map[string]astfilter.Decl{}}
		for _, d := range facts.Decls {
			if _, dup := fi.decls[d.Name]; !dup {
				fi.decls[d.Name] = d
			}
		}
		infos = append(infos, fi)
	}

	// ---- nodos seleccionados -------------------------------------------------
	nodes := map[string]*nodeInfo{}
	var order []*nodeInfo
	byName := map[string][]*nodeInfo{}
	kinds := map[string]string{}
	selOf := map[string][]selector{}
	for _, lf := range files {
		selOf[lf.path] = lf.sels
	}
	newNode := func(fi *fileInfo, d astfilter.Decl, style string) *nodeInfo {
		id := fi.rel + "#" + d.Name
		if n, ok := nodes[id]; ok {
			return n
		}
		n := &nodeInfo{id: id, name: d.Name, file: fi, start: d.Start, style: style}
		nodes[id] = n
		order = append(order, n)
		kinds[id] = d.Kind
		return n
	}
	for _, fi := range infos {
		found := map[string]bool{}
		for _, d := range fi.facts.Decls {
			style := ""
			for _, s := range selOf[fi.abs] {
				if !((s.kind == "" && wholeFileKinds[d.Kind]) || (s.kind == d.Kind && s.names[d.Name])) {
					continue
				}
				found[s.kind+":"+d.Name] = true
				if style == "" || s.style == "excluded" {
					style = s.style
				}
			}
			if style == "" {
				continue
			}
			if _, dup := nodes[fi.rel+"#"+d.Name]; dup {
				continue
			}
			n := newNode(fi, d, style)
			byName[d.Name] = append(byName[d.Name], n)
		}
		var missing []string
		for _, s := range selOf[fi.abs] {
			for nm := range s.names {
				if !found[s.kind+":"+nm] {
					missing = append(missing, nm)
				}
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			st.warnings = append(st.warnings, tr(req.Lang, "graph.warn_missing", map[string]any{"task": req.TaskName, "file": fi.rel, "names": strings.Join(dedupe(missing), ", ")}))
		}
	}

	// ---- resolución de módulos ----------------------------------------------
	stems := map[string]*fileInfo{}
	for _, fi := range infos {
		s := stemOf(fi.abs)
		stems[s] = fi
		if strings.HasSuffix(s, "/index") {
			stems[strings.TrimSuffix(s, "/index")] = fi
		}
	}
	resolveModule := func(from *fileInfo, module string) *fileInfo {
		m := slash(strings.TrimSpace(module))
		if strings.HasPrefix(m, ".") && !strings.HasPrefix(m, "./") && !strings.HasPrefix(m, "../") { // python ".x"
			return nil
		}
		if strings.HasPrefix(m, ".") {
			return stems[stemOf(path.Join(path.Dir(slash(from.abs)), m))]
		}
		if !strings.Contains(m, "/") {
			m = strings.ReplaceAll(m, ".", "/") // python dotted / java
		}
		m = stemOf(m)
		var hit *fileInfo
		for _, fi := range infos {
			if s := stemOf(fi.abs); s == m || strings.HasSuffix(s, "/"+m) {
				if hit != nil && hit != fi {
					return nil // ambiguo: no adivinar
				}
				hit = fi
			}
		}
		return hit
	}

	// ---- aristas -------------------------------------------------------------
	type ekey struct{ f, t, k string }
	edgeSet := map[ekey]bool{}
	var edges []diagram.GraphEdge
	addEdge := func(from, to, kind string, inferred bool) {
		if from == to || edgeSet[ekey{from, to, kind}] {
			return
		}
		edgeSet[ekey{from, to, kind}] = true
		edges = append(edges, diagram.GraphEdge{From: from, To: to, Kind: kind, Inferred: inferred})
	}
	target := func(fi *fileInfo, name string) (string, bool) {
		d, ok := fi.decls[name]
		if !ok {
			return "", false
		}
		return newNode(fi, d, "outside").id, true
	}
	importsOf := func(fi *fileInfo, local string) []astfilter.Import {
		var out []astfilter.Import
		for _, im := range fi.facts.Imports {
			if im.Local == local {
				out = append(out, im)
			}
		}
		return out
	}
	lastSeg := func(q string) string {
		for _, sep := range []string{"::", "->", "."} {
			if i := strings.LastIndex(q, sep); i >= 0 {
				q = q[i+len(sep):]
			}
		}
		return q
	}
	namedMember := func(im astfilter.Import) bool { return im.Member != "" && im.Member != "*" && im.Member != "default" }
	selfQ := map[string]bool{"this": true, "self": true, "super": true, "cls": true, "$this": true}

	selected := append([]*nodeInfo(nil), order...)
	for _, n := range selected {
		fi := n.file
		for _, c := range fi.facts.Calls {
			if c.Caller != n.name {
				continue
			}
			var to string
			var ok bool
			switch {
			case c.Qualifier == "":
				for _, im := range importsOf(fi, c.Name) {
					if g := resolveModule(fi, im.Module); namedMember(im) && g != nil {
						if to, ok = target(g, im.Member); ok {
							break
						}
					}
				}
				if !ok {
					to, ok = target(fi, c.Name)
				}
			case selfQ[c.Qualifier]:
				to, ok = target(fi, c.Name)
			case c.Qualifier != "?":
				q := lastSeg(c.Qualifier)
				for _, im := range importsOf(fi, q) {
					if g := resolveModule(fi, im.Module); g != nil {
						if to, ok = target(g, c.Name); ok {
							break
						}
					}
				}
				if _, isLocal := fi.decls[q]; !ok && isLocal {
					to, ok = target(fi, c.Name)
				}
			}
			inferred := false
			if !ok { // último recurso: nombre único entre los símbolos de focus/exclude
				if cands := byName[c.Name]; len(cands) == 1 {
					to, ok, inferred = cands[0].id, true, true
				}
			}
			if ok {
				addEdge(n.id, to, "call", inferred)
			}
		}
		isVar := func(d astfilter.Decl) bool { return d.Kind == "var" || d.Kind == "const" }
		for _, r := range fi.facts.Refs {
			if r.Caller != n.name {
				continue
			}
			if len(importsOf(fi, r.Name)) > 0 { // `const X = require(..)`: es un import, no una variable
				continue
			}
			if d, ok := fi.decls[r.Name]; ok && isVar(d) {
				to, _ := target(fi, r.Name)
				addEdge(n.id, to, "ref", false)
				continue
			}
			for _, im := range importsOf(fi, r.Name) {
				if g := resolveModule(fi, im.Module); namedMember(im) && g != nil {
					if d, ok := g.decls[im.Member]; ok && isVar(d) {
						to, _ := target(g, im.Member)
						addEdge(n.id, to, "ref", false)
					}
				}
			}
		}
	}

	// ---- clusters, nodos finales e importaciones ------------------------------
	sort.SliceStable(order, func(i, j int) bool {
		a, b := order[i], order[j]
		if a.file != b.file {
			return false
		}
		if (a.style == "outside") != (b.style == "outside") {
			return b.style == "outside"
		}
		return a.start < b.start
	})
	g := &diagram.GraphData{}
	inCluster := map[*fileInfo]bool{}
	for _, n := range order {
		inCluster[n.file] = true
	}
	for _, fi := range infos {
		if inCluster[fi] {
			g.Clusters = append(g.Clusters, diagram.GraphCluster{ID: fi.rel, Label: fi.rel})
		}
	}
	counts := map[string]int{}
	for _, n := range order {
		g.Nodes = append(g.Nodes, diagram.GraphNode{ID: n.id, Cluster: n.file.rel, Label: labelOf(n.name, kinds[n.id]), Kind: nodeKind(kinds[n.id]), Style: n.style})
		counts[n.style]++
	}
	for _, fi := range infos {
		if !inCluster[fi] {
			continue
		}
		for _, im := range fi.facts.Imports {
			if gf := resolveModule(fi, im.Module); gf != nil && gf != fi && inCluster[gf] {
				addEdge(fi.rel, gf.rel, "import", false)
			}
		}
	}
	g.Edges = edges

	g.Labels = diagram.GraphLabels{
		Title: tr(req.Lang, "graph.title", map[string]any{"project": req.ProjectName}),
		Subtitle: []string{
			tr(req.Lang, "graph.subtitle_task", map[string]any{"task": req.TaskName, "repo": req.RepoPath}),
			tr(req.Lang, "graph.subtitle_counts", map[string]any{"files": len(g.Clusters), "focus": counts["focus"], "excluded": counts["excluded"], "outside": counts["outside"], "edges": len(g.Edges)}),
		},
		Legend: tr(req.Lang, "graph.legend"), File: tr(req.Lang, "graph.file"), Func: tr(req.Lang, "graph.func"), Class: tr(req.Lang, "graph.class"),
		Var: tr(req.Lang, "graph.var"), Outside: tr(req.Lang, "graph.outside"), Excluded: tr(req.Lang, "graph.excluded"),
		Call: tr(req.Lang, "graph.call"), Ref: tr(req.Lang, "graph.ref"), Import: tr(req.Lang, "graph.import"), Inferred: tr(req.Lang, "graph.inferred"),
	}
	return g, st
}

func labelOf(name, kind string) string {
	if kind == "func" {
		return name + "()"
	}
	return name
}

func dedupe(in []string) []string {
	var out []string
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}

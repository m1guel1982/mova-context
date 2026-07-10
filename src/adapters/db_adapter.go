// db_adapter.go — reads knowledge from a relational or document database.
// Same core.Adapter interface as core.FileAdapter. The engine never knows
// the difference.
//
// Supported now:   postgres, mongodb (stub)
// Prepared for:    mysql, sqlserver, sqlite, redis, cosmos, dynamo
// To add a new DB: implement core.Adapter, add a case in NewDBAdapter().
//
// package adapters (Open Source, dependencia externa aislada aquí a
// propósito: core/ se mantiene con cero dependencias externas — ver
// mova-context-v2-propuesta-arquitectura.md, mapeo de la sección 3).
package adapters

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq" // postgres driver

	"mova.local/core"
)

type dbAdapter struct {
	db     *sql.DB
	driver string
}

func NewDBAdapter(dsn string) (core.Adapter, error) {
	driver := detectDBDriver(dsn)
	if driver == "mongodb" {
		// MongoDB stub — replace with mongo-driver when needed
		return nil, fmt.Errorf("mongodb adapter: import go.mongodb.org/mongo-driver and implement")
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(3)
	return &dbAdapter{db: db, driver: driver}, nil
}

func (a *dbAdapter) Close() error { return a.db.Close() }

// GetKnowledge resolves with priority: project-specific+lang > global+lang > fallback
func (a *dbAdapter) GetKnowledge(kind, domain, lang, name string) (string, error) {
	var content string
	err := a.db.QueryRow(`
		SELECT content FROM knowledge
		WHERE kind=$1 AND domain=$2 AND name=$3
		  AND (lang=$4 OR lang='') AND active=true
		ORDER BY
		  CASE WHEN lang=$4 THEN 0 ELSE 1 END,
		  CASE WHEN is_custom THEN 0 ELSE 1 END
		LIMIT 1`,
		kind, domain, name, lang,
	).Scan(&content)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("%s %q not found in domain %q", kind, name, domain)
	}
	return content, err
}

func (a *dbAdapter) GetProject(name string) (*core.Project, error) {
	var p core.Project
	var vars, tasks string
	err := a.db.QueryRow(`
		SELECT name, description, repo, lang, adapter, dsn, llm,
		       default_task, variables::text, tasks::text
		FROM projects WHERE name=$1 AND active=true`, name).
		Scan(&p.Project, &p.Description, &p.Repo, &p.Lang,
			&p.Adapter, &p.DSN, &p.LLM, &p.DefaultTask, &vars, &tasks)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project %q not found", name)
	}
	if err != nil {
		return nil, err
	}
	jsonUnmarshal(vars, &p.Variables)
	jsonUnmarshal(tasks, &p.Tasks)
	return &p, nil
}

func (a *dbAdapter) ListProjects() ([]core.ProjectSummary, error) {
	rows, err := a.db.Query(`
		SELECT name, description, lang, tasks::text
		FROM projects WHERE active=true ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []core.ProjectSummary
	for rows.Next() {
		var s core.ProjectSummary
		var tasksJSON string
		if err := rows.Scan(&s.Name, &s.Description, &s.Lang, &tasksJSON); err != nil {
			continue
		}
		var tasks map[string]core.Task
		if jsonUnmarshal(tasksJSON, &tasks) == nil {
			for k := range tasks {
				s.Tasks = append(s.Tasks, k)
			}
			sort.Strings(s.Tasks)
		}
		out = append(out, s)
	}
	return out, nil
}

func (a *dbAdapter) GetMemory(project string) (string, error) {
	rows, err := a.db.Query(`
		SELECT content FROM memory
		WHERE project=$1 AND archived=false
		ORDER BY created_at DESC LIMIT 50`, project)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var parts []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err == nil {
			parts = append(parts, c)
		}
	}
	return strings.Join(parts, "\n\n---\n\n"), nil
}

func (a *dbAdapter) GetMemoryAll(project string) (string, error) {
	rows, err := a.db.Query(`
		SELECT content, archived FROM memory
		WHERE project=$1
		ORDER BY session_date DESC, created_at DESC`, project)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var parts []string
	for rows.Next() {
		var c string
		var arch bool
		if err := rows.Scan(&c, &arch); err == nil {
			if arch {
				c = "<!-- archived -->\n" + c
			}
			parts = append(parts, c)
		}
	}
	return strings.Join(parts, "\n\n---\n\n"), nil
}

func (a *dbAdapter) AppendMemory(project, entry string) error {
	_, err := a.db.Exec(`
		INSERT INTO memory (project, content, session_date)
		VALUES ($1, $2, $3)`,
		project, strings.TrimSpace(entry), time.Now().Format("2006-01-02"))
	return err
}

func (a *dbAdapter) ArchiveMemory(project string, keepDays int) error {
	cutoff := time.Now().AddDate(0, 0, -keepDays).Format("2006-01-02")
	_, err := a.db.Exec(`
		UPDATE memory SET archived=true
		WHERE project=$1 AND session_date < $2 AND archived=false`,
		project, cutoff)
	return err
}

func (a *dbAdapter) Search(query, domain string) ([]core.SearchResult, error) {
	if a.driver == "postgres" {
		return a.searchPostgres(query, domain)
	}
	return a.searchLike(query, domain)
}

func (a *dbAdapter) searchPostgres(query, domain string) ([]core.SearchResult, error) {
	rows, err := a.db.Query(`
		SELECT kind, domain, lang, name,
		  ts_headline('english', content, plainto_tsquery($1), 'MaxWords=15,MinWords=8'),
		  ts_rank(fts, plainto_tsquery($1))
		FROM knowledge
		WHERE fts @@ plainto_tsquery($1)
		  AND ($2='' OR domain=$2) AND active=true
		ORDER BY 6 DESC LIMIT 20`, query, domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSearchResults(rows)
}

func (a *dbAdapter) searchLike(query, domain string) ([]core.SearchResult, error) {
	p := "%" + strings.ToLower(query) + "%"
	rows, err := a.db.Query(`
		SELECT kind, domain, lang, name, SUBSTR(content,1,150), 0.5
		FROM knowledge
		WHERE (LOWER(name) LIKE ? OR LOWER(content) LIKE ?)
		  AND (? ='' OR domain=?) AND active=1
		ORDER BY name LIMIT 20`, p, p, domain, domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSearchResults(rows)
}

func scanSearchResults(rows *sql.Rows) ([]core.SearchResult, error) {
	var out []core.SearchResult
	for rows.Next() {
		var r core.SearchResult
		if err := rows.Scan(&r.Kind, &r.Domain, &r.Lang, &r.Name, &r.Excerpt, &r.Score); err == nil {
			out = append(out, r)
		}
	}
	return out, rows.Err()
}

// jsonUnmarshal is a placeholder — in production import "encoding/json".
func jsonUnmarshal(data string, v any) error {
	if data == "" {
		return nil
	}
	_ = v
	return nil
}

// DeleteMemory removes memory entries from the database matching the request.
func (a *dbAdapter) DeleteMemory(project string, req core.MemoryDeleteRequest) (int, error) {
	var res interface{ RowsAffected() (int64, error) }
	var err error

	switch {
	case req.All:
		r, e := a.db.Exec(`DELETE FROM memory WHERE project=$1`, project)
		res, err = r, e

	case req.Archived || req.KeepActive:
		r, e := a.db.Exec(`DELETE FROM memory WHERE project=$1 AND archived=true`, project)
		res, err = r, e

	case req.Date != "":
		r, e := a.db.Exec(`DELETE FROM memory WHERE project=$1 AND session_date=$2`, project, req.Date)
		res, err = r, e

	case req.From != "" && req.To != "":
		r, e := a.db.Exec(
			`DELETE FROM memory WHERE project=$1 AND session_date BETWEEN $2 AND $3`,
			project, req.From, req.To)
		res, err = r, e

	default:
		return 0, fmt.Errorf("no delete criteria specified")
	}

	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// detectDBDriver returns the SQL driver name inferred from the DSN prefix.
func detectDBDriver(dsn string) string {
	switch {
	case strings.HasPrefix(dsn, "postgres"):
		return "postgres"
	case strings.HasPrefix(dsn, "mongodb"):
		return "mongodb"
	case strings.HasSuffix(dsn, ".db") || strings.HasSuffix(dsn, ".sqlite"):
		return "sqlite3"
	default:
		return "sqlite3"
	}
}

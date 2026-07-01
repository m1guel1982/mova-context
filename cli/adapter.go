// adapter.go — storage abstraction.
// The engine never knows if data comes from files or a database.
// To add a new database: implement Adapter and register in newAdapter().
package main

import (
	"fmt"
	"os"
	"strings"
)

// Adapter is the single storage contract.
// FileAdapter and DBAdapter both implement this.
type Adapter interface {
	GetKnowledge(kind, domain, lang, name string) (string, error)
	GetProject(name string) (*Project, error)
	ListProjects() ([]ProjectSummary, error)
	GetMemory(project string) (string, error)
	GetMemoryAll(project string) (string, error)
	AppendMemory(project, entry string) error
	ArchiveMemory(project string, keepDays int) error
	DeleteMemory(project string, req MemoryDeleteRequest) (int, error) // returns count deleted
	Search(query, domain string) ([]SearchResult, error)
}

// newAdapter creates the right adapter from project config or environment.
// Priority: project.json > MOVA_ADAPTER env > file (default).
func newAdapter(root string, proj *Project) Adapter {
	adapterType := "file"
	dsn := ""

	if proj != nil && proj.Adapter != "" {
		adapterType = proj.Adapter
		dsn = proj.DSN
	}

	// Environment variables override project.json
	if env := os.Getenv("MOVA_ADAPTER"); env != "" {
		adapterType = env
	}
	if env := os.Getenv("MOVA_DSN"); env != "" {
		dsn = env
	}

	switch adapterType {
	case "db":
		if dsn == "" {
			fmt.Fprintln(os.Stderr, "warning: adapter=db but no dsn set, falling back to file")
			return newFileAdapter(root)
		}
		db, err := newDBAdapter(dsn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: db connect failed (%v), falling back to file\n", err)
			return newFileAdapter(root)
		}
		return db
	default:
		return newFileAdapter(root)
	}
}

// adapterType returns "file" or "db" from dsn prefix.
func detectDriver(dsn string) string {
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

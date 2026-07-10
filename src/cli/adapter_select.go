// adapter_select.go — decide QUÉ Adapter instanciar (file vs db) según
// project.json / variables de entorno. Antes vivía dentro de core/adapter.go
// como newAdapter(); se movió aquí porque elegir el backend de storage es
// una decisión de aplicación (CLI/HTTP/MCP), no del motor — core solo
// conoce la interfaz Adapter, nunca decide cuál instanciar. Este es también
// el único archivo del binario que importa tanto "mova.local/core" como
// "mova.local/adapters", precisamente para que core no tenga que importar
// adapters (evita un ciclo: adapters ya importa core para los tipos
// compartidos).

package main

import (
	"fmt"
	"os"
	"mova.local/adapters"
	"mova.local/core"
)

// newAdapter creates the right adapter from project config or environment.
// Priority: project.json > MOVA_ADAPTER env > file (default).
func newAdapter(root string, proj *core.Project) core.Adapter {
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
			return core.NewFileAdapter(root)
		}
		db, err := adapters.NewDBAdapter(dsn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: db connect failed (%v), falling back to file\n", err)
			return core.NewFileAdapter(root)
		}
		return db
	default:
		return core.NewFileAdapter(root)
	}
}

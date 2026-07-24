// models_cmd.go — comandos de gestión de modelos locales:
//
//	mova config       <provider>              elige el proveedor activo
//	mova show config  [modelo]                muestra proveedor activo o un modelo
//	mova install      modelo1,modelo2,...     descarga modelos (con barra de progreso)
//	mova model-list                           modelos instalados en el proveedor activo
//	mova remove       modelo1,modelo2,...     elimina modelos del proveedor activo
//
// Todo esto vive en config/models/ — ver docs/i18n/es/COMMANDS.md.
package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"mova.local/models"
)

// runConfigProvider — mova config <provider>
func runConfigProvider(root, provider string) {
	must(models.SetActiveProvider(root, provider))
	consolePrint("proveedor activo: " + provider + "\n")
}

// runShowConfig — mova show config [modelo]
func runShowConfig(root, model string) {
	state, err := models.GetActiveState(root)
	must(err)
	if state.Provider == "" {
		die("no hay proveedor activo — corré `mova config <provider>` primero (p.ej. `mova config ollama`)")
	}

	name := model
	if name == "" {
		name = state.Config
	}
	if name == "" {
		names, err := models.ListModelConfigs(root, state.Provider)
		must(err)
		consolePrint("proveedor activo: " + state.Provider + " (sin modelo elegido todavía)\n")
		if len(names) == 0 {
			consolePrint("no hay archivos de configuración en config/models/" + state.Provider + " — creá uno (ver docs/i18n/es/COMMANDS.md)\n")
		} else {
			consolePrint("modelos configurados: " + strings.Join(names, ", ") + "\n")
		}
		return
	}

	resolved, err := models.FindModelFile(root, state.Provider, name)
	must(err)
	mc, err := models.DefaultCache.GetModel(root, state.Provider, resolved)
	must(err)
	consolePrint(fmt.Sprintf("%s/%s:\n\n", state.Provider, resolved))
	printJSON(mc)
}

// runInstall — mova install modelo1,modelo2,...
func runInstall(root, csv string) {
	state, err := models.GetActiveState(root)
	must(err)
	if state.Provider == "" {
		die("no hay proveedor activo — corré `mova config <provider>` primero")
	}
	names := splitCSV(csv)
	if len(names) == 0 {
		die("indicá al menos un modelo: mova install llama3.1,mistral")
	}

	bars := map[string]*progressBar{}
	err = models.Install(root, state.Provider, names, func(model, status string, percent int) {
		bar, ok := bars[model]
		if !ok {
			bar = newProgressBar(model)
			bars[model] = bar
		}
		bar.update(status, percent)
	})
	for _, bar := range bars {
		bar.finish()
	}
	must(err)
	consolePrint("instalación completa: " + strings.Join(names, ", ") + "\n")
}

// runModelList — mova model-list
func runModelList(root string) {
	state, err := models.GetActiveState(root)
	must(err)
	if state.Provider == "" {
		die("no hay proveedor activo — corré `mova config <provider>` primero")
	}
	list, err := models.ListInstalled(root, state.Provider)
	must(err)
	if len(list) == 0 {
		consolePrint("no hay modelos instalados en " + state.Provider + "\n")
		return
	}
	for _, m := range list {
		mark := "  "
		if m.Name == state.Config {
			mark = "* "
		}
		consolePrint(fmt.Sprintf("%s%-24s %8.1f GB   %s\n", mark, m.Name, float64(m.Size)/1e9, m.ModifiedAt))
	}
}

// runRemove — mova remove modelo1,modelo2,...
func runRemove(root, csv string) {
	state, err := models.GetActiveState(root)
	must(err)
	if state.Provider == "" {
		die("no hay proveedor activo — corré `mova config <provider>` primero")
	}
	names := splitCSV(csv)
	if len(names) == 0 {
		die("indicá al menos un modelo: mova remove llama3.1,mistral")
	}
	must(models.Remove(root, state.Provider, names))
	consolePrint("eliminado: " + strings.Join(names, ", ") + "\n")
}

// ── helpers ───────────────────────────────────────────────────────────────

func splitCSV(csv string) []string {
	var out []string
	for _, part := range strings.Split(csv, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func printJSON(v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	consolePrint(string(data) + "\n")
}

// progressBar — barra de progreso simple para `mova install`, una línea
// por modelo, actualizada in-place con retorno de carro (\r).
type progressBar struct {
	model string
	done  bool
}

func newProgressBar(model string) *progressBar {
	return &progressBar{model: model}
}

func (b *progressBar) update(status string, percent int) {
	if b.done {
		return
	}
	if percent < 0 {
		consolePrint(fmt.Sprintf("\r%-16s %-22s", b.model, status))
		return
	}
	filled := percent / 5 // barra de 20 caracteres
	bar := strings.Repeat("=", filled) + strings.Repeat("-", 20-filled)
	consolePrint(fmt.Sprintf("\r%-16s [%s] %3d%%  %s", b.model, bar, percent, status))
	if status == "success" {
		b.done = true
		consolePrint("\n")
	}
}

func (b *progressBar) finish() {
	if !b.done {
		consolePrint("\n")
	}
}

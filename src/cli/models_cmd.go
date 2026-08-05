// models_cmd.go — local model management commands:
//
//   mova config       <provider>             selects active provider
//   mova show config  [model]                shows active provider or a model
//   mova install      model1,model2,...      downloads models (with progress bar)
//   mova model-list                          installed models in active provider
//   mova remove       model1,model2,...      deletes models from active provider
//
// All of this lives in config/models/ — see docs/COMMANDS.md.
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
	consolePrint("active provider: " + provider + "\n")
}

// runShowConfig — mova show config [model]
func runShowConfig(root, model string) {
	state, err := models.GetActiveState(root)
	must(err)
	if state.Provider == "" {
		die("no active provider — run `mova config <provider>` first (e.g. `mova config ollama`)")
	}

	name := model
	if name == "" {
		name = state.Config
	}
	if name == "" {
		names, err := models.ListModelConfigs(root, state.Provider)
		must(err)
		consolePrint("active provider: " + state.Provider + " (no model selected yet)\n")
		if len(names) == 0 {
			consolePrint("no configuration files in config/models/" + state.Provider + " — create one (see docs/COMMANDS.md)\n")
		} else {
			consolePrint("configured models: " + strings.Join(names, ", ") + "\n")
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

// runInstall — mova install model1,model2,...
func runInstall(root, csv string) {
	state, err := models.GetActiveState(root)
	must(err)
	if state.Provider == "" {
		die("no active provider — run `mova config <provider>` first")
	}
	names := splitCSV(csv)
	if len(names) == 0 {
		die("specify at least one model: mova install llama3.1,mistral")
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
	consolePrint("installation complete: " + strings.Join(names, ", ") + "\n")
}

// runModelList — mova model-list
func runModelList(root string) {
	state, err := models.GetActiveState(root)
	must(err)
	if state.Provider == "" {
		die("no active provider — run `mova config <provider>` first")
	}
	list, err := models.ListInstalled(root, state.Provider)
	must(err)
	if len(list) == 0 {
		consolePrint("no installed models in " + state.Provider + "\n")
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

// runRemove — mova remove model1,model2,...
func runRemove(root, csv string) {
	state, err := models.GetActiveState(root)
	must(err)
	if state.Provider == "" {
		die("no active provider — run `mova config <provider>` first")
	}
	names := splitCSV(csv)
	if len(names) == 0 {
		die("specify at least one model: mova remove llama3.1,mistral")
	}
	must(models.Remove(root, state.Provider, names))
	consolePrint("removed: " + strings.Join(names, ", ") + "\n")
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

// progressBar — simple progress bar for `mova install`, one line
// per model, updated in-place using carriage return (\r).
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
	filled := percent / 5 // 20-character bar
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
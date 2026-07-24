package documents

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// DiffusionConfig mirrors config/models/diffusion/config.json — the local
// image-generation server's connection details, following the same
// per-provider config.json convention used by config/models/ollama and
// config/models/lmstudio.
type DiffusionConfig struct {
	Provider string `json:"provider"` // e.g. "automatic1111", "comfyui"
	BaseURL  string `json:"base_url"`
	Model    string `json:"model,omitempty"` // checkpoint/model name, if the server needs it
}

var diffusionClient = &http.Client{Timeout: 180 * time.Second}

// TriggerDiffusionImage sends prompt to the configured local diffusion
// server's AUTOMATIC1111-compatible /sdapi/v1/txt2img endpoint and writes
// the returned image to path. aspectRatio picks one of a few fixed
// resolutions (see aspectToSize) rather than arbitrary width/height, keeping
// the tool's surface simple.
func TriggerDiffusionImage(cfg DiffusionConfig, path, prompt, aspectRatio string) error {
	if cfg.BaseURL == "" {
		return fmt.Errorf("trigger_diffusion_image: config/models/diffusion/config.json no tiene base_url configurada")
	}
	width, height := aspectToSize(aspectRatio)

	body := map[string]any{
		"prompt":       prompt,
		"width":        width,
		"height":       height,
		"steps":        25,
		"cfg_scale":    7,
		"sampler_name": "Euler a",
	}
	if cfg.Model != "" {
		body["override_settings"] = map[string]string{"sd_model_checkpoint": cfg.Model}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.BaseURL+"/sdapi/v1/txt2img", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := diffusionClient.Do(req)
	if err != nil {
		return fmt.Errorf("trigger_diffusion_image: no se pudo contactar %s (¿está corriendo el servidor de difusión local?): %w", cfg.BaseURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("trigger_diffusion_image: el servidor devolvió %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed struct {
		Images []string `json:"images"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil || len(parsed.Images) == 0 {
		return fmt.Errorf("trigger_diffusion_image: respuesta sin imágenes")
	}
	imgData, err := base64.StdEncoding.DecodeString(parsed.Images[0])
	if err != nil {
		return fmt.Errorf("trigger_diffusion_image: imagen en base64 inválida: %w", err)
	}
	return os.WriteFile(path, imgData, 0o644)
}

func aspectToSize(aspectRatio string) (int, int) {
	switch aspectRatio {
	case "portrait", "9:16":
		return 576, 1024
	case "wide", "16:9":
		return 1024, 576
	case "square", "1:1", "":
		return 768, 768
	default:
		return 768, 768
	}
}

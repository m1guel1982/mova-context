// documents_tool.go — wires the "Documentos Avanzados y Formatos de
// Oficina", "Generación de Medios", and file/directory-creation capability
// groups (documents.*) into the same executeTool switch every other tool
// goes through — no separate code path, no separate transport. HTTP gets
// these for free because http/server.go is a thin wrapper over Process().
//
// See documents_tool_helpers.go for boolArg/hasArg/resolveSmartDir/
// resolveSmartFile/repoFor/formatAmbiguousMessage/parseSheetsData/
// loadDiffusionConfig.
package mcp

import (
	"fmt"

	"mova.local/core"
	"mova.local/documents"
)

func documentTool(adapter core.Adapter, root, tool string, args map[string]any) (string, error) {
	switch tool {
	// delete_path — the SINGLE unified entry point for removing files and
	// directories (see documents/delete_service.go), reachable
	// identically from chat's "/delete", this MCP tool, and HTTP's
	// POST /delete. Without confirm:true, nothing is deleted — the exact
	// "Delete \"x\"? (Y/N)" prompt text is returned instead, same
	// convention chat_completion's apply_edits already uses for
	// natural-language edits on non-interactive doors.
	case "delete_path":
		result, err := documents.Delete(root, documents.DeleteRequest{
			Paths:   pathsArg(args),
			Repo:    repoFor(adapter, args),
			Confirm: boolArg(args, "confirm"),
		})
		if err != nil {
			return "", err
		}
		return result.Message, nil

	case "create_directory":
		path, ambiguousMsg, err := resolveSmartDir(adapter, root, args, "path")
		if err != nil {
			return "", err
		}
		if ambiguousMsg != "" {
			return ambiguousMsg, nil
		}
		if err := documents.CreateDirectory(path); err != nil {
			return "", err
		}
		return "directory created: " + path, nil

	// save — the SINGLE unified entry point (see documents/save_service.go):
	// chooses Word/PDF/Excel/SVG/plain-text generation automatically from
	// the extension in "path", so a caller (chat, MCP, or the HTTP /save
	// endpoint) never needs to know generate_pdf_document/
	// generate_word_contract/generate_excel_report/write_file exist.
	// Those legacy tools are kept below, unchanged, for backward
	// compatibility with existing MCP clients/scripts.
	case "save":
		content := str(args, "content")
		if content == "" && args["history"] != nil {
			// "history": [{"role":"user","content":"..."}, ...] — the same
			// shape chat_completion's own "history" argument uses. "mode":
			// "all" | "range" (default: the last exchange, unchanged),
			// "range": "N-M" (1-indexed, only with mode:"range"),
			// "code_only"/"text_only": booleans — same selection logic
			// `/save -all`/`-range`/`-c`/`-text` use in chat (see
			// documents/save_selection.go), so Chat/MCP/HTTP behave
			// identically for "which text to save", not just for how the
			// resulting file gets written.
			turns := chatTurnsArg(args["history"])
			mode := documents.SelectionMode(str(args, "mode"))
			rangeStart, rangeEnd := documents.ParseRangeToken(str(args, "range"))
			selected, err := documents.SelectContent(turns, mode, rangeStart, rangeEnd, boolArg(args, "code_only"), boolArg(args, "text_only"))
			if err != nil {
				return "", err
			}
			content = selected
		}
		result, err := documents.Save(root, documents.SaveRequest{
			Path:              str(args, "path"),
			Directory:         str(args, "directory"),
			Content:           content,
			Append:            boolArg(args, "append"),
			Overwrite:         boolArg(args, "overwrite"),
			OverwriteExplicit: hasArg(args, "overwrite"),
			Repo:              repoFor(adapter, args),
		})
		if err != nil {
			return "", err
		}
		return result.Message, nil

	case "read_document_layer":
		path, ambiguousMsg, err := resolveSmartFile(adapter, root, args, "filename")
		if err != nil {
			return "", err
		}
		if ambiguousMsg != "" {
			return ambiguousMsg, nil
		}
		return documents.ReadDocumentLayer(path)

	case "read_file":
		path, ambiguousMsg, err := resolveSmartFile(adapter, root, args, "filename")
		if err != nil {
			return "", err
		}
		if ambiguousMsg != "" {
			return ambiguousMsg, nil
		}
		return documents.ReadFile(path)

	case "write_file":
		path, ambiguousMsg, err := resolveSmartFile(adapter, root, args, "filename")
		if err != nil {
			return "", err
		}
		if ambiguousMsg != "" {
			return ambiguousMsg, nil
		}
		if err := ensureDir(path); err != nil {
			return "", err
		}
		content := str(args, "content")
		if err := documents.WriteFile(path, content); err != nil {
			return "", err
		}
		return "file written: " + path, nil

	case "patch_file":
		path, ambiguousMsg, err := resolveSmartFile(adapter, root, args, "filename")
		if err != nil {
			return "", err
		}
		if ambiguousMsg != "" {
			return ambiguousMsg, nil
		}
		search := str(args, "search")
		replace := str(args, "replace")
		if search == "" {
			return "", fmt.Errorf("search is required")
		}
		if err := documents.PatchFile(path, search, replace); err != nil {
			return "", err
		}
		return "file patched: " + path, nil

	case "generate_word_contract":
		path, ambiguousMsg, err := resolveSmartFile(adapter, root, args, "filename")
		if err != nil {
			return "", err
		}
		if ambiguousMsg != "" {
			return ambiguousMsg, nil
		}
		if err := ensureDir(path); err != nil {
			return "", err
		}
		content := str(args, "markdown_content")
		if err := documents.GenerateWordContract(path, content); err != nil {
			return "", err
		}
		return "Word document generated: " + path, nil

	case "generate_pdf_document":
		path, ambiguousMsg, err := resolveSmartFile(adapter, root, args, "filename")
		if err != nil {
			return "", err
		}
		if ambiguousMsg != "" {
			return ambiguousMsg, nil
		}
		if err := ensureDir(path); err != nil {
			return "", err
		}
		layout := str(args, "layout_html_css")
		if err := documents.GeneratePDFDocument(path, layout); err != nil {
			return "", err
		}
		return "PDF generated: " + path, nil

	case "generate_vector_graphic":
		path, ambiguousMsg, err := resolveSmartFile(adapter, root, args, "filename")
		if err != nil {
			return "", err
		}
		if ambiguousMsg != "" {
			return ambiguousMsg, nil
		}
		if err := ensureDir(path); err != nil {
			return "", err
		}
		svgCode := str(args, "svg_code")
		if err := documents.GenerateVectorGraphic(path, svgCode); err != nil {
			return "", err
		}
		return "SVG generated: " + path, nil

	case "generate_excel_report":
		path, ambiguousMsg, err := resolveSmartFile(adapter, root, args, "filename")
		if err != nil {
			return "", err
		}
		if ambiguousMsg != "" {
			return ambiguousMsg, nil
		}
		if err := ensureDir(path); err != nil {
			return "", err
		}
		sheets, err := parseSheetsData(args["sheets_data"])
		if err != nil {
			return "", err
		}
		if err := documents.GenerateExcelReport(path, sheets); err != nil {
			return "", err
		}
		return "Excel generated: " + path, nil

	case "trigger_diffusion_image":
		path, ambiguousMsg, err := resolveSmartFile(adapter, root, args, "filename")
		if err != nil {
			return "", err
		}
		if ambiguousMsg != "" {
			return ambiguousMsg, nil
		}
		if err := ensureDir(path); err != nil {
			return "", err
		}
		cfg, err := loadDiffusionConfig(root)
		if err != nil {
			return "", err
		}
		prompt := str(args, "prompt")
		aspectRatio := str(args, "aspect_ratio")
		if err := documents.TriggerDiffusionImage(cfg, path, prompt, aspectRatio); err != nil {
			return "", err
		}
		return "image generated: " + path, nil

	default:
		return "", fmt.Errorf("unknown tool: %s", tool)
	}
}

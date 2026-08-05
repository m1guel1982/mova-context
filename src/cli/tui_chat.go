// tui_chat.go — the chat screen. Setup (session, project context,
// Budget gate) mirrors `mova chat`'s runChat (chat_cmd.go) exactly —
// same models.NewSession, core.BuildContextSections, budget.EnforceLimit
// — and every ordinary message is sent through the same sendWithTools
// (chat_helpers.go) the CLI REPL uses. Replies are shown once complete
// (no token-by-token streaming in the TUI, by design — see chatSendCmd).
//
// Commands (set -model, /memory, /budget, /tools, /clear, /save,
// /delete, exit|quit) are intercepted the SAME way `mova chat`'s REPL
// intercepts them (see chat_cmd.go's switch) — see handleCommand below —
// so typing one of these here never gets sent to the model as an
// ordinary chat message. The underlying implementations (runChatMemory,
// runChatBudget, runChatSave in chat_save.go) are shared with the CLI;
// only the output sink differs (an `emit` callback that appends to this
// screen's in-memory transcript instead of writing to stdout, since
// writing straight to the terminal mid-render would corrupt a Bubble Tea
// screen — same convention sendWithTools already established). `/delete`
// has no interactive terminal to read a y/n from here, so it uses its
// own small pending-confirmation state machine instead of chat_cmd.go's
// blocking scanner (see pendingDelete below) — same non-blocking
// contract documents.Delete already offers MCP/HTTP.
package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"mova.local/budget"
	"mova.local/core"
	"mova.local/documents"
	"mova.local/mcp"
	"mova.local/models"
)

type chatScreen struct {
	app       *tuiApp
	project   string
	sess      *models.Session
	adapter   core.Adapter
	proj      *core.Project
	fileState *chatFileState

	// pendingDelete holds an unconfirmed /delete request while it waits
	// for the person's next line to be "y"/"n" — see handleCommand and
	// resolvePendingDelete.
	pendingDelete *documents.DeleteResult

	transcript strings.Builder
	vp         viewport.Model
	input      textinput.Model
	spin       spinner.Model
	waiting    bool
	setupNote  string
}

func newChatScreen(app *tuiApp, project string) *chatScreen {
	c := &chatScreen{app: app, project: project, fileState: &chatFileState{}}

	sess, err := models.NewSession(app.root)
	if err != nil {
		c.setupNote = "Failed to start session: " + err.Error()
	}
	c.sess = sess

	if project != "" && sess != nil {
		fa := core.NewFileAdapter(app.root)
		proj, _ := fa.GetProject(project)
		c.proj = proj
		c.adapter = newAdapter(app.root, proj)
		applyProjectLLMProfile(sess, app.root, proj)

		// budget.BuildGatedContext runs the full Token Firewall
		// (Sanitizer → Circuit Breaker → the existing max_tokens gate)
		// — the exact same pipeline the CLI's runChat (chat_cmd.go)
		// uses, so the TUI never has its own copy of "build then gate".
		gated := budget.BuildGatedContext(c.adapter, app.root, project, "")
		if gated.Err != nil {
			c.setupNote = gated.Err.Error()
		} else {
			systemText, boundary, _ := applyCacheLayoutQuiet(gated.Sections, proj)
			sess.SetSystem(systemText + mcp.ToolsSystemPrompt(proj.Tools))
			sess.CacheBoundary = boundary
			c.setupNote = "Loaded project: " + project
			if gated.CircuitBreaker.Message != "" {
				c.setupNote += " — " + gated.CircuitBreaker.Message
			}
		}
	}

	vp := viewport.New(90, 20)
	ti := textinput.New()
	ti.Placeholder = "Type a message, or a command (set -model, /memory, /budget, /save, /delete, /tools, /clear, exit)…"
	ti.Focus()
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	c.vp, c.input, c.spin = vp, ti, sp
	return c
}

func (c *chatScreen) Init() tea.Cmd { return nil }

type chatReplyMsg struct {
	reply string
	err   error
}

func chatSendCmd(c *chatScreen, text string) tea.Cmd {
	sess, adapter, proj, root := c.sess, c.adapter, c.proj, c.app.root
	return func() tea.Msg {
		reply, _, err := sendWithTools(sess, adapter, proj, root, text, func(string) {})
		return chatReplyMsg{reply: reply, err: err}
	}
}

// emit appends one line of command output to the transcript — the sink
// passed to runChatMemory/runChatBudget/runChatSave (chat_save.go)
// instead of consolePrint, so their output lands in this screen instead
// of corrupting the Bubble Tea render.
func (c *chatScreen) emit(s string) {
	c.transcript.WriteString(s)
}

// handleCommand intercepts the same set of chat commands `mova chat`'s
// REPL recognizes (see chat_cmd.go): exit/quit, "set -model", /memory,
// /budget, /tools, /clear, /save, and /delete. Returns handled=false for
// anything else, so it falls through to an ordinary chat turn — exactly
// mirroring chat_cmd.go's switch/default. Never sends a recognized
// command to the model.
func (c *chatScreen) handleCommand(text string) (handled bool, cmd tea.Cmd) {
	switch {
	case text == "exit" || text == "quit" || text == "salir":
		return true, tuiPop()

	case strings.HasPrefix(text, "set -model") || strings.HasPrefix(text, "set model"):
		name := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(text, "set -model"), "set model"))
		if name == "" {
			c.emit("Usage: set -model <name>\n")
			return true, nil
		}
		if err := c.sess.SetModel(name); err != nil {
			c.emit("Error: " + err.Error() + "\n")
			return true, nil
		}
		c.emit(fmt.Sprintf("[Model] Switched to: %s (provider: %s)\n", c.sess.Model, c.sess.Provider))
		return true, nil

	case text == "/memory":
		runChatMemory(c.adapter, c.project, c.sess, c.emit)
		return true, nil

	case text == "/budget":
		runChatBudget(c.app.root, c.adapter, c.project, "", c.emit)
		return true, nil

	case text == "/tools":
		c.emit(mcp.FileToolsHelp())
		return true, nil

	case text == "/clear":
		// A real terminal clear (chat_cmd.go's clearScreen) would shell
		// out and corrupt the Bubble Tea render here — resetting the
		// in-memory transcript is the TUI-native equivalent.
		c.transcript.Reset()
		return true, nil

	case strings.HasPrefix(text, "/save"):
		runChatSave(c.adapter, c.app.root, c.proj, c.sess, strings.TrimSpace(strings.TrimPrefix(text, "/save")), c.fileState, c.emit)
		return true, nil

	case strings.HasPrefix(text, "/delete"):
		c.startDelete(strings.TrimSpace(strings.TrimPrefix(text, "/delete")))
		return true, nil
	}
	return false, nil
}

// startDelete implements `/delete "a.txt" ["b.txt" ...]` — builds the
// confirmation prompt via the same documents.Delete(Confirm:false) every
// non-interactive door (MCP/HTTP) already uses (see delete_service.go),
// and stores it in pendingDelete so the NEXT line typed is read as y/n
// instead of a chat message (see resolvePendingDelete) — there is no
// blocking scanner to read from inside a Bubble Tea Update loop, unlike
// chat_cmd.go's REPL.
func (c *chatScreen) startDelete(rest string) {
	paths := parseDeletePaths(rest)
	if len(paths) == 0 {
		c.emit("Usage: /delete \"file.txt\" [\"another.txt\" \"dir/\" ...]\n")
		return
	}
	repo := "."
	if c.proj != nil && c.proj.Repo != "" {
		repo = c.proj.Repo
	}
	pending, err := documents.Delete(c.app.root, documents.DeleteRequest{Paths: paths, Repo: repo})
	if err != nil {
		c.emit("Error: " + err.Error() + "\n")
		return
	}
	c.pendingDelete = &pending
	c.emit(documents.FormatDeletePrompt(pending.Items) + "\n")
}

// resolvePendingDelete reads the person's answer to a pending /delete
// prompt (see startDelete) — "y"/"yes"/"s"/"si"/"sí" confirms every
// pending item, anything else cancels. Same accepted answers as
// chat_cmd.go's runChatDelete.
func (c *chatScreen) resolvePendingDelete(answer string) {
	pending := c.pendingDelete
	c.pendingDelete = nil

	answer = strings.ToLower(strings.TrimSpace(answer))
	if answer != "y" && answer != "yes" && answer != "s" && answer != "si" && answer != "sí" {
		c.emit("Nothing deleted.\n")
		return
	}

	paths := make([]string, len(pending.Items))
	for i, item := range pending.Items {
		paths[i] = item.Requested
	}
	repo := "."
	if c.proj != nil && c.proj.Repo != "" {
		repo = c.proj.Repo
	}
	result, err := documents.Delete(c.app.root, documents.DeleteRequest{Paths: paths, Repo: repo, Confirm: true})
	if err != nil {
		c.emit("Error: " + err.Error() + "\n")
		return
	}
	c.emit(result.Message + "\n")
}

func (c *chatScreen) Update(msg tea.Msg) (tuiScreen, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.vp.Width, c.vp.Height = msg.Width-6, msg.Height-11
		c.input.Width = msg.Width - 10

	case tea.KeyMsg:
		if msg.String() == "enter" && !c.waiting {
			text := strings.TrimSpace(c.input.Value())
			if text == "" || c.sess == nil {
				return c, nil
			}
			c.transcript.WriteString("\n> " + text + "\n")
			c.input.SetValue("")

			// A pending /delete confirmation takes priority over
			// anything else typed next — it consumes exactly one line
			// (y/n), never falls through to a command or a chat turn.
			if c.pendingDelete != nil {
				c.resolvePendingDelete(text)
				c.vp.SetContent(c.transcript.String())
				c.vp.GotoBottom()
				return c, nil
			}

			if handled, cmd := c.handleCommand(text); handled {
				c.vp.SetContent(c.transcript.String())
				c.vp.GotoBottom()
				return c, cmd
			}

			c.vp.SetContent(c.transcript.String())
			c.vp.GotoBottom()
			c.waiting = true
			return c, tea.Batch(chatSendCmd(c, text), c.spin.Tick)
		}

	case chatReplyMsg:
		c.waiting = false
		if msg.err != nil {
			c.transcript.WriteString("\n" + msg.err.Error() + "\n")
		} else {
			c.transcript.WriteString(fmt.Sprintf("\n[%s]\n%s\n", c.sess.Model, msg.reply))
		}
		c.vp.SetContent(c.transcript.String())
		c.vp.GotoBottom()
		return c, nil

	case spinner.TickMsg:
		if c.waiting {
			var cmd tea.Cmd
			c.spin, cmd = c.spin.Update(msg)
			return c, cmd
		}
		return c, nil
	}

	var cmd, cmd2 tea.Cmd
	c.input, cmd = c.input.Update(msg)
	c.vp, cmd2 = c.vp.Update(msg)
	return c, tea.Batch(cmd, cmd2)
}

func (c *chatScreen) View() string {
	title := "Chat"
	if c.project != "" {
		title = "Chat — " + c.project
	}
	body := tuiHeader(title) + "\n"
	if c.setupNote != "" {
		body += tuiHelpStyle.Render(c.setupNote) + "\n\n"
	}
	body += c.vp.View() + "\n\n"
	if c.waiting {
		body += c.spin.View() + " waiting for response…\n"
	} else if c.pendingDelete != nil {
		body += c.input.View() + "\n"
		body += tuiFooter("y/n: confirm delete · esc: back")
		return tuiDocStyle.Render(body)
	} else {
		body += c.input.View() + "\n"
	}
	body += tuiFooter("enter: send · commands: set -model, /memory, /budget, /tools, /clear, /save, /delete · esc: back")
	return tuiDocStyle.Render(body)
}

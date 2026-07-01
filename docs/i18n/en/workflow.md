# workflow.md

Instruction system for resolving project context.

> Full documentation: [docs/i18n/es/workflow.md](docs/i18n/es/workflow.md) · [docs/i18n/en/workflow.md](docs/i18n/en/workflow.md)

---

## WORKSPACE

```text
mova-context/              ← convention repo (agents, skills, prompts, projects)
├── agents/[domain]/i18n/[lang]/
├── skills/[domain]/i18n/[lang]/
├── prompts/[domain]/i18n/[lang]/
├── projects/
│   └── [PROJECT]/
│       ├── project.json
│       └── memory.md
└── workflow.md             ← you are here
```

Generated code can live in any directory — inside or outside `mova-context`.
The working path is declared in `project.json` as `"repo"`.

```json
"repo": "."                                  ← relative to mova-context (default)
"repo": "../local-test-app"                  ← external sibling folder
"repo": "E:/projects/my-project"             ← absolute path, any OS
```

Working directory resolution rules:

```text
IF "repo" is "." or absent
→ work inside mova-context

IF "repo" is a relative path
→ resolve relative to mova-context's location

IF "repo" is an absolute path
→ use it directly

IF the path does not exist
→ create it before generating any file
→ confirm to the user the path being used
```

Agent/skill/prompt/project lookup: always inside `mova-context`, regardless of `repo`.
Code and project file generation: always at the path indicated by `repo`.

---

## ACTIVATION

Valid inputs:

```text
Read workflow.md
Read workflow.md → [PROJECT]
Read workflow.md → [PROJECT] → [TASK]

load [PROJECT]
run [TASK] from [PROJECT]
```

---

## FILE ACCESS

```text
IF the environment allows filesystem access
→ locate and read files automatically, recursively within "repo" and mova-context

IF the environment does NOT allow filesystem access
→ request only the missing files
→ continue once available
```

---

## PROJECT DETECTION

```text
IF only one project exists
→ use it automatically

IF multiple projects exist
→ ask for the name once

IF the user specifies a project
→ use that project
```

---

## PROJECT RESOLUTION

```text
1. Locate projects/[PROJECT]/project.json

IF it does not exist
→ report the error
→ request a valid project
→ stop execution

IF it exists
→ continue
```

---

## TASK RESOLUTION

```text
IF the user specifies a task
→ use that task

IF no task is specified
→ use default_task

IF the task does not exist
→ report error
→ show available tasks

IF default_task does not exist
→ request a valid task
→ stop execution
```

---

## EXECUTION SEQUENCE

```text
1.  Read project.json
2.  Resolve lang (configured language)
3.  Resolve llm_profile (provider + model + profile)
4.  Resolve adapter (storage: file / postgresql / mongodb)
5.  Resolve task
6.  Load agents
7.  Load skills
8.  Load prompt
9.  Read memory.md
10. Merge variables (task > global)
11. Inject variables
12. Execute
13. Update memory.md
```

---

## FILE RESOLUTION (i18n + domain)

For `agent: backend-dev`, `domain: software`, `lang: en`:

```text
agents/software/i18n/en/backend-dev.md   ← use
agents/software/i18n/es/backend-dev.md   ← fallback if en/ is missing
agents/base/i18n/en/backend-dev.md       ← legacy fallback
```

The same resolution logic applies to `skills/` and `prompts/`.

---

## AGENTS

```json
"agents": {
  "domain": "software",
  "use": ["backend-dev", "security-architect"]
}
```

Load order, for each name in `use`:

```text
1. agents/[domain]/i18n/[lang]/[name].md      ← domain base
2. agents/custom/i18n/[lang]/[name].md        ← project-specific override, if present
```

Agents defined in a task are added to the global ones (not replaced).

---

## SKILLS

```json
"skills": {
  "domain": "legal",
  "use": ["privacy-law-obligations", "data-subject-rights"]
}
```

Load order, for each name in `use`:

```text
1. skills/[domain]/i18n/[lang]/[name].md      ← domain base
2. skills/custom/i18n/[lang]/[name].md        ← project-specific override, if present
```

Skills defined in a task are added to the global ones (not replaced).

---

## PROMPTS

```json
"tasks": {
  "analyze-contract": {
    "prompt": "analyze-contract-data"
  }
}
```

Load:

```text
1. prompts/[domain]/i18n/[lang]/[name].md     ← domain base
2. prompts/custom/i18n/[lang]/[name].md       ← project-specific override, if present
```

---

## VARIABLES

Source:

```text
project.json → variables (global)
task → variables (override global on key match)
```

Priority:

```text
task > global
```

Automatic normalization:

```text
Every snake_case key is converted to {{UPPER_CASE}} for injection.

Rule: convert to uppercase, wrap in {{ }}

Examples:
  project        → {{PROJECT}}
  api_prefix     → {{API_PREFIX}}
  document_type  → {{DOCUMENT_TYPE}}
  any_new_key    → {{ANY_NEW_KEY}}
```

There is no fixed list of allowed variables.
Any key declared in `variables` (global or task-level) is automatically normalized and injected into every loaded agent, skill, and prompt.

If an agent, skill, or prompt uses `{{VARIABLE_NAME}}` and that variable was not declared in `project.json`, the placeholder is left unreplaced and the user is informed which variable is missing.

System reserved variables (always available without declaring them):

```text
{{PROJECT}}    → value of "project" in project.json
{{REPO}}       → value of "repo" in project.json
{{TASK}}       → active task name
{{LANG}}       → configured language
```

---

## FOCUS

Defines exactly what to work on: specific files, directories, or source code symbols.
When present, the model works only on those elements, not the entire project.

Declaration in `project.json` (global or within a task):

```json
"focus": ["file.js", "src/services", "functionName()", "ClassName"]
```

Resolution rules:

```text
IF the element has an absolute path
→ use it directly

IF the element is just a name (no path separator)
→ search recursively within "repo"
→ IF it appears in more than one place → report matches and ask for confirmation
→ IF not found → report and continue without it

IF the element ends in () or has the form Name without extension
→ treat it as a symbol (function, method, class)
→ search within the "focus" files, or the whole project if no other focus is set
```

Priority (same as variables):

```text
task > global
```

`{{FOCUS}}` is injected as a readable list into agents, skills, and prompts that reference it.
If `focus` is not declared, the model works on the entire project.

---

## MEMORY

Path:

```text
projects/[PROJECT]/memory.md
```

Rules:

```text
IF it exists → read before executing
IF it does not exist → create it

Update at the end of each session.
```

Format:

```md
## YYYY-MM-DD — session

**Done:**
**Resolved:**
**Pending:**
**Decisions:**
**LLM Errors:**
```

---

## LOAD RULES

```text
domain base → custom

custom complements or overrides base.
```

```text
IF a file does not exist
→ skip and continue
```

Full order:

```text
1.  global agents, domain base
2.  global agents, custom
3.  task agents, domain base
4.  task agents, custom

5.  global skills, domain base
6.  global skills, custom
7.  task skills, domain base
8.  task skills, custom

9.  prompt, domain base
10. prompt, custom
```

---

## REFERENCES

Apply the active context to:

```text
src/[path]
file.ext
function()
class
absolute path
```

---

## EXPECTED RESULT

```text
1. Resolve project and task
2. Resolve lang, llm_profile, and adapter
3. Load context (agents + skills + prompt + memory) recursively
4. Apply and merge variables (including focus)
5. Execute
6. Update memory.md
```
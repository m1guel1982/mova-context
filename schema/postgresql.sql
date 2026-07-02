-- Mova Context — PostgreSQL schema
-- KISS: three tables, that's it.
-- Mirrors the file structure: kind/domain/lang/name → content

-- knowledge mirrors agents/, skills/, prompts/ folders
CREATE TABLE knowledge (
    id        SERIAL PRIMARY KEY,
    kind      TEXT NOT NULL CHECK (kind IN ('agent','skill','prompt','workflow')),
    domain    TEXT NOT NULL DEFAULT '',  -- 'software', 'callcenter', 'legal', ''
    lang      TEXT NOT NULL DEFAULT '',  -- 'es', 'en', 'fr', '' = universal/legacy
    name      TEXT NOT NULL,
    content   TEXT NOT NULL DEFAULT '',
    is_custom BOOLEAN NOT NULL DEFAULT false,
    active    BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fts TSVECTOR GENERATED ALWAYS AS (
        to_tsvector('simple', name || ' ' || content)
    ) STORED,
    UNIQUE (kind, domain, lang, name)
);
CREATE INDEX ON knowledge USING GIN(fts);
CREATE INDEX ON knowledge(kind, domain, lang) WHERE active;

-- projects mirrors projects/ folder
CREATE TABLE projects (
    name         TEXT PRIMARY KEY,
    description  TEXT,
    repo         TEXT,
    lang         TEXT NOT NULL DEFAULT '',
    adapter      TEXT NOT NULL DEFAULT 'file',
    dsn          TEXT,
    llm          TEXT,
    default_task TEXT,
    variables    JSONB NOT NULL DEFAULT '{}',
    agents       JSONB NOT NULL DEFAULT '{}',
    skills       JSONB NOT NULL DEFAULT '{}',
    tasks        JSONB NOT NULL DEFAULT '{}',
    active       BOOLEAN NOT NULL DEFAULT true,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- memory mirrors projects/[name]/memory.md
CREATE TABLE memory (
    id           SERIAL PRIMARY KEY,
    project      TEXT NOT NULL REFERENCES projects(name) ON DELETE CASCADE,
    content      TEXT NOT NULL,
    session_date DATE NOT NULL DEFAULT CURRENT_DATE,
    archived     BOOLEAN NOT NULL DEFAULT false,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON memory(project, created_at DESC) WHERE NOT archived;
CREATE INDEX ON memory(project, session_date) WHERE archived;

-- Auto-update timestamp
CREATE OR REPLACE FUNCTION _updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$;
CREATE TRIGGER t_knowledge_ts BEFORE UPDATE ON knowledge
    FOR EACH ROW EXECUTE FUNCTION _updated_at();
CREATE TRIGGER t_projects_ts BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION _updated_at();

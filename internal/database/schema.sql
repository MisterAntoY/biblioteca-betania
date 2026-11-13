-- Esquema de Banco de Dados de Alta Performance - BiblioTânia (Colégio Betânia)
-- Otimizado para SQLite com FTS5, concorrência WAL e escala até 100 Milhões

PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 5000;
PRAGMA cache_size = -64000; -- 64MB cache em memória

-- 1. Disciplinas Escolares
CREATE TABLE IF NOT EXISTS disciplinas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nome TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE,
    icone TEXT NOT NULL DEFAULT '📚',
    ordem INTEGER NOT NULL DEFAULT 0
);

-- 2. Níveis de Ensino do Colégio
CREATE TABLE IF NOT EXISTS niveis (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nome TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE
);

-- 3. Acervo de Livros
CREATE TABLE IF NOT EXISTS livros (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    titulo TEXT NOT NULL,
    autor TEXT NOT NULL DEFAULT 'Autor Desconhecido',
    disciplina_id INTEGER REFERENCES disciplinas(id) ON DELETE SET NULL,
    nivel_id INTEGER REFERENCES niveis(id) ON DELETE SET NULL,
    sinopse TEXT,
    ano_publicacao INTEGER,
    formato TEXT NOT NULL DEFAULT 'pdf', -- pdf, docx, link
    arquivo_url TEXT,
    link_externo TEXT,
    tamanho_bytes INTEGER DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'ativo', -- ativo, pendente, rejeitado
    submetido_por TEXT DEFAULT 'Curadoria Colégio Betânia',
    visualizacoes INTEGER DEFAULT 0,
    downloads INTEGER DEFAULT 0,
    criado_em DATETIME DEFAULT CURRENT_TIMESTAMP,
    atualizado_em DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Índices de alta velocidade para paginação e filtragem
CREATE INDEX IF NOT EXISTS idx_livros_status_id ON livros(status, id DESC);
CREATE INDEX IF NOT EXISTS idx_livros_disciplina ON livros(disciplina_id, status);
CREATE INDEX IF NOT EXISTS idx_livros_nivel ON livros(nivel_id, status);
CREATE INDEX IF NOT EXISTS idx_livros_criado_em ON livros(criado_em DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_livros_titulo_autor ON livros(titulo, autor);

-- 4. Tabela Virtual FTS5 para Busca Textual Instantânea
CREATE VIRTUAL TABLE IF NOT EXISTS livros_fts USING fts5(
    titulo,
    autor,
    sinopse,
    content='livros',
    content_rowid='id',
    tokenize='unicode61 remove_diacritics 2'
);

-- Triggers atômicos para sincronização imediata com FTS5
CREATE TRIGGER IF NOT EXISTS trg_livros_ai AFTER INSERT ON livros BEGIN
    INSERT INTO livros_fts(rowid, titulo, autor, sinopse)
    VALUES (new.id, new.titulo, new.autor, new.sinopse);
END;

CREATE TRIGGER IF NOT EXISTS trg_livros_ad AFTER DELETE ON livros BEGIN
    INSERT INTO livros_fts(livros_fts, rowid, titulo, autor, sinopse)
    VALUES ('delete', old.id, old.titulo, old.autor, old.sinopse);
END;

CREATE TRIGGER IF NOT EXISTS trg_livros_au AFTER UPDATE ON livros BEGIN
    INSERT INTO livros_fts(livros_fts, rowid, titulo, autor, sinopse)
    VALUES ('delete', old.id, old.titulo, old.autor, old.sinopse);
    INSERT INTO livros_fts(rowid, titulo, autor, sinopse)
    VALUES (new.id, new.titulo, new.autor, new.sinopse);
END;

-- 5. Fila de Submissões de Estudantes (Triagem da IA)
CREATE TABLE IF NOT EXISTS submissoes_estudantes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    estudante_nome TEXT NOT NULL,
    estudante_classe TEXT NOT NULL,
    titulo TEXT NOT NULL,
    autor TEXT DEFAULT '',
    disciplina_id INTEGER REFERENCES disciplinas(id) ON DELETE SET NULL,
    nivel_id INTEGER REFERENCES niveis(id) ON DELETE SET NULL,
    tipo_envio TEXT NOT NULL, -- arquivo, link
    arquivo_path TEXT,
    arquivo_nome_original TEXT,
    link_url TEXT,
    status TEXT NOT NULL DEFAULT 'pendente', -- pendente, aprovado, rejeitado, erro
    ai_parecer TEXT DEFAULT '',
    ai_avaliado_em DATETIME,
    livro_id INTEGER REFERENCES livros(id) ON DELETE SET NULL,
    criado_em DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_submissoes_status ON submissoes_estudantes(status, criado_em ASC);

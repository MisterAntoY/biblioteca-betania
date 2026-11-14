# 📖 BiblioTânia — Biblioteca Digital do Colégio Betânia

> **Plataforma Educacional Aberta de Alto Desempenho e Baixo Consumo**, projetada especificamente para o **Colégio Betânia**, com triagem autônoma por Inteligência Artificial e arquitetura em Go preparada para escalar rumo a 100 Milhões de obras abertas.
> 
> 📄 Para o manifesto pedagógico e histórico institucional completo, consulte **[SOBRE.md](SOBRE.md)**.

---

## 🏛️ 1. Visão Geral e Contexto
A **BiblioTânia** nasceu a partir do sucesso da curadoria pedagógica e foi concebida sob medida para a realidade do **Colégio Betânia**. 

Diferente de aplicações web tradicionais baseadas em Next.js e pesadas runtimes de JavaScript (que consomem centenas de megabytes de RAM e exigem manutenções constantes de pacotes npm), a BiblioTânia foi reescrita inteiramente do zero em **Go (Golang)**:
- **Binário Único Auto-Contido (~12MB):** Todo o servidor, rotas, banco de dados embarcado, templates HTML, estilos CSS, ícones vetoriais e capas estão embutidos diretamente no executável via `go:embed`.
- **Consumo Mínimo de Recursos:** Menos de **20MB de memória RAM** em repouso e consumo de CPU quase nulo.
- **Zero Dependências em Produção:** Sem necessidade de Node.js, npm, Docker pesado ou serviços externos de banco de dados.
- **Identidade Visual Institucional:** Paleta em **Azul Marinho Betânia** (`#0B1E3B`) e **Ouro Académico** (`#D97706`), com logotipo oficial e capas editoriais físicas (proporção 3:4 com relevo de lombada).
- **Acervo Curricular Nacional:** 152+ obras didáticas abrangendo o Ensino Primário (1ª à 6ª classe), I Ciclo (7ª à 9ª classe), II Ciclo / Pré-Universitário (10ª à 12ª classe), Ensino Técnico (TLP, RCC, SEAC, SIBD), Cânone da Literatura Angolana e Línguas Nacionais (Kimbundu, Umbundu, Kikongo e Cokwe).

---

## ⚡ 2. Arquitetura & Escala para 100 Milhões de Livros
- **Motor de Banco de Dados:** SQLite com **FTS5** (Full Text Search) embutido, configurado em modo **WAL (Write-Ahead Logging)**, `busy_timeout` de 5.000ms e cache de memória de 64MB (`PRAGMA cache_size = -64000`).
- **Concorrência Segura:** Serialização de escrita e leituras paralelas protegidas por `sync.RWMutex`, eliminando completamente race conditions e erros de `database is locked`.
- **Indexação por Triggers:** Triggers atômicos no banco atualizam o índice FTS5 instantaneamente a cada nova inserção, exclusão ou atualização.

---

## 🛡️ 3. Segurança e Robustez Operacional

| Vetor de Risco | Mitigação Implementada na BiblioTânia |
| :--- | :--- |
| **SQL Injection** | 100% das consultas utilizam **Prepared Statements** parametrizados com placeholders `?`. Termos FTS5 são estritamente sanitizados via regex e tokenização de unicode. |
| **Cross-Site Scripting (XSS)** | O motor nativo `html/template` de Go aplica **auto-escaping contextual automático** em todas as variáveis renderizadas. Injeção de tags `<script>` e handlers de imagem são convertidos em entidades inofensivas. |
| **Cabeçalhos HTTP (CSP & Sniffing)** | Middleware com `Content-Security-Policy`, `X-Content-Type-Options: nosniff`, `X-Frame-Options: SAMEORIGIN` e `Referrer-Policy`. |
| **Race Conditions** | Pool de conexões otimizado + `sync.RWMutex` serializando mutações no SQLite. Validado com `go test -race`. |
| **Path Traversal & DoS** | Nomes de arquivos sanitizados com hashes criptográficos aleatórios; uploads limitados a 50MB via `http.MaxBytesReader`; validação de MIME types pelos primeiros 512 bytes (sniffing com `http.DetectContentType`). |
| **SSRF em Links de Alunos** | Links externos são validados contra loopback (`localhost`, `127.0.0.1`, IPs privados RFC 1918) antes do processamento. |

---

## 🤖 4. Mecanismo de Admissão por Inteligência Artificial
O fluxo de novos livros segue o modelo pedagógico colaborativo:

1. **Submissão pelo Aluno (`/submeter`):**
   - O estudante informa seu Nome, Classe/Turma, Título, Disciplina e Nível.
   - Fornece o material via **Arquivo (PDF / DOCX)** ou **Link Externo Público (HTTPS)**.
2. **Registro e Fila:**
   - A submissão é gravada com `status = 'pendente'`.
3. **Triagem Automática pelo Worker da IA (`curator.Worker`):**
   - O worker assíncrono avalia o material utilizando a API de IA configurada (**Google Gemini 2.5 Flash** ou **OpenAI** / **Heurística Local Educacional**).
   - O Curador de IA avalia:
     - **Relevância Pedagógica:** Adequação ao Ensino Primário ou Secundário do Colégio Betânia.
     - **Segurança Escolar:** Ausência de violência, conteúdo adulto, apostas, malware ou material comercial.
4. **Catalogação Automática:**
   - Se **Aprovado:** A obra é imediatamente inserida no catálogo ativo (`livros`), indexada no FTS5 e disponibilizada para leitura de toda a comunidade escolar.
   - Se **Rejeitado:** Um parecer pedagógico construtivo é gerado e registrado no **Mural da IA** (`/submissoes`) para orientação do aluno.

---

## 🚀 5. Como Executar

### Pré-requisitos
- Go 1.22+ (instalado no sistema).

### Compilação e Execução
```bash
# 1. Clonar ou acessar a pasta
cd /home/rmendes/biblioteca-betania

# 2. Executar a suíte de testes de segurança
make test

# 3. Compilar o binário único
make build

# 4. Iniciar o servidor
make run
```
O servidor estará disponível imediatamente em: **`http://localhost:8080`**.

---

## ⚙️ 6. Variáveis de Ambiente (Opcionais)

| Variável | Padrão | Descrição |
| :--- | :--- | :--- |
| `PORT` | `8080` | Porta TCP do servidor web |
| `DB_PATH` | `./data/bibliotania.db` | Caminho do arquivo SQLite com FTS5 |
| `UPLOAD_DIR` | `./uploads` | Diretório para armazenamento dos livros enviados pelos alunos |
| `MAX_FILE_SIZE_MB`| `50` | Limite de tamanho por upload em Megabytes |
| `GEMINI_API_KEY` | *(opcional)* | Chave de API para o Google Gemini 2.5 Flash |
| `OPENAI_API_KEY` | *(opcional)* | Chave de API para OpenAI / modelo compatível |

*Nota: Se nenhuma chave de API externa for configurada, o sistema ativa automaticamente o **Modo Heurístico Educacional Resiliente**, garantindo que o envio de livros e o funcionamento da biblioteca nunca sejam interrompidos.*

---

## 📂 7. Estrutura do Código

```
biblioteca-betania/
├── cmd/
│   └── server/main.go            # Ponto de entrada, servidor HTTP e graceful shutdown
├── internal/
│   ├── config/config.go          # Configurações e variáveis de ambiente
│   ├── database/
│   │   ├── db.go                 # Gerenciamento SQLite, concorrência, FTS5 e transações
│   │   ├── schema.sql            # Esquema SQL otimizado para alta escala
│   │   └── seed.go               # Carga inicial das disciplinas escolares e livros didáticos
│   ├── models/models.go          # Estruturas de dados de domínio
│   ├── curator/
│   │   ├── ai_evaluator.go       # Avaliador de IA (Gemini, OpenAI e Heurística Local)
│   │   └── worker.go             # Fila assíncrona de triagem de submissões
│   └── handlers/
│       ├── handlers.go           # Middleware de segurança HTTP
│       ├── books.go              # Catálogo, busca FTS5, leitura e detalhes
│       └── submissions.go        # Formulário e upload seguro de estudantes
├── web/
│   ├── embedded.go               # Empacotamento estático via go:embed
│   ├── static/                   # CSS puro, JS vanilla e favicon vetorial
│   └── templates/                # Templates HTML nativos seguros contra XSS
├── test/
│   └── security_test.go          # Testes automatizados de SQLi, XSS, concorrência e IA
├── Makefile                      # Automação de compilação, testes e execução
└── go.mod                        # Módulo Go e dependências mínimas
```

---

## 📜 Licença
Desenvolvido com foco no avanço educacional do **Colégio Betânia**. Código aberto sob licença Apache 2.0.

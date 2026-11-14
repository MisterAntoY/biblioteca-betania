<p align="center">
  <img src="docs/images/logo.png" alt="Complexo Escolar Privado Betânia - Zango 3" width="110" height="103" />
</p>

<h1 align="center">BiblioTânia</h1>

<p align="center">
  <strong>Biblioteca Digital Oficial do Complexo Escolar Privado Betânia · Zango 3, Luanda</strong><br>
  Plataforma educacional aberta de alto desempenho em Go, com SQLite FTS5, acervo curricular nacional do MED/INIDE e triagem autônoma por IA.
</p>

<p align="center">
  <a href="https://github.com/MisterAntoY/biblioteca-betania"><img src="https://shieldcn.dev/github/stars/MisterAntoY/biblioteca-betania.svg" alt="GitHub Stars" /></a>
  <img src="https://shieldcn.dev/static/Go/1.22+.svg?logo=go" alt="Go 1.22+" />
  <img src="https://shieldcn.dev/static/Acervo/155+Obras.svg?logo=book" alt="155 Obras" />
  <img src="https://shieldcn.dev/static/Motor/SQLite_FTS5.svg?logo=sqlite" alt="SQLite FTS5" />
  <a href="https://misterantoy.github.io/biblioteca-betania/"><img src="https://shieldcn.dev/static/Deploy/GitHub_Pages.svg?logo=githubpages" alt="GitHub Pages" /></a>
  <img src="https://shieldcn.dev/github/license/MisterAntoY/biblioteca-betania.svg" alt="Licença Apache 2.0" />
</p>

<p align="center">
  <a href="https://misterantoy.github.io/biblioteca-betania/"><strong>🌐 Acessar Portal Web Online</strong></a> ·
  <a href="SOBRE.md"><strong>📖 Manifesto Pedagógico & Histórico</strong></a> ·
  <a href="#-matriz-curricular"><strong>📚 Acervo Curricular</strong></a> ·
  <a href="#-como-executar"><strong>⚡ Execução Rápida</strong></a>
</p>

---

## ⚡ Métricas e Desempenho

| Métrica | Valor | Descrição |
| :--- | :--- | :--- |
| **Consumo de Memória (RAM)** | `< 20 MB` | Redução drástica em relação a runtimes tradicionais em Node.js / Next.js (que demandavam 300MB+) |
| **Tempo de Resposta (FTS5)** | `< 1 ms` | Consultas de texto integral em milissegundos sobre centenas de milhares de termos indexados |
| **Binário Único Compilado** | `~12 MB` | Servidor, templates, rotas e recursos embutidos nativamente via `go:embed` |
| **Concorrência e Mutex** | `Zero Race Conditions` | Transações SQLite protegidas via WAL e `sync.RWMutex`, validado com `go test -race` |
| **Dependências em Produção** | `0 (Zero)` | Executável auto-suficiente sem necessidade de Node.js, npm ou banco de dados externo |

---

## 🏛️ Visão e Arquitetura do Projeto

A **BiblioTânia** foi concebida sob medida para a comunidade pedagógica do **Complexo Escolar Privado Betânia - Zango 3** (Luanda, Angola), com uma arquitetura desenhada para suportar o marco de **100 milhões de recursos educacionais abertos (REA)**.

### Pilares Fundamentais:

1. **Capas Editoriais Reais e Oficiais (Proporção 3:4):**
   - **Clássicos da Literatura Angolana:** Capas originais das edições históricas (União dos Escritores Angolanos, Editorial Caminho, Leya, Dom Quixote) obtidas através de acervos públicos abertos do Open Library.
   - **Manuais Escolares Oficiais:** Fac-símile oficial fiel ao padrão do Ministério da Educação de Angola / INIDE, com insígnia da República de Angola, cores normalizadas por disciplina e o timbre do Colégio Betânia.
   - **Efeito de Estante Física:** Relevo de lombada curva sombreada e micro-interação tridimensional em CSS puro.

2. **Triagem Autônoma por Inteligência Artificial:**
   - Estudantes sugerem obras via `/submeter` (arquivos PDF/DOCX ou links abertos HTTPS).
   - O worker assíncrono em segundo plano avalia a adequação didática, moral e curricular através de modelos de IA com fallback local resiliente.
   - Transparência total através do Mural da IA (`/submissoes`).

3. **Portal Híbrido (GitHub Pages + Cloud VPS):**
   - **Frontend Estático SPA:** Hospedado gratuitamente no GitHub Pages (`https://misterantoy.github.io/biblioteca-betania/`) com busca instantânea em memória sobre `data/books.json`.
   - **Backend Go na VPS:** Executa a triagem de IA, persistência no SQLite FTS5 e uploads de ficheiros com suporte completo a CORS.

---

## 📚 Matriz Curricular Nacional de Angola (INIDE / MED)

O catálogo ativo reúne **155 obras didáticas e literárias** organizadas rigorosamente por ciclo e classe:

```
├── Ensino Primário (1ª à 6ª Classe)
│   ├── Língua Portuguesa (Alfabetização, Leitura e Gramática)
│   ├── Matemática (Aritmética, Geometria e Cálculo)
│   ├── Estudo do Meio (Higiene, Sociedade e Meio Ambiente)
│   ├── Ciências da Natureza (Ecologia e Seres Vivos)
│   └── História e Geografia de Angola
│
├── Ensino Secundário Geral — I Ciclo (7ª à 9ª Classe)
│   ├── Língua Portuguesa e Literatura Escolar
│   ├── Matemática (Álgebra, Geometria Plana e Estatística)
│   ├── Física Fundamental e Química Geral
│   ├── Biologia Celular e Educação Moral e Cívica
│   └── História de África e Geografia Geral
│
├── Ensino Secundário — II Ciclo / Pré-Universitário (10ª à 12ª Classe)
│   ├── CFB: Matemática Avançada, Física Moderna, Química Orgânica e Genética
│   ├── CEJ: Economia, Mercados, Noções de Direito e Ordem Jurídica Angolana
│   └── Ciências Humanas: Filosofia Política, Ética e Sociologia de Angola
│
├── Ensino Técnico-Profissional (Informática)
│   ├── TLP: Técnicas e Linguagens de Programação
│   ├── RCC: Redes de Comunicação e Computadores
│   ├── SEAC: Sistemas de Exploração e Arquitetura de Computadores
│   └── SIBD: Sistemas de Informação e Bases de Dados
│
├── Cânone da Literatura Angolana
│   ├── Pepetela (Mayombe, Yaka, Lueji, O Cão e os Caluandas, As Aventuras de Ngunga)
│   ├── José Luandino Vieira (Luuanda, Domingos Xavier, Nós os do Makulusu, Velhas Estórias)
│   ├── Agostinho Neto (Sagrada Esperança, Poemas)
│   ├── José Eduardo Agualusa (Teoria Geral do Esquecimento, O Vendedor de Passados)
│   ├── Ondjaki (Bom Dia Camaradas, Os Transparentes, AvóDezanove)
│   ├── Uanhenga Xitu (Mestre Tamoda, Bola com Feitiço, Manana, Vozes da Terra)
│   └── Paula Tavares, Manuel Rui, Viriato da Cruz, António Jacinto, Óscar Ribas
│
└── Línguas Nacionais & Apoio Comunitário
    ├── Kimbundu, Umbundu, Kikongo e Cokwe (Gramáticas e Dicionários Escolares)
    ├── Saúde Comunitária Preventiva (Malária, Nutrição Infantil e Primeiros Socorros)
    └── Agropecuária Sustentável (Mandioca, Horticultura Tropical e Solos)
```

---

## 🛡️ Segurança e Robustez Operacional

```
[Cliente Web / GitHub Pages]
          │  HTTPS / CORS Seguro
          ▼
[SecurityHeadersMiddleware]
  ├── Content-Security-Policy
  ├── X-Content-Type-Options: nosniff
  ├── X-Frame-Options: SAMEORIGIN
  └── Access-Control-Allow-Origin: *
          │
          ▼
   [Router / HTTP Mux]
   ├── /api/livros       ──> Consulta Parametrizada SQLite FTS5 (Zero SQLi)
   ├── /api/submissoes   ──> Upload Sanitizado (UUID, Sniffing MIME, máx 50MB)
   └── /api/stats        ──> Estatísticas em Tempo Real
          │
          ▼
[Curator Worker (IA)]  ──> Triagem Ética e Pedagógica em Segundo Plano
```

- **SQL Injection:** 100% das consultas utilizam *Prepared Statements* parametrizados com placeholders `?`.
- **Cross-Site Scripting (XSS):** O motor nativo `html/template` de Go aplica *auto-escaping* contextual obrigatório.
- **Race Conditions:** Mutex `sync.RWMutex` serializa gravações e viabiliza centenas de leituras simultâneas sem contenção de concorrência.
- **SSRF:** URLs submetidas por estudantes são verificadas para bloquear redes privadas e endereços loopback.

---

## ⚡ Como Executar

### Pré-requisitos
- **Go 1.22+** (com suporte a CGO para compilação nativa do SQLite)

### 1. Clonar o Repositório
```bash
git clone https://github.com/MisterAntoY/biblioteca-betania.git
cd biblioteca-betania
```

### 2. Executar a Suíte Completa de Testes
```bash
# Testes unitários e de integração
go test -v ./test/...

# Detector estrito de race conditions
go test -race ./test/...
```

### 3. Compilar e Iniciar
```bash
# Compilar binário otimizado
CGO_ENABLED=1 go build -ldflags="-s -w" -o bin/bibliotania cmd/server/main.go

# Iniciar o servidor na porta desejada (padrão: 8090)
PORT=8090 DB_PATH=./data/bibliotania.db ./bin/bibliotania
```
O servidor estará ativo em: **`http://localhost:8090`**.

---

## 📂 Estrutura do Repositório

```
biblioteca-betania/
├── cmd/
│   └── server/main.go            # Ponto de entrada, roteamento HTTP e graceful shutdown
├── docs/                         # Portal estático publicado no GitHub Pages
│   ├── css/style.css             # Estilos com identidade Betânia e capas 3:4
│   ├── data/books.json           # Base completa dos 155 livros para busca instantânea
│   ├── images/covers/            # 155 capas reais e oficiais em alta qualidade
│   ├── js/app.js                 # SPA cliente com busca e sincronização com a VPS
│   └── index.html                # Aplicação web estática
├── internal/
│   ├── config/config.go          # Variáveis de ambiente e portas
│   ├── database/                 # SQLite FTS5, migrações atômicas e seed curricular
│   ├── models/models.go          # Estruturas de dados de domínio
│   ├── curator/                  # Motor de triagem de IA e fila assíncrona
│   └── handlers/                 # Middlewares de segurança, rotas e APIs REST
├── web/                          # Servidor web embutido no binário Go
│   ├── embedded.go               # Empacotamento estático via go:embed all:static
│   ├── static/                   # CSS, favicon e capas originais
│   └── templates/                # Templates HTML nativos seguros contra XSS
├── test/
│   ├── security_test.go          # Testes de SQLi, XSS, concorrência e upload seguro
│   └── visualization_test.go     # Testes de visualização E2E e renderização de capas
├── SOBRE.md                      # Manifesto pedagógico e histórico institucional
├── Makefile                      # Automação de tarefas
└── go.mod                        # Módulo Go e dependências mínimas
```

---

## 📜 Licença e Governança

Desenvolvido para o **Complexo Escolar Privado Betânia - Zango 3** (Luanda, Angola).  
Código aberto sob a licença **Apache 2.0**.

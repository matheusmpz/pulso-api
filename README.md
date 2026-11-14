# Pulso API

API do **Pulso**, o dashboard de BI da Fábrica de Software (Grupo 7, "Estratégia / Dashboard de BI"). Mostra, em tempo real, a saúde operacional e financeira: andamento dos projetos, vendas e custo de IA no desenvolvimento.

```
[Gerador de eventos]  →  [PostgreSQL pulso_db]  ←(polling)  [API Go]  →(REST + WebSocket)→  [Front]
     Python                                                  este projeto
```

- O gerador (Python) só faz `INSERT` e nunca chama esta API.
- Esta API só faz `SELECT` nas tabelas de dados, detecta dado novo por polling e envia as mudanças pelo WebSocket.
- O front só desenha o que a API manda. Toda a regra de cálculo está aqui.

> **Status:** em construção. Por enquanto existem a conexão com o banco e o `GET /health`.

## Stack

- **Go 1.25** + Gin + gin-contrib/cors
- **PostgreSQL** via `database/sql` + `lib/pq` (SQL puro, sem ORM)
- **JWT** (`golang-jwt/jwt/v5`, HS256) e **bcrypt**
- **godotenv** para variáveis de ambiente
- **gorilla/websocket** para o tempo real
- **swaggo** para o Swagger

## Estrutura

```
pulso-api/
├── cmd/api/main.go      → ponto de entrada (.env, Gin, CORS, shutdown gracioso)
├── routes/              → registro de rotas (um arquivo por domínio)
├── controllers/         → recebe requisições e devolve respostas
├── repositories/        → queries SQL por domínio (interface + implementação)
├── models/              → entidades, filtros e formatos dos cards
├── middlewares/         → validação do JWT
├── services/
│   ├── calculos/        → um Calculador por card
│   └── tempo_real/      → hub do WebSocket e poller de eventos
├── db/
│   ├── migrations/      → scripts PULAPI-XXX_script.sql (aplicados manualmente)
│   ├── schema_v2.sql    → DDL (já executado; só referência)
│   └── seed.sql         → seed (já executado; só referência)
├── utils/               → respostas padronizadas e helpers
└── docs/                → Swagger gerado pelo swag (não editar à mão)
```

## Rodando localmente

Pré-requisitos: Go 1.25+ e o banco `pulso_db` com `db/schema_v2.sql` e `db/seed.sql` executados.

```bash
# 1. Copiar variáveis de ambiente e preencher os valores
cp .env.example .env

# 2. Instalar dependências
go mod tidy

# 3. Rodar
go run ./cmd/api
```

O servidor encerra de forma graciosa com `Ctrl+C` (SIGINT) ou SIGTERM.

### Debug no VS Code

O arquivo `.vscode/launch.json` traz a configuração **Pulso-Api-Debugger**, que roda `cmd/api` lendo as variáveis do `.env` (`envFile`). Requer a extensão Go e o Delve (`dlv`).

## Variáveis de ambiente

| Variável | Obrigatória | Padrão | Descrição |
|---|---|---|---|
| `PORT` | não | `8080` | Porta HTTP |
| `GIN_MODE` | não | `release` | `debug` ou `release` |
| `DB_HOST` | sim | — | Host do PostgreSQL |
| `DB_PORT` | não | `5432` | Porta do PostgreSQL |
| `DB_USER` | sim | — | Usuário do banco |
| `DB_PASSWORD` | sim | — | Senha do banco |
| `DB_NAME` | sim | `pulso_db` | Nome do banco |
| `DB_SSLMODE` | não | `disable` | `sslmode` do `lib/pq` |
| `JWT_SECRET` | sim | — | Chave de assinatura do JWT |
| `JWT_EXPIRA_HORAS` | não | `24` | Validade do token |
| `CORS_ORIGINS` | não | `http://localhost:3000` | Origens permitidas (separadas por vírgula), também usadas no WebSocket |
| `POLL_INTERVAL_MS` | não | `2000` | Intervalo do polling de eventos |
| `APP_TIMEZONE` | não | `America/Sao_Paulo` | Fuso dos limites de dia |
| `SWAGGER_ENABLED` | não | `true` | Serve o Swagger UI em `/swagger/index.html` |

Se faltar alguma variável obrigatória, a API não sobe e informa quais faltam:

```
Configuração inválida: variáveis de ambiente obrigatórias não definidas: DB_HOST, JWT_SECRET
```

Se o banco não responder na inicialização (prazo de 5 s), a API também não sobe:

```
Erro ao conectar no banco (confira as variáveis DB_*): ...
```

## Testes

```bash
go test ./...
```

Os testes de unidade não precisam de banco.

## Branches

- `main` → produção
- `develop` → desenvolvimento
- `PULAPI-XXX` → uma branch por etapa, criada a partir da `develop`

Commits no formato `PULAPI-XXX - Descrição simples do que foi feito`. Mudanças de banco ficam em `db/migrations/PULAPI-XXX_script.sql` e são aplicadas manualmente.

---

## Rotas

As rotas são documentadas aqui à medida que forem entregues (método, caminho, body, resposta de sucesso e tabela de erros).

Padrão de resposta das rotas da API:

```json
{ "sucesso": true,  "dados": { } }
{ "sucesso": false, "erro": "mensagem" }
```

### Rotas públicas

---

#### `GET /health`
Verifica se a API está no ar e se o banco responde (ping com prazo de 2 s). Não usa o envelope `sucesso`/`dados`, para ser lido direto por monitores e orquestradores.

**Resposta de sucesso `200`**
```json
{ "status": "ok", "banco": "ok" }
```

**Erros**

| Status | Quando | Corpo |
|---|---|---|
| `503` | O banco não respondeu ao ping | `{ "status": "erro", "banco": "indisponivel" }` |

---

## Decisões de implementação

- **Module path:** `pulso-api` (nenhum foi informado). Para renomear: `go mod edit -module <novo>` e ajuste dos imports.
- **Versão do Go no `go.mod`:** `go 1.25.0`, conforme a especificação, mesmo com o projeto sendo desenvolvido com um Go mais novo.
- **Pastas vazias:** recebem um `.gitkeep` para existirem no repositório desde a primeira etapa. O arquivo é removido quando a pasta ganhar código.
- **`main.go` com `http.Server`:** em vez de `r.Run`, o servidor usa `http.Server` com `Shutdown` para o encerramento gracioso exigido (SIGINT/SIGTERM).
- **Padrões no `main.go`:** `GIN_MODE` vazio vira `release` e `CORS_ORIGINS` vazio vira `http://localhost:3000` (o `gin-contrib/cors` entra em pânico com lista vazia).
- **CORS:** métodos liberados são só `GET`, `POST` e `OPTIONS`, porque a API não tem rotas de escrita além do login.
- **`.gitattributes` com LF:** força quebra de linha LF nos arquivos, porque o `gofmt` exige LF e o `core.autocrlf` do Windows trocava por CRLF no checkout.
- **Carregamento do `.env`:** feito só no `main.go`, antes de validar as variáveis. O `db.GetDB()` apenas lê o ambiente já carregado.
- **Variáveis obrigatórias:** `DB_HOST`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` e `JWT_SECRET` já são exigidas desde esta etapa, mesmo com o JWT entrando só na autenticação. `DB_PORT` e `DB_SSLMODE` têm padrão (`5432` e `disable`).
- **String de conexão:** montada no formato `chave='valor'` com escape de aspas e barras, para senhas com espaço ou aspas não quebrarem a conexão.
- **`/health` fora do envelope:** devolve `{ status, banco }` direto, como pede a especificação. O controller recebe uma interface `Pinger` (o `*sql.DB` a satisfaz), para ser testado sem banco.
- **Pool fechado no encerramento:** depois do `Shutdown` do servidor, o `main.go` fecha o `*sql.DB`.
- **`.gitignore`:** além do pedido, ignora `.env.*` (exceto `.env.example`), arquivos de chave/certificado e o binário do Delve (`__debug_bin*`). Em `.vscode/`, só o `launch.json` é versionado.

---

## Histórico de etapas

- `PULAPI-001` — estrutura do projeto: módulo Go, pastas, `.gitignore`, `.env.example`, debugger do VS Code, `main.go` mínimo com shutdown gracioso e README inicial.
- `PULAPI-002` — conexão com o banco (pool singleton com `Ping`), validação das variáveis obrigatórias, padrão de resposta, `GET /health` (503 se o banco cair), `.gitattributes` com LF e testes de unidade.

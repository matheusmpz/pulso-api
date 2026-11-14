# Pulso API

API do **Pulso**, o dashboard de BI da Fábrica de Software (Grupo 7, "Estratégia / Dashboard de BI"). Mostra, em tempo real, a saúde operacional e financeira: andamento dos projetos, vendas e custo de IA no desenvolvimento.

```
[Gerador de eventos]  →  [PostgreSQL pulso_db]  ←(polling)  [API Go]  →(REST + WebSocket)→  [Front]
     Python                                                  este projeto
```

- O gerador (Python) só faz `INSERT` e nunca chama esta API.
- Esta API só faz `SELECT` nas tabelas de dados, detecta dado novo por polling e envia as mudanças pelo WebSocket.
- O front só desenha o que a API manda. Toda a regra de cálculo está aqui.

> **Status:** em construção. Nesta etapa existe apenas o esqueleto do projeto; o servidor sobe, mas ainda não há rotas.

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

A validação das variáveis obrigatórias (falha na inicialização com mensagem clara) entra junto com a conexão com o banco e a autenticação.

## Branches

- `main` → produção
- `develop` → desenvolvimento
- `PULAPI-XXX` → uma branch por etapa, criada a partir da `develop`

Commits no formato `PULAPI-XXX - Descrição simples do que foi feito`. Mudanças de banco ficam em `db/migrations/PULAPI-XXX_script.sql` e são aplicadas manualmente.

---

## Rotas

Ainda não há rotas. Elas serão documentadas aqui à medida que forem entregues (método, caminho, body, resposta de sucesso e tabela de erros).

---

## Decisões de implementação

- **Module path:** `pulso-api` (nenhum foi informado). Para renomear: `go mod edit -module <novo>` e ajuste dos imports.
- **Versão do Go no `go.mod`:** `go 1.25.0`, conforme a especificação, mesmo com o projeto sendo desenvolvido com um Go mais novo.
- **Pastas vazias:** recebem um `.gitkeep` para existirem no repositório desde a primeira etapa. O arquivo é removido quando a pasta ganhar código.
- **`main.go` com `http.Server`:** em vez de `r.Run`, o servidor usa `http.Server` com `Shutdown` para o encerramento gracioso exigido (SIGINT/SIGTERM).
- **Padrões no `main.go`:** `GIN_MODE` vazio vira `release` e `CORS_ORIGINS` vazio vira `http://localhost:3000` (o `gin-contrib/cors` entra em pânico com lista vazia).
- **CORS:** métodos liberados são só `GET`, `POST` e `OPTIONS`, porque a API não tem rotas de escrita além do login.
- **`.gitignore`:** além do pedido, ignora `.env.*` (exceto `.env.example`), arquivos de chave/certificado e o binário do Delve (`__debug_bin*`). Em `.vscode/`, só o `launch.json` é versionado.

---

## Histórico de etapas

- `PULAPI-001` — estrutura do projeto: módulo Go, pastas, `.gitignore`, `.env.example`, debugger do VS Code, `main.go` mínimo com shutdown gracioso e README inicial.

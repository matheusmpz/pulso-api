# Pulso API

API do **Pulso**, o dashboard de BI da Fábrica de Software (Grupo 7, "Estratégia / Dashboard de BI"). Mostra, em tempo real, a saúde operacional e financeira: andamento dos projetos, vendas e custo de IA no desenvolvimento.

```
[Gerador de eventos]  →  [PostgreSQL pulso_db]  ←(polling)  [API Go]  →(REST + WebSocket)→  [Front]
     Python                                                  este projeto
```

- O gerador (Python) só faz `INSERT` e nunca chama esta API.
- Esta API só faz `SELECT` nas tabelas de dados, detecta dado novo por polling e envia as mudanças pelo WebSocket.
- O front só desenha o que a API manda. Toda a regra de cálculo está aqui.

> **Status:** em construção. Por enquanto existem o `GET /health`, a autenticação (`POST /auth/login` e `GET /auth/me`) e os metadados dos dashboards com controle de acesso (`GET /dashboards` e `GET /dashboards/:chave`).

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

## Rodando com Docker

Sobe a API e um PostgreSQL já com o schema e o seed, sem instalar Go nem Postgres na máquina.

### Pré-requisitos

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (Windows/Mac) ou Docker Engine com o plugin Compose (Linux). Confira com `docker compose version`.
- Um arquivo `.env` na raiz, criado a partir do `.env.example`, com `DB_USER`, `DB_PASSWORD`, `DB_NAME` e `JWT_SECRET` preenchidos.

### Arquivos

| Arquivo | Para que serve |
|---|---|
| `Dockerfile` | Compila a API em `golang:1.25` e gera uma imagem pequena em `alpine`, rodando com usuário sem privilégio e com `HEALTHCHECK` em `/health` |
| `.dockerignore` | O que não entra no build (`.git`, `.env`, `CLAUDE.md`, `Estrutura.md`, `.vscode`, binários, logs) |
| `docker-compose.yml` | Serviços `db` (Postgres 16) e `api`, ligados na mesma rede |

### Passo a passo

```bash
# 1. Criar o .env e preencher DB_USER, DB_PASSWORD, DB_NAME e JWT_SECRET
cp .env.example .env

# 2. Construir a imagem e subir os dois serviços
docker compose up --build

# 3. Em outro terminal, testar
curl http://localhost:8080/health
# {"status":"ok","banco":"ok"}
```

Como funciona:

- O serviço `db` usa `DB_USER`, `DB_PASSWORD` e `DB_NAME` do `.env` para criar o usuário e o banco na **primeira** subida. Em seguida, executa `db/schema_v2.sql` e `db/seed.sql`, que ficam montados como `01-schema.sql` e `02-seed.sql`.
- O serviço `api` só sobe depois que o banco passa no `pg_isready`. Ele lê todas as variáveis do `.env`, mas o compose força `DB_HOST=db` e `DB_PORT=5432` (nome do serviço e porta interna na rede do Docker).
- A API fica em `localhost:<PORT>` (padrão 8080). O banco fica acessível na sua máquina em `localhost:5433`, ou na porta definida em `DB_PORTA_DOCKER`, para usar DBeaver ou `psql`.
- Os dados ficam no volume `pulso_pgdata` e sobrevivem a `docker compose down`.

### Comandos úteis

| Comando | O que faz |
|---|---|
| `docker compose up --build` | Reconstrói a imagem da API e sobe tudo (use depois de mudar o código) |
| `docker compose up -d` | Sobe em segundo plano (libera o terminal) |
| `docker compose ps` | Mostra os containers e se estão `healthy` |
| `docker compose logs -f api` | Acompanha os logs da API (`Ctrl+C` sai só dos logs) |
| `docker compose down` | Para e remove os containers. **Mantém** os dados do banco |
| `docker compose down -v` | Para, remove os containers **e apaga o volume** do banco (o schema e o seed rodam de novo na próxima subida) |
| `docker compose exec db psql -U <DB_USER> -d <DB_NAME>` | Abre o `psql` dentro do container do banco |

### Carregar os dados de exemplo

O script `db/dev_dados_exemplo.sql` (só para desenvolvimento) será entregue numa etapa futura. Com o banco no ar, carregue-o assim, trocando `<DB_USER>` e `<DB_NAME>` pelos valores do seu `.env`:

```bash
docker compose exec -T db psql -U <DB_USER> -d <DB_NAME> < db/dev_dados_exemplo.sql
```

O `-T` desliga o terminal interativo, e isso permite redirecionar o arquivo com `<`. No PowerShell, que não aceita `<`, use:

```powershell
Get-Content db/dev_dados_exemplo.sql | docker compose exec -T db psql -U <DB_USER> -d <DB_NAME>
```

### Problemas comuns

- **Porta em uso** (`Bind for 0.0.0.0:8080 failed: port is already allocated`): outro programa usa a porta. Troque `PORT` (API) ou `DB_PORTA_DOCKER` (banco) no `.env`, ou pare o programa que a ocupa. No Windows, `netstat -ano | findstr :8080` mostra o PID.
- **O schema ou o seed não rodaram / mudei o SQL e nada aconteceu:** os scripts de `/docker-entrypoint-initdb.d` só rodam quando o volume é criado **vazio**. Se o volume já existe, eles são ignorados. Para recriar do zero (apaga os dados): `docker compose down -v` e `docker compose up --build`. O mesmo vale se você trocar `DB_USER`, `DB_PASSWORD` ou `DB_NAME` depois da primeira subida.
- **`required variable DB_USER is missing a value`** (ou `DB_PASSWORD`, `DB_NAME`): falta preencher a variável no `.env`.
- **A API reinicia em loop com `Configuração inválida`:** falta alguma variável obrigatória no `.env`, em geral o `JWT_SECRET`. Veja com `docker compose logs api`.
- **A API não conecta no banco:** confira se você não está rodando a API fora do Docker com `DB_HOST=db`. Fora do compose, use `DB_HOST=localhost` e `DB_PORT=5433` para usar o banco do container.
- **Horário dos logs em UTC:** o container usa UTC no relógio do sistema. Os cálculos de dia usam `APP_TIMEZONE`, não o fuso do container.

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
| `DB_PORTA_DOCKER` | não | `5433` | Porta do Postgres do docker compose na sua máquina (só no compose) |
| `JWT_SECRET` | sim | — | Chave de assinatura do JWT. Gere uma aleatória: `openssl rand -hex 32` |
| `JWT_EXPIRA_HORAS` | não | `24` | Validade do token, em horas inteiras maiores que zero |
| `CORS_ORIGINS` | não | `http://localhost:3000` | Origens permitidas (separadas por vírgula), também usadas no WebSocket |
| `POLL_INTERVAL_MS` | não | `2000` | Intervalo do polling de eventos |
| `APP_TIMEZONE` | não | `America/Sao_Paulo` | Fuso dos limites de dia e dos timestamps das respostas (ex.: `expira_em`) |
| `SWAGGER_ENABLED` | não | `true` | Serve o Swagger UI em `/swagger/index.html` |

Se faltar alguma variável obrigatória, ou se `JWT_EXPIRA_HORAS` ou `APP_TIMEZONE` forem inválidos, a API não sobe e informa o motivo:

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

#### `POST /auth/login`
Autentica um usuário **ativo** com e-mail e senha e devolve um JWT (HS256). O e-mail não diferencia maiúsculas. Os usuários vêm do seed; não há cadastro pela API.

**Body**
```json
{ "email": "diretoria@pulso.local", "senha": "sua_senha" }
```

**Resposta de sucesso `200`**
```json
{
  "sucesso": true,
  "dados": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expira_em": "2026-11-15T12:00:00-03:00",
    "usuario": {
      "id": "5b1f8a2e-3c4d-4e5f-8a9b-0c1d2e3f4a5b",
      "nome": "Diretoria (dev)",
      "email": "diretoria@pulso.local",
      "perfil": "diretoria"
    }
  }
}
```

**Erros**

| Status | Quando | Mensagem |
|---|---|---|
| `400` | JSON inválido, ou `email`/`senha` ausentes ou vazios | `Informe e-mail e senha.` |
| `401` | E-mail inexistente, senha errada ou usuário inativo (sempre a mesma mensagem) | `E-mail ou senha inválidos` |
| `500` | Erro no banco ou ao gerar o token | `Erro interno. Tente novamente mais tarde.` |

### Rotas protegidas

Enviam o token no cabeçalho:

```
Authorization: Bearer <token>
```

Erros comuns a todas as rotas protegidas:

| Status | Quando | Mensagem |
|---|---|---|
| `401` | Cabeçalho ausente ou fora do formato `Bearer <token>` | `Token não informado.` |
| `401` | Token com assinatura errada, adulterado, de outro emissor ou expirado | `Token inválido ou expirado.` |

---

#### `GET /auth/me`
Dados do usuário dono do token e do perfil dele.

**Resposta de sucesso `200`**
```json
{
  "sucesso": true,
  "dados": {
    "id": "5b1f8a2e-3c4d-4e5f-8a9b-0c1d2e3f4a5b",
    "nome": "Diretoria (dev)",
    "email": "diretoria@pulso.local",
    "perfil": {
      "id": 1,
      "nome": "diretoria",
      "descricao": "Acesso a todos os dashboards e cards, incluindo vendas."
    }
  }
}
```

**Erros**

| Status | Quando | Mensagem |
|---|---|---|
| `401` | Erros de token (tabela acima) | — |
| `401` | O token é válido, mas o usuário foi desativado | `Sessão inválida. Faça login novamente.` |
| `500` | Erro no banco | `Erro interno. Tente novamente mais tarde.` |

---

### Controle de acesso por perfil

A tabela `perfil_cards` define quais cards cada perfil vê. Tudo é aplicado no back-end:

- Um **dashboard** aparece para o perfil se estiver ativo e tiver ao menos um card **ativo e permitido**.
- Um **card** aparece se estiver ativo e for permitido ao perfil.
- Recurso inexistente ou inativo → **404**. Recurso que existe, mas o perfil não pode ver → **403** (`Você não tem acesso a este recurso`). A existência é verificada antes da permissão: um dashboard inativo dá 404 para todos.

No seed, `diretoria` vê os 41 cards (projetos, vendas e IA) e `lider_tecnologia` vê só os 15 de projetos.

---

#### `GET /dashboards`
Dashboards que o perfil do usuário pode ver, com a quantidade de cards visíveis em cada um, ordenados por `id`. Se o perfil não vê nenhum, a lista vem vazia (`[]`).

**Resposta de sucesso `200`** (perfil `lider_tecnologia`)
```json
{
  "sucesso": true,
  "dados": {
    "dashboards": [
      {
        "id": 1,
        "chave": "projetos",
        "titulo": "Projetos",
        "descricao": "Andamento dos projetos em desenvolvimento: progresso, horas por etapa, sprints e eficiência, por projeto, time ou sprint.",
        "total_cards": 15
      }
    ]
  }
}
```

**Erros**

| Status | Quando | Mensagem |
|---|---|---|
| `401` | Erros de token | — |
| `500` | Erro no banco | `Erro interno. Tente novamente mais tarde.` |

---

#### `GET /dashboards/:chave`
O dashboard e a lista dos cards que o perfil pode ver, **sem os dados calculados**, ordenados por `codigo`. A `descricao` de cada card é o texto do modal de informação do front. `chave` é `projetos`, `vendas` ou `ia`.

**Resposta de sucesso `200`** (`GET /dashboards/projetos`, resumida)
```json
{
  "sucesso": true,
  "dados": {
    "id": 1,
    "chave": "projetos",
    "titulo": "Projetos",
    "descricao": "Andamento dos projetos em desenvolvimento: ...",
    "cards": [
      {
        "codigo": "proj-01",
        "titulo": "Progresso",
        "descricao": "Mostra quanto do trabalho planejado já foi concluído. ...",
        "tipo": "numero"
      },
      {
        "codigo": "proj-02",
        "titulo": "Horas",
        "descricao": "Mostra quanto das horas estimadas já foi consumido. ...",
        "tipo": "numero"
      }
    ]
  }
}
```

**Erros**

| Status | Quando | Mensagem |
|---|---|---|
| `401` | Erros de token | — |
| `403` | O dashboard existe e está ativo, mas o perfil não vê nenhum card dele | `Você não tem acesso a este recurso` |
| `404` | Dashboard inexistente ou inativo | `Dashboard não encontrado.` |
| `500` | Erro no banco | `Erro interno. Tente novamente mais tarde.` |

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
- **Docker antes da autenticação:** a etapa de Docker (`PULAPI-003`) veio antes do login a pedido do dono do projeto. A autenticação passa a ser a próxima etapa.
- **Porta do banco no compose configurável:** o padrão é `5433:5432`, mas o lado do host pode mudar por `DB_PORTA_DOCKER`, para resolver colisão de porta sem editar o compose.
- **`DB_PORT=5432` forçado no compose, junto com `DB_HOST=db`:** o `.env` local costuma ter a porta do Postgres da máquina, e dentro da rede do Docker a porta é sempre a interna.
- **Credenciais do Postgres do container:** `POSTGRES_USER`, `POSTGRES_PASSWORD` e `POSTGRES_DB` são preenchidas com as mesmas `DB_USER`, `DB_PASSWORD` e `DB_NAME` do `.env`, para a API e o banco nunca divergirem. O compose para com mensagem clara se alguma faltar.
- **Imagem final `alpine:3.21`:** versão fixa, para o build ser reproduzível. O healthcheck usa o `wget` que já vem no alpine.
- **Configuração do JWT injetada:** `ConfigJWT` (segredo e validade) é lida uma vez no `main.go` e passada para `routes.Register`, o controller e o `AuthMiddleware(cfg)`. Isso permite testar sem variável de ambiente. `JWT_EXPIRA_HORAS` inválido impede a API de subir.
- **Validação do token:** aceita só HS256, exige expiração e confere o emissor (`iss = pulso-api`). Além dos claims pedidos (`usuario_id`, `perfil_id`, `perfil`, `email`), o token leva `sub` (id do usuário) e `iat`.
- **Login sem revelar quais e-mails existem:** além da mensagem única, quando o e-mail não existe a API compara a senha com um hash fictício. Assim o tempo de resposta fica igual ao de uma senha errada (medido: ~52 ms nos dois casos).
- **400 no login:** JSON inválido ou campos vazios devolvem 400 (`Informe e-mail e senha.`), porque é erro de formato, não de credencial. O formato do e-mail não é validado: um e-mail malformado simplesmente não existe e cai no 401.
- **`/auth/me` consulta o banco:** além de validar o token, busca o usuário ativo. Um usuário desativado depois do login recebe 401 (`Sessão inválida`), mesmo com o token ainda válido. As outras rotas protegidas confiam só no token até expirar.
- **Fuso da aplicação:** `APP_TIMEZONE` é carregado na inicialização (`utils.ConfigurarFuso`) e os timestamps das respostas saem com o offset do fuso (ex.: `-03:00`). A base de fusos vai embutida no binário (`time/tzdata`), então não depende do sistema operacional nem do container.
- **`expira_em` em segundos:** truncado para segundos, para bater com o `exp` gravado no JWT.
- **Regra de acesso isolada:** `utils.AvaliarAcesso(existe, ativo, permitido)` decide entre 404, 403 e permitido. É uma função pura, testada sem banco, e vai ser reaproveitada pelos cards. Nos dashboards, "permitido" quer dizer que o perfil vê ao menos um card ativo dele.
- **Permissões nos repositories de dashboard e card:** a checagem de `perfil_cards` vai como `JOIN` nas próprias consultas (`ListarVisiveis`), sem um `PermissaoRepository` separado por enquanto. Ele entra se a checagem de um card isolado (`/cards/:codigo/dados`) precisar.
- **Formato de `GET /dashboards`:** `dados.dashboards` é uma lista (nunca `null`), sem paginação, porque são no máximo 3 itens. Ordenada por `id`, que segue a ordem do seed: projetos, vendas e IA.
- **`GET /dashboards/:chave` sem cards permitidos dá 403**, e não lista vazia, como pede a regra "pedir dashboard sem permissão → 403".
- **`.gitignore`:** além do pedido, ignora `.env.*` (exceto `.env.example`), arquivos de chave/certificado e o binário do Delve (`__debug_bin*`). Em `.vscode/`, só o `launch.json` é versionado.

---

## Histórico de etapas

- `PULAPI-001` — estrutura do projeto: módulo Go, pastas, `.gitignore`, `.env.example`, debugger do VS Code, `main.go` mínimo com shutdown gracioso e README inicial.
- `PULAPI-002` — conexão com o banco (pool singleton com `Ping`), validação das variáveis obrigatórias, padrão de resposta, `GET /health` (503 se o banco cair), `.gitattributes` com LF e testes de unidade.
- `PULAPI-003` — Docker: `Dockerfile` multi-stage (usuário sem privilégio, `HEALTHCHECK`), `.dockerignore`, `docker-compose.yml` com Postgres 16 (schema e seed na primeira subida, porta 5433) e API, `.env.example` atualizado e seção "Rodando com Docker".
- `PULAPI-004` — autenticação: `POST /auth/login` (bcrypt, mensagem genérica, tempo constante), `GET /auth/me`, geração e validação de JWT HS256, `AuthMiddleware` (injeta `usuario_id`, `perfil_id` e `perfil`), fuso da aplicação (`APP_TIMEZONE`) e testes.
- `PULAPI-005` — dashboards, cards e permissões: `GET /dashboards` (só dashboards com cards ativos e permitidos, com `total_cards`), `GET /dashboards/:chave` (cards visíveis, sem dados), regra de acesso 404/403 e testes.

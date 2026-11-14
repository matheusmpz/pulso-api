# Pulso API

API do **Pulso**, o dashboard de BI da Fábrica de Software (Grupo 7, "Estratégia / Dashboard de BI"). Mostra, em tempo real, a saúde operacional e financeira: andamento dos projetos, vendas e custo de IA no desenvolvimento.

```
[Gerador de eventos]  →  [PostgreSQL pulso_db]  ←(polling)  [API Go]  →(REST + WebSocket)→  [Front]
     Python                                                  este projeto
```

- O gerador (Python) só faz `INSERT` e nunca chama esta API.
- Esta API só faz `SELECT` nas tabelas de dados, detecta dado novo por polling e envia as mudanças pelo WebSocket.
- O front só desenha o que a API manda. Toda a regra de cálculo está aqui.

> **Status:** em construção. Já existem o `GET /health`, a autenticação (`POST /auth/login` e `GET /auth/me`), os metadados dos dashboards com controle de acesso (`GET /dashboards` e `GET /dashboards/:chave`), as opções de filtro (`GET /filtros/opcoes`) e as rotas de dados dos cards (`GET /dashboards/:chave/dados` e `GET /cards/:codigo/dados`), todas documentadas no [Swagger](#swagger). Os 41 cards estão registrados, mas os **cálculos ainda estão pendentes**: até cada grupo ser implementado, o card volta com `erro`.

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

O script `db/dev_dados_exemplo.sql` (**só para desenvolvimento**) cria dados fictícios e coerentes dos últimos ~90 dias, para todos os cards terem o que mostrar. Veja o que ele cria em [Dados de exemplo](#dados-de-exemplo).

Com o banco no ar e **ainda sem dados**, carregue-o trocando `<DB_USER>` e `<DB_NAME>` pelos valores do seu `.env`:

```bash
docker compose exec -T db psql -v ON_ERROR_STOP=1 -U <DB_USER> -d <DB_NAME> < db/dev_dados_exemplo.sql
```

O `-T` desliga o terminal interativo, e isso permite redirecionar o arquivo com `<`. No PowerShell, que não aceita `<`, use:

```powershell
Get-Content db/dev_dados_exemplo.sql | docker compose exec -T db psql -v ON_ERROR_STOP=1 -U <DB_USER> -d <DB_NAME>
```

Também dá para rodar pelo **DBeaver**: abra o arquivo num editor SQL da conexão "Pulso (Docker)" e use **Executar script** (`Alt+X`).

O script roda numa transação só (entra tudo ou nada) e **se recusa a rodar** se o banco já tiver dados de negócio. Para recarregar do zero, use `docker compose down -v` e `docker compose up -d db`; o schema e o seed rodam de novo, e aí você carrega o script.

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
| `SWAGGER_ENABLED` | não | `true` | Serve o Swagger UI em `/swagger/index.html` (`true`/`false`; valor inválido impede a API de subir) |

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

### Testes de integração

Rodam as consultas no PostgreSQL de verdade. Ficam atrás da build tag `integration` e são **pulados** se as variáveis `DB_*` não existirem. Eles leem o ambiente ou o `.env` da raiz.

```bash
# Só SELECT: confere se o SQL dos repositories é válido no banco
go test -tags integration ./repositories/

# Também confere a semântica dos conjuntos do escopo com um cenário de teste.
# Insere dados DENTRO DE UMA TRANSAÇÃO e faz ROLLBACK: nada fica gravado,
# mas as sequências (ids) avançam. Prefira um banco descartável.
PULSO_TESTE_FIXTURE=1 go test -tags integration ./repositories/
```

## Swagger

Com a API rodando e `SWAGGER_ENABLED=true` (o padrão), a documentação interativa fica em:

**http://localhost:8080/swagger/index.html** (`/swagger` também funciona e redireciona para lá)

Para testar uma rota protegida pelo navegador:

1. Abra `POST /auth/login` → **Try it out**, preencha e-mail e senha → **Execute**.
2. Copie o valor de `token` da resposta.
3. Clique em **Authorize** (cadeado no topo) e informe `Bearer <token>`, com a palavra `Bearer`, um espaço e o token.
4. Pronto: as rotas com cadeado passam a enviar o token. Ele continua salvo se você recarregar a página.

Com `SWAGGER_ENABLED=false`, as rotas `/swagger` deixam de existir (404).

### Regenerar a documentação

A pasta `docs/` é **gerada** a partir dos comentários `@Summary`, `@Param`, `@Router` etc. dos handlers e do `cmd/api/main.go`. Não edite à mão. Depois de criar ou mudar uma rota:

```bash
# Uma vez: instalar a CLI do swag (mesma versão da biblioteca no go.mod)
go install github.com/swaggo/swag/cmd/swag@v1.16.4

# Regenerar
swag init -g cmd/api/main.go -o docs
```

O `docs/` é versionado e entra na imagem Docker, então o Docker não precisa do `swag` para compilar.

## Dados de exemplo

`db/dev_dados_exemplo.sql` cria um cenário fictício, com datas relativas a **hoje** (fuso `America/Sao_Paulo`), para desenvolver e demonstrar os cards. Ele também é a **referência para o gerador Python**: mostra a ordem dos INSERTs, as regras de integridade e o formato esperado dos dados, com comentários explicando cada bloco.

| Área | O que tem |
|---|---|
| Cadastros | 3 clientes; Time Alfa e Time Beta, com 3 desenvolvedores cada |
| Funil | 20 oportunidades em 4 origens (`site`, `indicacao`, `linkedin`, `evento`): 5 ganhas, 7 perdidas (4 motivos) e 8 em aberto em várias etapas; 29 reuniões |
| Orçamentos | 16, alguns com versão 2 (renegociação); status pendente, em revisão, aprovada e recusada, e um sem nenhum evento (fica "pendente" por padrão) |
| Projetos e vendas | 5 projetos (2 web, mobile, api, integracao; em andamento, concluído e pausado), cada um com sua venda e ligado ao orçamento ganho |
| Sprints | 6 de 14 dias por time: 4 passadas, a atual e uma futura |
| Atividades | 37, com todos os tipos e complexidades: concluídas, em andamento, abertas, sem nenhum evento (backlog e sprint futura) e várias estourando a estimativa (até 180%) |
| Horas | ~150 lançamentos nas três etapas, com o custo/hora de cada desenvolvedor |
| IA | ~250 consumos em 3 modelos (`claude-sonnet`, `gemini-flash`, `gpt-4o-mini`), ligados às atividades e também sem atividade; quota dos projetos em 144% (crítica), 84% (atenção) e abaixo de 80%; dois picos de consumo |
| Metas | Meta de receita do mês atual e dos 2 anteriores |

### Guia do gerador Python

O gerador simula os outros grupos inserindo dados continuamente. Ele **nunca chama esta API**. As regras completas estão no topo de `db/dev_dados_exemplo.sql`, e o schema está em [`db/schema_v2.sql`](db/schema_v2.sql). Resumo:

1. **Só `INSERT`**, nunca `UPDATE` nem `DELETE`. Mudança de estado é uma linha nova em `atividade_eventos`, `oportunidade_eventos` ou `orcamento_eventos`. O estado atual é o do último evento (maior `ocorrido_em`; empate, maior `id`). Sem evento, vale o estado inicial: atividade `aberta`, oportunidade `lead`, orçamento `pendente`.
2. **Autocommit ou transações curtas.** A API detecta dado novo pelo `MAX(id)` de cada tabela, a cada `POLL_INTERVAL_MS`. Uma transação longa pode tornar visível um id menor depois de um maior já lido, e esse evento se perde.
3. **Ordem dos INSERTs:** clientes → times → desenvolvedores; oportunidades → eventos do funil e reuniões; orçamentos → eventos de status; projetos (com `orcamento_id` do orçamento ganho) → vendas; sprints → atividades → eventos de atividade; horas e tokens.
4. **Chaves compostas:** a sprint da atividade é do mesmo time; horas e tokens com atividade são do mesmo projeto da atividade. `consumo_tokens.atividade_id` pode ser nulo (uso geral no projeto).
5. `motivo_perda` existe **se e somente se** a etapa é `perdida`. `ganha` e `perdida` são finais.
6. Orçamento é único por (oportunidade, versão): renegociar é criar uma nova versão. Uma venda por orçamento; `projetos.orcamento_id` é único.
7. **Datas coerentes:** evento depois da criação da entidade, horas e tokens dentro do período de trabalho da atividade, nada no futuro. `registros_horas.horas` vai de 0 a 24.
8. **Não grave** nas tabelas de configuração (`dashboards`, `cards`, `perfis`, `usuarios`, `perfil_cards`, `funil_etapas`).

Cada `INSERT` numa tabela monitorada vira um evento de tempo real e atualiza os cards afetados:

| Tabela | Evento |
|---|---|
| `registros_horas` | `horas_lancadas` |
| `atividade_eventos` | `status_atividade` |
| `atividades` | `atividade_criada` |
| `consumo_tokens` | `consumo_tokens` |
| `vendas` | `venda_fechada` |
| `oportunidade_eventos` | `etapa_oportunidade` |
| `reunioes` | `reuniao_registrada` |
| `orcamentos` | `orcamento_emitido` |
| `orcamento_eventos` | `orcamento_status` |

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

### Filtros, escopo e período

Os filtros vão na **query string** das rotas de dados dos cards (próximas etapas) e, no WebSocket, no objeto `filtros`. Eles são validados e **normalizados** antes de calcular: padrões aplicados, listas sem repetição e ordenadas, e parâmetros que não valem para o dashboard descartados. O filtro normalizado é a chave de cache e de assinatura do WebSocket.

| Parâmetro | Tipo | Valores | Vale em | Padrão |
|---|---|---|---|---|
| `escopo` | enum | `projeto`, `time`, `sprint` | projetos, ia | `projeto` |
| `id` | inteiro positivo | id do projeto, do time ou da sprint; vazio = todos | projetos, ia | vazio |
| `de`, `ate` | `AAAA-MM-DD` | período | todos | últimos 30 dias, contando hoje |
| `tipo_projeto` | lista | `web`, `mobile`, `integracao`, `api`, `outro` | todos (em vendas, só nos cards de orçamento/venda) | todos |
| `status_projeto` | lista | `em_andamento`, `concluido`, `pausado`, `cancelado` | projetos, ia | todos |
| `origem_lead` | lista | texto livre (valores de `oportunidades.origem`) | vendas | todos |
| `modelo` | lista | texto livre (valores de `consumo_tokens.modelo`) | ia | todos |

Listas aceitam valores separados por vírgula (`tipo_projeto=web,api`) e/ou o parâmetro repetido (`tipo_projeto=web&tipo_projeto=api`). Exemplo:

```
GET /dashboards/projetos/dados?escopo=time&id=2&de=2026-10-01&ate=2026-10-31&tipo_projeto=web,api
```

**Período padrão:** sem `de` nem `ate`, os 30 dias terminados hoje (no fuso `APP_TIMEZONE`). Só com `de`: de `de` até hoje. Só com `ate`: os 30 dias terminados em `ate`.

**Validação (400, com mensagem em português):** escopo ou enum inválido, `id` que não seja inteiro positivo, data fora do formato `AAAA-MM-DD` ou inexistente (ex.: `2026-02-30`), `de` maior que `ate`, intervalo maior que 5 anos, `escopo=sprint` sem `id`, sprint inexistente, mais de 50 valores numa lista ou valor com mais de 100 caracteres. Parâmetros que não valem para o dashboard são **ignorados**, mesmo se inválidos (ex.: `escopo=xyz` em vendas).

**Escopo** (dashboards projetos e ia):

| Conjunto | `escopo=projeto` | `escopo=time` | `escopo=sprint` |
|---|---|---|---|
| **P** projetos (sempre filtrados por `tipo_projeto` e `status_projeto`) | o projeto `id` (todos, se vazio) | projetos com ao menos uma atividade do time `id` (com qualquer atividade, se vazio) | projetos com ao menos uma atividade da sprint `id` |
| **A** atividades (de projetos de P, criadas até o fim do dia `ate`) | todas | do time `id` (todas, se vazio) | da sprint `id` |
| **H** horas | lançamentos das atividades de A | idem | idem |
| **T** tokens (filtrados por `modelo`) | todo o consumo dos projetos de P, inclusive sem atividade | consumo de desenvolvedores do time `id`, em projetos de P | consumo ligado às atividades de A |
| **Quota** | soma de `limite_tokens` dos projetos de P | idem | idem |

Em `escopo=time`, a quota de um projeto é compartilhada entre os times que trabalham nele. Por isso o percentual de quota por time é **aproximado**: compara o consumo de um time com a quota inteira dos projetos.

**Regra de período:** cada card declara um modo.
- **Acumulado:** considera tudo até o fim do dia `ate` e ignora `de` (progresso, horas contra estimativa, quota).
- **Período:** considera eventos entre o início do dia `de` e o fim do dia `ate` (custo no período, vendas fechadas).
- Em `escopo=sprint`, `de` e `ate` passam a ser o início e o fim da sprint `id`, qualquer que seja o valor enviado.
- **Período anterior** (para as variações): o intervalo de mesma duração que termina no dia anterior a `de`. Ex.: 11/11 a 20/11 → 01/11 a 10/11.

---

#### `GET /filtros/opcoes`
Opções para preencher os filtros do front. Em cada sprint, `atual` indica se hoje (no fuso do app) está entre `inicio` e `fim`, inclusive; o front usa isso para pré-selecionar a sprint atual. Listas sem dados vêm vazias (`[]`).

**Resposta de sucesso `200`** (resumida)
```json
{
  "sucesso": true,
  "dados": {
    "tipos_projeto": [ { "valor": "web", "rotulo": "Web" }, { "valor": "integracao", "rotulo": "Integração" } ],
    "status_projeto": [ { "valor": "em_andamento", "rotulo": "Em andamento" } ],
    "origens": ["indicacao", "site"],
    "modelos": ["modelo-a", "modelo-b"],
    "projetos": [ { "id": 1, "nome": "Portal", "tipo": "web", "status": "em_andamento" } ],
    "times": [ { "id": 1, "nome": "Time Beta" } ],
    "sprints": [
      { "id": 2, "time_id": 1, "time_nome": "Time Beta", "numero": 2,
        "inicio": "2026-11-08", "fim": "2026-11-14", "atual": true }
    ]
  }
}
```

Ordem: origens e modelos em ordem alfabética, projetos e times por nome, sprints por time e início.

**Erros**

| Status | Quando | Mensagem |
|---|---|---|
| `401` | Erros de token | — |
| `500` | Erro no banco | `Erro interno. Tente novamente mais tarde.` |

---

### Dados dos cards

Os cards são calculados na API, a partir dos filtros (seção [Filtros, escopo e período](#filtros-escopo-e-período)). O front só desenha o que recebe.

#### `GET /dashboards/:chave/dados`
Calcula **todos** os cards que o perfil vê no dashboard, para os filtros da query string, e devolve os resultados na ordem dos códigos.
- Os cards são calculados em paralelo, até **4 ao mesmo tempo**, com **10 s por card**.
- **Se um card falhar** (erro, timeout ou cálculo pendente), o endpoint **não falha**: aquele card volta com `erro` no lugar de `dados`, e o motivo real vai só para o log.
- `filtros_aplicados` mostra os filtros já normalizados: padrões aplicados, listas ordenadas e parâmetros que não valem para o dashboard como `null`.

```
GET /dashboards/vendas/dados?de=2026-10-01&ate=2026-10-31&origem_lead=site,indicacao
```

**Resposta de sucesso `200`** (resumida; valores ilustrativos)
```json
{
  "sucesso": true,
  "dados": {
    "dashboard": { "id": 2, "chave": "vendas", "titulo": "Vendas" },
    "filtros_aplicados": {
      "id": null, "de": "2026-10-01", "ate": "2026-10-31",
      "tipo_projeto": null, "status_projeto": null,
      "origem_lead": ["indicacao", "site"], "modelo": null
    },
    "cards": [
      {
        "codigo": "ven-01", "titulo": "Receita fechada", "tipo": "numero",
        "dados": { "valor": 125000.0, "unidade": "R$", "variacao": { "valor": 12.5, "unidade": "%" }, "detalhe": [ { "rotulo": "Vendas", "valor": 4, "unidade": "vendas" } ] },
        "alerta": null,
        "atualizado_em": "2026-11-14T12:00:00-03:00"
      },
      {
        "codigo": "ven-02", "titulo": "Taxa de conversão", "tipo": "numero",
        "alerta": null,
        "erro": "Não foi possível calcular este card."
      }
    ]
  }
}
```

**Erros**

| Status | Quando | Mensagem |
|---|---|---|
| `400` | Filtro inválido (ver [validação](#filtros-escopo-e-período)) | mensagem específica, ex.: `'de' (2026-11-10) não pode ser maior que 'ate' (2026-11-01).` |
| `401` | Erros de token | — |
| `403` | O perfil não vê nenhum card do dashboard | `Você não tem acesso a este recurso` |
| `404` | Dashboard inexistente ou inativo | `Dashboard não encontrado.` |
| `500` | Erro no banco ao buscar o dashboard, os cards ou a sprint | `Erro interno. Tente novamente mais tarde.` |

---

#### `GET /cards/:codigo/dados`
Calcula **um** card. Os filtros valem conforme o dashboard do card: por exemplo, `escopo` é ignorado em `ven-01`. Assim como no endpoint do dashboard, uma falha de cálculo devolve **200** com o card trazendo `erro`.

**Resposta de sucesso `200`**
```json
{
  "sucesso": true,
  "dados": {
    "filtros_aplicados": { "escopo": "projeto", "id": 3, "de": "2026-10-16", "ate": "2026-11-14", "tipo_projeto": null, "status_projeto": null, "origem_lead": null, "modelo": null },
    "card": {
      "codigo": "proj-01", "titulo": "Progresso", "tipo": "numero",
      "dados": { "valor": 72.5, "unidade": "%", "variacao": null, "detalhe": [ { "rotulo": "Concluídas", "valor": 12, "unidade": "atividades" } ] },
      "alerta": null,
      "atualizado_em": "2026-11-14T12:00:00-03:00"
    }
  }
}
```

**Erros**

| Status | Quando | Mensagem |
|---|---|---|
| `400` | Filtro inválido | mensagem específica |
| `401` | Erros de token | — |
| `403` | O card existe e está ativo, mas o perfil não pode vê-lo | `Você não tem acesso a este recurso` |
| `404` | Card inexistente, inativo ou de dashboard inativo | `Card não encontrado.` |
| `500` | Erro no banco | `Erro interno. Tente novamente mais tarde.` |

---

### Formato dos dados dos cards

Envelope de cada card (`CardResultado`):

| Campo | Descrição |
|---|---|
| `codigo`, `titulo`, `tipo` | Vêm da tabela `cards`. A `descricao` (texto do modal) vem em `GET /dashboards/:chave` |
| `dados` | Formato conforme o `tipo` (abaixo). Ausente quando há `erro` |
| `alerta` | Sempre `null` no MVP (limiares fora do escopo) |
| `atualizado_em` | Quando o card foi calculado, com o offset do fuso. Ausente quando há `erro` |
| `erro` | Só existe quando o card falhou: `Não foi possível calcular este card.` |

Formatos de `dados` por tipo (as chaves seguem o formato de dados do Recharts, usado pelo shadcn/ui):

| Tipo | Formato |
|---|---|
| `numero` | `{ "valor": 72.5, "unidade": "%", "variacao": { "valor": 3.2, "unidade": "%" } \| null, "detalhe": [ { "rotulo": "Entrada", "valor": 1200, "unidade": "tokens" } ] \| null }` |
| `linhas`, `area` | `{ "series": [ { "chave": "ideal", "nome": "Ideal" } ], "pontos": [ { "x": "2026-11-01", "ideal": 0, "real": null } ], "eixo_x": "...", "eixo_y": "...", "unidade": "%", "referencia": { "nome": "...", "valor": 1000 } \| null, "extra": { } \| null }` |
| `barras` | `{ "itens": [ { "rotulo": "...", "valor": 12.5, "extra": "38%" } ], "unidade": "h", "horizontal": true, "total": 12 \| null }` |
| `barras_agrupadas` | `{ "series": [ { "chave": "real", "nome": "Real", "pilha": "custo" } ], "itens": [ { "rotulo": "Análise", "real": 12, "estimado": 10 } ], "unidade": "h" }`. Séries com o mesmo `pilha` empilham; `pilha` ausente = sem empilhamento |
| `donut` | `{ "fatias": [ { "rotulo": "Abertas", "chave": "aberta", "valor": 4 } ], "total": 10 }` |
| `radial` | `{ "valor": 65.0, "meta": 100, "unidade": "%", "detalhe": [ { "rotulo": "Consumido", "valor": 650000, "unidade": "tokens" } ] }` |
| `lista` | `{ "colunas": ["Projeto", "Alerta", "Valor", "Quando"], "linhas": [ ["P1", "Quota", "85,3%", "2026-11-10"] ], "total": 3 }` |

Convenções: uma razão com denominador 0 devolve `null` (o front mostra "—"). Percentuais saem com 1 casa, R$ e horas com 2, dias com 1. Valor ausente numa série é `null`. **Dado vazio nunca é erro:** a estrutura vem com zeros ou listas vazias. As chaves extras de cada card (ex.: `n` em `proj-10`, `tokens` em `ia-07`) são documentadas junto com o cálculo do card.

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
- **Swagger antes do fim do plano:** o plano original deixava o Swagger perto do final. Ele entrou na `PULAPI-006` para o front já testar as rotas existentes, e o `docs/` é regenerado a cada etapa nova. Filtros e escopo passaram para a `PULAPI-007`.
- **Versão fixa do swag (`v1.16.4`):** a CLI e a biblioteca usam a mesma versão, para o `docs/docs.go` gerado ser compatível com o `go.mod`.
- **`/swagger` redireciona para `/swagger/index.html`:** o `gin-swagger` só responde em `/swagger/*any`. O redirecionamento evita um 404 em quem digita só `/swagger`.
- **Swagger UI com `PersistAuthorization`:** o token informado em **Authorize** continua salvo ao recarregar a página.
- **`SWAGGER_ENABLED` aceita `true/false`, `1/0` e `sim/nao`.** Um valor diferente impede a API de subir, em vez de ser ignorado em silêncio.
- **Período padrão = 30 dias contando hoje:** "últimos 30 dias até hoje" foi interpretado como `de = hoje − 29`, ou seja, 30 dias no total, incluindo hoje.
- **Parâmetros que não valem para o dashboard são ignorados antes da validação:** `escopo=xyz` em vendas não dá 400. Assim o front pode mandar o mesmo conjunto de filtros para qualquer dashboard.
- **Sprint inexistente em `escopo=sprint` dá 400** (`Sprint N não encontrada.`), porque sem a sprint não há período. Já um projeto ou time inexistente em `id` não é erro: devolve dados vazios, já que "dado vazio nunca é erro".
- **Limites nas listas:** no máximo 50 valores por filtro e 100 caracteres por valor, para evitar consultas gigantes com texto livre (`origem_lead`, `modelo`).
- **Rótulos de `status_projeto`:** a especificação não os define. Foram usados "Em andamento", "Concluído", "Pausado" e "Cancelado". Todos os rótulos de enums ficam em `models/enums.go`.
- **Conjuntos do escopo como CTEs (`repositories.CTEsEscopo`):** `p`, `a`, `h`, `t` e `quota` são montados uma vez, com placeholders `$n` numerados por um `ConsultaSQL` compartilhado com a consulta do card. O texto SQL só sai de trechos fixos escolhidos por `switch` (whitelist), e toda entrada vai como parâmetro (listas com `pq.Array` e cast para o enum). As CTEs não aplicam a regra de período: cada card filtra `h` e `t` pelo modo que usa.
- **`escopo=time` sem `id`:** P são os projetos com ao menos uma atividade (de qualquer time), e T é o consumo de qualquer desenvolvedor nesses projetos.
- **Datas como `models.Data`:** um dia do calendário (meia-noite no fuso do app) que vira `"AAAA-MM-DD"` no JSON. Colunas `DATE` lidas do banco são convertidas sem trocar de dia (`utils.DataDoBanco`).
- **Teste de semântica com transação e rollback:** o cenário dos conjuntos do escopo é inserido numa transação desfeita no final. Mesmo assim, só roda com `PULSO_TESTE_FIXTURE=1`, porque as sequências avançam.
- **Os 41 cards registrados desde já, com cálculo "pendente":** a API não sobe se algum card ativo ficar sem Calculador. Para os cards entrarem aos poucos sem quebrar essa regra, cada código tem um calculador pendente com os **gatilhos reais** (seção 12). Ele devolve "cálculo ainda não implementado", o que vira o `erro` padrão do card. Cada etapa de cards troca os pendentes do seu grupo pelo cálculo real (`Registro.Registrar`).
- **Calculador sobrescrito pelo serviço:** `codigo`, `titulo`, `tipo`, `alerta` (sempre `null`) e `atualizado_em` são preenchidos pelo `Servico` a partir da tabela `cards`. O calculador só monta `dados`.
- **Timeout garantido mesmo se o cálculo ignorar o contexto:** o cálculo roda numa goroutine e o serviço espera o resultado ou o prazo de 10 s, o que vier primeiro. Um pânico num calculador vira erro daquele card, sem derrubar a requisição.
- **`GET /cards/:codigo/dados` com falha de cálculo devolve 200** e o card com `erro`, igual ao endpoint do dashboard, para o front tratar falha de card do mesmo jeito nos dois casos.
- **Card de dashboard inativo dá 404**, mesmo se o card estiver ativo.
- **`alerta: null` também nos cards com erro:** o envelope sempre traz `alerta`; `dados` e `atualizado_em` só aparecem no sucesso, e `erro` só na falha.
- **`models.Data` lê e grava `"AAAA-MM-DD"` em JSON**, para os filtros aplicados poderem ser lidos de volta (ex.: no WebSocket).
- **Dados de exemplo antes dos cards:** o `dev_dados_exemplo.sql` entrou na `PULAPI-009`, antes dos grupos de cards, para cada card ser conferido contra dados reais desde a sua etapa.
- **Dados de exemplo com 5 projetos e 37 atividades** (a especificação pede no mínimo 4 e ~25): um projeto concluído e um pausado exercitam o filtro de status, e 6 sprints por time preenchem o histórico de 6 sprints do `proj-13`.
- **Dados de exemplo determinísticos:** valores fixos e variação calculada a partir dos ids, sem `random()`. Assim, duas cargas no mesmo dia produzem os mesmos números, o que facilita comparar os cards.
- **Script de exemplo com trava e transação única:** recusa rodar se já houver clientes, times, oportunidades ou projetos. Usa tabelas temporárias de apoio (`_opp`, `_atv`...), que somem no `COMMIT`; nas tabelas reais, só `INSERT`.
- **Preço dos modelos fictício** (R$ por milhão de tokens), só para os custos terem ordens de grandeza diferentes entre os modelos.
- **`.gitignore`:** além do pedido, ignora `.env.*` (exceto `.env.example`), arquivos de chave/certificado e o binário do Delve (`__debug_bin*`). Em `.vscode/`, só o `launch.json` é versionado.

---

## Histórico de etapas

- `PULAPI-001` — estrutura do projeto: módulo Go, pastas, `.gitignore`, `.env.example`, debugger do VS Code, `main.go` mínimo com shutdown gracioso e README inicial.
- `PULAPI-002` — conexão com o banco (pool singleton com `Ping`), validação das variáveis obrigatórias, padrão de resposta, `GET /health` (503 se o banco cair), `.gitattributes` com LF e testes de unidade.
- `PULAPI-003` — Docker: `Dockerfile` multi-stage (usuário sem privilégio, `HEALTHCHECK`), `.dockerignore`, `docker-compose.yml` com Postgres 16 (schema e seed na primeira subida, porta 5433) e API, `.env.example` atualizado e seção "Rodando com Docker".
- `PULAPI-004` — autenticação: `POST /auth/login` (bcrypt, mensagem genérica, tempo constante), `GET /auth/me`, geração e validação de JWT HS256, `AuthMiddleware` (injeta `usuario_id`, `perfil_id` e `perfil`), fuso da aplicação (`APP_TIMEZONE`) e testes.
- `PULAPI-005` — dashboards, cards e permissões: `GET /dashboards` (só dashboards com cards ativos e permitidos, com `total_cards`), `GET /dashboards/:chave` (cards visíveis, sem dados), regra de acesso 404/403 e testes.
- `PULAPI-006` — Swagger: `docs/` gerado pelo swag com as rotas existentes, Swagger UI em `/swagger/index.html` (com redirecionamento de `/swagger`), autenticação Bearer no **Authorize**, `SWAGGER_ENABLED` e instruções para regenerar.
- `PULAPI-007` — filtros e escopo: parse, validação e normalização dos filtros (padrões, enums, datas, limite de 5 anos, sprint), período anterior, conjuntos do escopo P/A/H/T/quota em SQL, `GET /filtros/opcoes`, testes de unidade e de integração.
- `PULAPI-008` — base dos cálculos: interface `Calculador`, registro dos 41 cards (com gatilhos e cálculo pendente), validação do registro na inicialização, serviço de cálculo (4 em paralelo, 10 s por card, isolamento de falhas), formatos de `dados` por tipo, `GET /dashboards/:chave/dados`, `GET /cards/:codigo/dados` e testes.
- `PULAPI-009` — dados de exemplo: `db/dev_dados_exemplo.sql` (~90 dias de funil, orçamentos, vendas, projetos, sprints, atividades, horas, tokens e metas, com trava e transação única) e guia do gerador Python no README.

-- =====================================================================
-- Pulso — Dashboard de BI da Fábrica de Software (Grupo 7)
-- Schema v2 — PostgreSQL 14+ (criado do zero)
--
-- CONTRATO ENTRE MÓDULOS
--   * Gerador (Python): apenas INSERT, nas tabelas de cadastro, CRM,
--     projetos, produção e IA. Nunca faz UPDATE nem DELETE. Mudança de
--     estado vira uma linha nova em uma tabela de eventos
--     (atividade_eventos, oportunidade_eventos, orcamento_eventos).
--   * API (Go): apenas SELECT. Detecta dado novo por polling, usando o
--     `id` de cada tabela como cursor (WHERE id > $ultimo_id).
--   * Tabelas de configuração e acesso (dashboards, cards, funil_etapas,
--     metas_receita, perfis, usuarios, perfil_cards): preenchidas por
--     seed/migração, nunca pelo gerador.
--
-- Estado atual de uma atividade, oportunidade ou orçamento = último
-- evento (maior ocorrido_em; desempate pelo maior id).
--
-- Para acrescentar um valor a um tipo enumerado depois:
--   ALTER TYPE tipo_projeto ADD VALUE 'novo_valor';
-- =====================================================================

-- ---------------------------------------------------------------------
-- 0. TIPOS ENUMERADOS
-- ---------------------------------------------------------------------

CREATE TYPE tipo_projeto     AS ENUM ('web', 'mobile', 'integracao', 'api', 'outro');
CREATE TYPE status_projeto   AS ENUM ('em_andamento', 'concluido', 'pausado', 'cancelado');
CREATE TYPE etapa_trabalho   AS ENUM ('analise', 'desenvolvimento', 'testes');
CREATE TYPE complexidade     AS ENUM ('pequena', 'media', 'grande');
CREATE TYPE tipo_atividade   AS ENUM ('analise', 'nova_funcionalidade', 'melhoria', 'correcao', 'outro');
CREATE TYPE status_atividade AS ENUM ('aberta', 'em_andamento', 'concluida');
CREATE TYPE etapa_funil      AS ENUM ('lead', 'qualificacao', 'reuniao', 'proposta_enviada',
                                      'negociacao', 'ganha', 'perdida');
CREATE TYPE status_orcamento AS ENUM ('pendente', 'aprovada', 'recusada', 'em_revisao');
CREATE TYPE tipo_card        AS ENUM ('numero', 'linhas', 'area', 'barras',
                                      'barras_agrupadas', 'donut', 'radial', 'lista');

-- ---------------------------------------------------------------------
-- 1. CONFIGURAÇÃO DO DASHBOARD (seed/migração, lida pela API)
-- ---------------------------------------------------------------------

-- Um registro por assunto principal: projetos, vendas e ia.
-- As abas Projeto, Time e Sprint são visões do mesmo dashboard.
CREATE TABLE dashboards (
    id         BIGSERIAL   PRIMARY KEY,
    chave      TEXT        NOT NULL UNIQUE CHECK (chave IN ('projetos', 'vendas', 'ia')),
    titulo     TEXT        NOT NULL,
    descricao  TEXT        NOT NULL,
    ativo      BOOLEAN     NOT NULL DEFAULT true,
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Um registro por card. `codigo` é a chave que liga o card à função de
-- cálculo em Go (ex.: 'proj-04'). `descricao` é o texto exibido no modal
-- de informação: o que o card mostra e como é calculado.
CREATE TABLE cards (
    id            BIGSERIAL   PRIMARY KEY,
    dashboard_id  BIGINT      NOT NULL REFERENCES dashboards(id),
    codigo        TEXT        NOT NULL UNIQUE,
    titulo        TEXT        NOT NULL,
    descricao     TEXT        NOT NULL,
    tipo          tipo_card   NOT NULL,
    ativo         BOOLEAN     NOT NULL DEFAULT true,
    criado_em     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cards_dashboard ON cards(dashboard_id);

-- ---------------------------------------------------------------------
-- 2. CADASTROS
-- ---------------------------------------------------------------------

CREATE TABLE clientes (
    id         BIGSERIAL   PRIMARY KEY,
    nome       TEXT        NOT NULL,
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE times (
    id         BIGSERIAL   PRIMARY KEY,
    nome       TEXT        NOT NULL UNIQUE,
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE desenvolvedores (
    id         BIGSERIAL   PRIMARY KEY,
    nome       TEXT        NOT NULL,
    time_id    BIGINT      NOT NULL REFERENCES times(id),
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_desenvolvedores_time ON desenvolvedores(time_id);

-- ---------------------------------------------------------------------
-- 3. CRM E ORÇAMENTAÇÃO (simula Grupos 2 e 3)
-- ---------------------------------------------------------------------

CREATE TABLE oportunidades (
    id          BIGSERIAL   PRIMARY KEY,
    cliente_id  BIGINT      NOT NULL REFERENCES clientes(id),
    origem      TEXT        NOT NULL,  -- canal de captação; alimenta o filtro origem_lead
    criado_em   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_oportunidades_cliente ON oportunidades(cliente_id);
CREATE INDEX idx_oportunidades_origem ON oportunidades(origem);

-- Funil como máquina de estados: cada mudança de etapa é uma linha nova.
CREATE TABLE oportunidade_eventos (
    id               BIGSERIAL    PRIMARY KEY,
    oportunidade_id  BIGINT       NOT NULL REFERENCES oportunidades(id),
    etapa            etapa_funil  NOT NULL,
    motivo_perda     TEXT,
    ocorrido_em      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    -- motivo_perda existe se, e somente se, a etapa for 'perdida'
    CONSTRAINT ck_motivo_perda CHECK ((etapa = 'perdida') = (motivo_perda IS NOT NULL))
);

CREATE INDEX idx_oport_eventos_oportunidade ON oportunidade_eventos(oportunidade_id, ocorrido_em DESC, id DESC);
CREATE INDEX idx_oport_eventos_ocorrido ON oportunidade_eventos(ocorrido_em);

CREATE TABLE reunioes (
    id               BIGSERIAL   PRIMARY KEY,
    oportunidade_id  BIGINT      NOT NULL REFERENCES oportunidades(id),
    realizada_em     TIMESTAMPTZ NOT NULL DEFAULT now(),
    criado_em        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_reunioes_oportunidade ON reunioes(oportunidade_id);

CREATE TABLE orcamentos (
    id               BIGSERIAL     PRIMARY KEY,
    oportunidade_id  BIGINT        NOT NULL REFERENCES oportunidades(id),
    tipo_projeto     tipo_projeto  NOT NULL,
    complexidade     complexidade  NOT NULL,
    horas_estimadas  NUMERIC(10,2) NOT NULL CHECK (horas_estimadas > 0),
    valor            NUMERIC(14,2) NOT NULL CHECK (valor > 0),
    margem_prevista  NUMERIC(5,2)  NOT NULL CHECK (margem_prevista >= 0 AND margem_prevista < 100),  -- em %
    versao           INTEGER       NOT NULL DEFAULT 1 CHECK (versao >= 1),
    emitido_em       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    UNIQUE (oportunidade_id, versao)
);

CREATE INDEX idx_orcamentos_emitido ON orcamentos(emitido_em);
CREATE INDEX idx_orcamentos_tipo ON orcamentos(tipo_projeto);

CREATE TABLE orcamento_eventos (
    id            BIGSERIAL         PRIMARY KEY,
    orcamento_id  BIGINT            NOT NULL REFERENCES orcamentos(id),
    status        status_orcamento  NOT NULL,
    ocorrido_em   TIMESTAMPTZ       NOT NULL DEFAULT now()
);

CREATE INDEX idx_orc_eventos_orcamento ON orcamento_eventos(orcamento_id, ocorrido_em DESC, id DESC);

-- ---------------------------------------------------------------------
-- 4. PROJETOS E VENDAS
-- ---------------------------------------------------------------------

CREATE TABLE projetos (
    id             BIGSERIAL      PRIMARY KEY,
    cliente_id     BIGINT         NOT NULL REFERENCES clientes(id),
    orcamento_id   BIGINT         UNIQUE REFERENCES orcamentos(id),  -- orçamento que originou o projeto
    nome           TEXT           NOT NULL,
    tipo           tipo_projeto   NOT NULL,                          -- alimenta o filtro tipo_projeto
    status         status_projeto NOT NULL DEFAULT 'em_andamento',
    horas_orcadas  NUMERIC(10,2)  NOT NULL CHECK (horas_orcadas > 0),
    limite_tokens  BIGINT         NOT NULL CHECK (limite_tokens > 0),  -- quota de IA do projeto
    criado_em      TIMESTAMPTZ    NOT NULL DEFAULT now()
);

CREATE INDEX idx_projetos_cliente ON projetos(cliente_id);
CREATE INDEX idx_projetos_tipo ON projetos(tipo);

CREATE TABLE vendas (
    id            BIGSERIAL     PRIMARY KEY,
    projeto_id    BIGINT        NOT NULL REFERENCES projetos(id),
    orcamento_id  BIGINT        NOT NULL UNIQUE REFERENCES orcamentos(id),  -- uma venda por orçamento
    valor         NUMERIC(14,2) NOT NULL CHECK (valor > 0),
    fechada_em    TIMESTAMPTZ   NOT NULL DEFAULT now(),
    criado_em     TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX idx_vendas_projeto ON vendas(projeto_id);
CREATE INDEX idx_vendas_fechada_em ON vendas(fechada_em);

-- ---------------------------------------------------------------------
-- 5. PRODUÇÃO (simula Grupo 5)
-- ---------------------------------------------------------------------

-- A sprint pertence a um time e reúne atividades de vários projetos.
CREATE TABLE sprints (
    id         BIGSERIAL   PRIMARY KEY,
    time_id    BIGINT      NOT NULL REFERENCES times(id),
    numero     INTEGER     NOT NULL CHECK (numero > 0),
    inicio     DATE        NOT NULL,
    fim        DATE        NOT NULL,
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (fim >= inicio),
    UNIQUE (time_id, numero),
    UNIQUE (id, time_id)   -- permite a chave estrangeira composta de atividades
);

CREATE INDEX idx_sprints_periodo ON sprints(inicio, fim);

-- O card (issue). sprint_id nulo = backlog. Mover uma atividade entre
-- sprints fica fora do MVP, para manter o contrato só de INSERT.
CREATE TABLE atividades (
    id                          BIGSERIAL      PRIMARY KEY,
    projeto_id                  BIGINT         NOT NULL REFERENCES projetos(id),
    time_id                     BIGINT         NOT NULL REFERENCES times(id),
    sprint_id                   BIGINT,
    titulo                      TEXT           NOT NULL,
    tipo                        tipo_atividade NOT NULL,
    complexidade                complexidade   NOT NULL,
    estimativa_analise          NUMERIC(8,2)   NOT NULL DEFAULT 0 CHECK (estimativa_analise >= 0),
    estimativa_desenvolvimento  NUMERIC(8,2)   NOT NULL DEFAULT 0 CHECK (estimativa_desenvolvimento >= 0),
    estimativa_testes           NUMERIC(8,2)   NOT NULL DEFAULT 0 CHECK (estimativa_testes >= 0),
    estimativa_horas            NUMERIC(9,2)   GENERATED ALWAYS AS
                                  (estimativa_analise + estimativa_desenvolvimento + estimativa_testes) STORED,
    criado_em                   TIMESTAMPTZ    NOT NULL DEFAULT now(),
    CONSTRAINT ck_atividade_estimativa
        CHECK (estimativa_analise + estimativa_desenvolvimento + estimativa_testes > 0),
    -- a sprint tem de ser do mesmo time da atividade (não verificada no backlog)
    FOREIGN KEY (sprint_id, time_id) REFERENCES sprints (id, time_id),
    UNIQUE (id, projeto_id)  -- permite chaves estrangeiras compostas de horas e tokens
);

CREATE INDEX idx_atividades_projeto ON atividades(projeto_id);
CREATE INDEX idx_atividades_time ON atividades(time_id);
CREATE INDEX idx_atividades_sprint ON atividades(sprint_id);

-- Mudanças de status: o status atual é o último evento da atividade.
CREATE TABLE atividade_eventos (
    id            BIGSERIAL        PRIMARY KEY,
    atividade_id  BIGINT           NOT NULL REFERENCES atividades(id),
    status        status_atividade NOT NULL,
    ocorrido_em   TIMESTAMPTZ      NOT NULL DEFAULT now()
);

CREATE INDEX idx_ativ_eventos_atividade ON atividade_eventos(atividade_id, ocorrido_em DESC, id DESC);
CREATE INDEX idx_ativ_eventos_ocorrido ON atividade_eventos(ocorrido_em);

-- Timesheet. projeto_id é redundante com a atividade, mas a chave composta
-- garante que os dois nunca discordem.
CREATE TABLE registros_horas (
    id               BIGSERIAL      PRIMARY KEY,
    projeto_id       BIGINT         NOT NULL REFERENCES projetos(id),
    atividade_id     BIGINT         NOT NULL,
    desenvolvedor_id BIGINT         NOT NULL REFERENCES desenvolvedores(id),
    etapa            etapa_trabalho NOT NULL,
    horas            NUMERIC(5,2)   NOT NULL CHECK (horas > 0 AND horas <= 24),
    custo_hora       NUMERIC(10,2)  NOT NULL CHECK (custo_hora > 0),  -- em R$
    trabalhado_em    DATE           NOT NULL DEFAULT CURRENT_DATE,
    criado_em        TIMESTAMPTZ    NOT NULL DEFAULT now(),
    FOREIGN KEY (atividade_id, projeto_id) REFERENCES atividades (id, projeto_id)
);

CREATE INDEX idx_horas_projeto ON registros_horas(projeto_id);
CREATE INDEX idx_horas_atividade ON registros_horas(atividade_id);
CREATE INDEX idx_horas_desenvolvedor ON registros_horas(desenvolvedor_id);
CREATE INDEX idx_horas_trabalhado_em ON registros_horas(trabalhado_em);

-- ---------------------------------------------------------------------
-- 6. IA (simula Grupo 4, do lado do desenvolvimento)
-- ---------------------------------------------------------------------

-- atividade_id é opcional: consumo sem atividade fica fora dos cards
-- por atividade, mas entra nos totais por projeto.
CREATE TABLE consumo_tokens (
    id                BIGSERIAL      PRIMARY KEY,
    projeto_id        BIGINT         NOT NULL REFERENCES projetos(id),
    atividade_id      BIGINT,
    desenvolvedor_id  BIGINT         NOT NULL REFERENCES desenvolvedores(id),
    etapa             etapa_trabalho NOT NULL,
    modelo            TEXT           NOT NULL,
    tokens_entrada    INTEGER        NOT NULL CHECK (tokens_entrada >= 0),
    tokens_saida      INTEGER        NOT NULL CHECK (tokens_saida >= 0),
    custo             NUMERIC(12,4)  NOT NULL CHECK (custo >= 0),  -- em R$
    ocorrido_em       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    criado_em         TIMESTAMPTZ    NOT NULL DEFAULT now(),
    -- quando há atividade, ela precisa ser do mesmo projeto (não verificada se for nula)
    FOREIGN KEY (atividade_id, projeto_id) REFERENCES atividades (id, projeto_id)
);

CREATE INDEX idx_tokens_projeto ON consumo_tokens(projeto_id);
CREATE INDEX idx_tokens_atividade ON consumo_tokens(atividade_id);
CREATE INDEX idx_tokens_desenvolvedor ON consumo_tokens(desenvolvedor_id);
CREATE INDEX idx_tokens_ocorrido_em ON consumo_tokens(ocorrido_em);
CREATE INDEX idx_tokens_modelo ON consumo_tokens(modelo);

-- ---------------------------------------------------------------------
-- 7. PARÂMETROS (seed/migração, lidos pela API)
-- ---------------------------------------------------------------------

-- Probabilidade fixa por etapa, usada no pipeline ponderado (ven-06).
CREATE TABLE funil_etapas (
    etapa          etapa_funil   PRIMARY KEY,
    ordem          SMALLINT      NOT NULL UNIQUE,
    probabilidade  NUMERIC(4,3)  NOT NULL CHECK (probabilidade >= 0 AND probabilidade <= 1)
);

-- Meta mensal de receita, usada na linha de referência de ven-10.
-- `mes` é sempre o primeiro dia do mês.
CREATE TABLE metas_receita (
    mes    DATE          PRIMARY KEY CHECK (EXTRACT(DAY FROM mes) = 1),
    valor  NUMERIC(14,2) NOT NULL CHECK (valor > 0)
);

-- ---------------------------------------------------------------------
-- 8. AUTENTICAÇÃO E ACESSO (seed/migração, lidas pela API)
-- ---------------------------------------------------------------------

-- Grupos de usuário (ex.: diretoria, lider_tecnologia).
CREATE TABLE perfis (
    id         BIGSERIAL   PRIMARY KEY,
    nome       TEXT        NOT NULL UNIQUE,
    descricao  TEXT        NOT NULL,
    criado_em  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Usuários do dashboard. O id é UUID (vai no claim do JWT). A senha é
-- guardada como hash bcrypt, nunca em texto. Os usuários entram por seed;
-- não há cadastro pela API.
CREATE TABLE usuarios (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    nome        TEXT        NOT NULL,
    email       TEXT        NOT NULL,
    senha_hash  TEXT        NOT NULL,
    perfil_id   BIGINT      NOT NULL REFERENCES perfis(id),
    ativo       BOOLEAN     NOT NULL DEFAULT true,
    criado_em   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- E-mail único sem diferenciar maiúsculas de minúsculas.
CREATE UNIQUE INDEX uq_usuarios_email ON usuarios (lower(email));
CREATE INDEX idx_usuarios_perfil ON usuarios(perfil_id);

-- Quais cards cada perfil enxerga. Um dashboard aparece para o usuário
-- se ele tiver acesso a pelo menos um card dele.
CREATE TABLE perfil_cards (
    perfil_id  BIGINT NOT NULL REFERENCES perfis(id),
    card_id    BIGINT NOT NULL REFERENCES cards(id),
    PRIMARY KEY (perfil_id, card_id)
);

CREATE INDEX idx_perfil_cards_card ON perfil_cards(card_id);

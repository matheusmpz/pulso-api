-- =====================================================================
-- Pulso — DADOS DE EXEMPLO (SOMENTE PARA DESENVOLVIMENTO)
--
-- NÃO RODE EM PRODUÇÃO. Este script cria dados fictícios e coerentes
-- para exercitar os 41 cards enquanto o gerador Python não existe, e
-- serve de REFERÊNCIA para quem for escrever o gerador: ele mostra a
-- ordem dos INSERTs, as regras de integridade e o "formato" esperado
-- dos dados.
--
-- Pré-requisitos: banco criado com db/schema_v2.sql e db/seed.sql e
-- SEM dados de negócio. O script se recusa a rodar se já houver dados.
-- Para recomeçar do zero no Docker: docker compose down -v
--
-- Como rodar (o banco é o do .env; troque usuário e banco se preciso):
--   Docker (Git Bash):
--     docker compose exec -T db psql -v ON_ERROR_STOP=1 -U postgres -d pulso_db < db/dev_dados_exemplo.sql
--   Docker (PowerShell):
--     Get-Content db/dev_dados_exemplo.sql | docker compose exec -T db psql -v ON_ERROR_STOP=1 -U postgres -d pulso_db
--   DBeaver: abra o arquivo num editor SQL da conexão e use
--     "Executar script" (Alt+X).
--
-- Tudo roda numa única transação: ou entra tudo, ou nada.
--
-- ---------------------------------------------------------------------
-- O QUE É CRIADO (datas relativas a HOJE, no fuso America/Sao_Paulo)
-- ---------------------------------------------------------------------
--   3 clientes, 2 times com 3 desenvolvedores cada
--   20 oportunidades em 4 origens: 5 ganhas, 7 perdidas (com motivo), 8
--     em aberto em várias etapas; eventos do funil, reuniões, orçamentos
--     (alguns com versão 2) e eventos de status dos orçamentos
--   5 projetos (web, web, mobile, api, integracao), um por oportunidade
--     ganha, cada um com sua venda; status em_andamento, concluido e
--     pausado; limites de tokens variados
--   6 sprints de 14 dias por time: 4 passadas, a atual e uma futura
--   37 atividades com tipos, complexidades e estimativas variadas,
--     eventos aberta → em_andamento → concluida, algumas em andamento,
--     abertas, sem nenhum evento (backlog) e algumas estourando a
--     estimativa
--   horas lançadas nas três etapas; consumo de tokens em 3 modelos
--     (ligado às atividades e também sem atividade), com picos
--   metas de receita para os 3 últimos meses
--
-- ---------------------------------------------------------------------
-- REGRAS QUE O GERADOR PYTHON PRECISA SEGUIR
-- ---------------------------------------------------------------------
--   1. Só INSERT. Nunca UPDATE nem DELETE. Mudança de estado é uma linha
--      nova em atividade_eventos, oportunidade_eventos ou
--      orcamento_eventos. Estado atual = último evento (maior
--      ocorrido_em; empate, maior id). Sem evento = estado inicial
--      (atividade 'aberta', oportunidade 'lead', orçamento 'pendente').
--   2. Autocommit ou transações CURTAS: a API detecta dado novo pelo
--      MAX(id) de cada tabela; uma transação longa pode tornar visível
--      um id menor depois de um maior já visto, e o evento se perde.
--   3. Ordem dos INSERTs (por causa das chaves estrangeiras):
--        clientes → times → desenvolvedores
--        oportunidades → oportunidade_eventos / reunioes
--        orcamentos → orcamento_eventos
--        projetos (orcamento_id do orçamento ganho) → vendas
--        sprints → atividades → atividade_eventos
--        registros_horas / consumo_tokens
--   4. Chaves compostas:
--        atividades.(sprint_id, time_id) → a sprint é do MESMO time da
--          atividade (sprint_id nulo = backlog);
--        registros_horas.(atividade_id, projeto_id) e
--        consumo_tokens.(atividade_id, projeto_id) → a atividade é do
--          MESMO projeto (consumo_tokens.atividade_id pode ser nulo).
--   5. oportunidade_eventos.motivo_perda existe se, e somente se,
--      etapa = 'perdida'. 'ganha' e 'perdida' são finais.
--   6. orcamentos: único por (oportunidade_id, versao); nova negociação =
--      nova versão. vendas: uma por orçamento; projetos.orcamento_id é
--      único e aponta para o orçamento que virou o projeto.
--   7. Datas coerentes: evento depois da criação da entidade; horas e
--      tokens dentro do período de trabalho da atividade; nada no futuro
--      (exceto sprints futuras). registros_horas.horas entre 0 e 24.
--   8. Não grave nas tabelas de configuração (dashboards, cards,
--      perfis, usuarios, perfil_cards, funil_etapas). metas_receita é
--      parâmetro: aqui entra só como exemplo.
-- =====================================================================

BEGIN;

-- ---------------------------------------------------------------------
-- 0. TRAVA: recusa rodar se já houver dados de negócio
-- ---------------------------------------------------------------------
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM clientes) OR EXISTS (SELECT 1 FROM projetos)
       OR EXISTS (SELECT 1 FROM oportunidades) OR EXISTS (SELECT 1 FROM times) THEN
        RAISE EXCEPTION 'O banco já tem dados de negócio. Este script só roda num banco recém-criado (schema + seed). No Docker: docker compose down -v e suba de novo.';
    END IF;
END $$;

-- "Hoje" no fuso da aplicação. As tabelas temporárias (_ref, _opp...)
-- são só apoio do script e somem no fim da transação.
CREATE TEMP TABLE _ref ON COMMIT DROP AS
SELECT (now() AT TIME ZONE 'America/Sao_Paulo')::date AS d;

-- Padrão usado em todo o script para "n dias atrás, às 10h (hora local)":
--   ((SELECT d FROM _ref) - n + TIME '10:00') AT TIME ZONE 'America/Sao_Paulo'

-- ---------------------------------------------------------------------
-- 1. CADASTROS: clientes, times e desenvolvedores
-- ---------------------------------------------------------------------
INSERT INTO clientes (nome, criado_em)
SELECT nome, ((SELECT d FROM _ref) - 120 + TIME '09:00') AT TIME ZONE 'America/Sao_Paulo'
FROM (VALUES ('Mercado Bom Preço'), ('Clínica Vida'), ('Logística Rota Sul')) v(nome);

INSERT INTO times (nome) VALUES ('Time Alfa'), ('Time Beta');

-- custo_hora não é coluna do desenvolvedor: vai em cada registro de
-- horas. Aqui fica numa tabela temporária só para o script.
CREATE TEMP TABLE _dev ON COMMIT DROP AS
SELECT v.nome, v.time_nome, v.ordem, v.custo_hora
FROM (VALUES
    ('Ana Souza',    'Time Alfa', 0, 120.00),
    ('Bruno Lima',   'Time Alfa', 1,  95.00),
    ('Carla Mendes', 'Time Alfa', 2, 140.00),
    ('Diego Rocha',  'Time Beta', 0, 110.00),
    ('Elisa Prado',  'Time Beta', 1,  85.00),
    ('Fábio Nunes',  'Time Beta', 2, 130.00)
) v(nome, time_nome, ordem, custo_hora);

INSERT INTO desenvolvedores (nome, time_id)
SELECT d.nome, t.id FROM _dev d JOIN times t ON t.nome = d.time_nome ORDER BY d.time_nome, d.ordem;

-- ---------------------------------------------------------------------
-- 2. CRM: oportunidades, eventos do funil e reuniões
-- ---------------------------------------------------------------------
-- Cada oportunidade percorre o funil a partir de 'lead' (dia "criado"),
-- avançando uma etapa a cada "passo" dias até "alcance" (a etapa mais
-- longe que chegou). Se houver desfecho ('ganha' ou 'perdida'), ele
-- ocorre um passo depois do alcance. As 5 ganhas viram projetos.
CREATE TEMP TABLE _opp ON COMMIT DROP AS
SELECT * FROM (VALUES
 -- cod   cliente               origem       criado alcance             desfecho   motivo                        passo tipo          complex.   horas valor     margem v2
 ('O01', 'Clínica Vida',       'indicacao',  108, 'negociacao',       'ganha',   NULL,                          4, 'web',        'pequena', 100,  30000.00, 45, false),
 ('O02', 'Mercado Bom Preço',  'site',       105, 'negociacao',       'ganha',   NULL,                          5, 'web',        'grande',  320,  96000.00, 35, true),
 ('O03', 'Clínica Vida',       'indicacao',   95, 'negociacao',       'ganha',   NULL,                          6, 'mobile',     'grande',  400, 140000.00, 30, false),
 ('O04', 'Logística Rota Sul', 'linkedin',    65, 'negociacao',       'ganha',   NULL,                          5, 'api',        'media',   240,  72000.00, 40, true),
 ('O05', 'Mercado Bom Preço',  'evento',      32, 'negociacao',       'ganha',   NULL,                          4, 'integracao', 'media',   160,  48000.00, 25, false),
 ('O06', 'Logística Rota Sul', 'site',        85, 'proposta_enviada', 'perdida', 'Preço acima do orçamento',    5, 'web',        'media',   200,  64000.00, 38, false),
 ('O07', 'Clínica Vida',       'linkedin',    70, 'reuniao',          'perdida', 'Escolheu um concorrente',     4, NULL,         NULL,      NULL, NULL,     NULL, false),
 ('O08', 'Mercado Bom Preço',  'evento',      55, 'qualificacao',     'perdida', 'Sem orçamento no momento',    6, NULL,         NULL,      NULL, NULL,     NULL, false),
 ('O09', 'Logística Rota Sul', 'site',        45, 'negociacao',       'perdida', 'Prazo de entrega',            5, 'integracao', 'grande',  300,  99000.00, 32, true),
 ('O10', 'Clínica Vida',       'indicacao',   30, 'proposta_enviada', 'perdida', 'Preço acima do orçamento',    4, 'mobile',     'media',   220,  70000.00, 36, false),
 ('O11', 'Mercado Bom Preço',  'site',        20, 'lead',             'perdida', 'Sem resposta do cliente',     5, NULL,         NULL,      NULL, NULL,     NULL, false),
 ('O12', 'Logística Rota Sul', 'linkedin',    25, 'reuniao',          'perdida', 'Escolheu um concorrente',     3, NULL,         NULL,      NULL, NULL,     NULL, false),
 ('O13', 'Mercado Bom Preço',  'site',        40, 'negociacao',       NULL,      NULL,                          6, 'web',        'media',   180,  58000.00, 34, true),
 ('O14', 'Clínica Vida',       'evento',      28, 'proposta_enviada', NULL,      NULL,                          5, 'outro',      'pequena',  60,  18000.00, 42, false),
 ('O15', 'Logística Rota Sul', 'indicacao',   18, 'reuniao',          NULL,      NULL,                          4, NULL,         NULL,      NULL, NULL,     NULL, false),
 ('O16', 'Mercado Bom Preço',  'linkedin',    12, 'qualificacao',     NULL,      NULL,                          5, NULL,         NULL,      NULL, NULL,     NULL, false),
 ('O17', 'Clínica Vida',       'site',         6, 'lead',             NULL,      NULL,                          3, NULL,         NULL,      NULL, NULL,     NULL, false),
 ('O18', 'Logística Rota Sul', 'site',         3, 'lead',             NULL,      NULL,                          3, NULL,         NULL,      NULL, NULL,     NULL, false),
 ('O19', 'Mercado Bom Preço',  'indicacao',   22, 'negociacao',       NULL,      NULL,                          4, 'api',        'grande',  350, 115000.00, 30, false),
 ('O20', 'Clínica Vida',       'linkedin',    35, 'proposta_enviada', NULL,      NULL,                          6, 'mobile',     'pequena',  90,  27000.00, 40, false)
) v(cod, cliente, origem, criado, alcance, desfecho, motivo, passo, tipo, complexidade, horas, valor, margem, tem_v2);

ALTER TABLE _opp ADD COLUMN id BIGINT;
ALTER TABLE _opp ADD COLUMN alcance_ordem INT;
UPDATE _opp o SET alcance_ordem = f.ordem FROM funil_etapas f WHERE f.etapa = o.alcance::etapa_funil;
-- (UPDATE só na tabela temporária do script; nas tabelas reais, só INSERT)

-- Insere as oportunidades (na ordem em que surgiram)
INSERT INTO oportunidades (cliente_id, origem, criado_em)
SELECT c.id, o.origem, ((SELECT d FROM _ref) - o.criado + TIME '09:00') AT TIME ZONE 'America/Sao_Paulo'
FROM _opp o JOIN clientes c ON c.nome = o.cliente
ORDER BY o.criado DESC;

-- Guarda o id de cada uma (cliente + criado_em é único nestes dados)
UPDATE _opp o SET id = op.id
FROM oportunidades op JOIN clientes c ON c.id = op.cliente_id
WHERE c.nome = o.cliente
  AND op.criado_em = ((SELECT d FROM _ref) - o.criado + TIME '09:00') AT TIME ZONE 'America/Sao_Paulo';

-- Eventos do funil: uma linha por etapa percorrida (lead até alcance),
-- mais o desfecho. O dia da etapa de ordem k é criado − (k − 1) × passo.
INSERT INTO oportunidade_eventos (oportunidade_id, etapa, motivo_perda, ocorrido_em)
SELECT o.id, f.etapa, NULL,
       ((SELECT d FROM _ref) - o.criado + (f.ordem - 1) * o.passo + TIME '10:00') AT TIME ZONE 'America/Sao_Paulo'
FROM _opp o
JOIN funil_etapas f ON f.ordem <= o.alcance_ordem AND f.etapa NOT IN ('ganha', 'perdida')
UNION ALL
SELECT o.id, o.desfecho::etapa_funil, o.motivo,
       ((SELECT d FROM _ref) - o.criado + o.alcance_ordem * o.passo + TIME '16:00') AT TIME ZONE 'America/Sao_Paulo'
FROM _opp o
WHERE o.desfecho IS NOT NULL
ORDER BY 4, 1;

-- Reuniões: oportunidades que chegaram à etapa 'reuniao' (ordem 3) têm
-- de 1 a 3 reuniões, a partir do dia em que entraram nessa etapa.
INSERT INTO reunioes (oportunidade_id, realizada_em, criado_em)
SELECT o.id, ts.t, ts.t
FROM _opp o
CROSS JOIN LATERAL generate_series(1, 1 + (substr(o.cod, 2)::int % 3)) AS g(i)
CROSS JOIN LATERAL (
    SELECT ((SELECT d FROM _ref) - o.criado + 2 * o.passo + (g.i - 1) + TIME '14:00') AT TIME ZONE 'America/Sao_Paulo' AS t
) ts
WHERE o.alcance_ordem >= 3
ORDER BY ts.t;

-- ---------------------------------------------------------------------
-- 3. ORÇAMENTOS e eventos de status
-- ---------------------------------------------------------------------
-- Versão 1: emitida no dia da 'proposta_enviada' (ordem 4).
-- Versão 2 (tem_v2): emitida no dia da 'negociacao' (ordem 5), 8% mais
-- barata. A versão final (maior) é a que vira projeto quando ganha.
CREATE TEMP TABLE _orc ON COMMIT DROP AS
SELECT o.cod, o.id AS oportunidade_id, v.versao,
       o.tipo, o.complexidade, o.horas,
       CASE WHEN v.versao = 2 THEN round(o.valor * 0.92, 2) ELSE o.valor END AS valor,
       CASE WHEN v.versao = 2 THEN o.margem - 4 ELSE o.margem END AS margem,
       o.criado - (CASE WHEN v.versao = 2 THEN 4 ELSE 3 END) * o.passo AS dia_emissao,
       (v.versao = CASE WHEN o.tem_v2 THEN 2 ELSE 1 END) AS final
FROM _opp o
CROSS JOIN (VALUES (1), (2)) v(versao)
WHERE o.alcance_ordem >= 4 AND (v.versao = 1 OR o.tem_v2);

ALTER TABLE _orc ADD COLUMN id BIGINT;

WITH novos AS (
    INSERT INTO orcamentos (oportunidade_id, tipo_projeto, complexidade, horas_estimadas, valor, margem_prevista, versao, emitido_em)
    SELECT r.oportunidade_id, r.tipo::tipo_projeto, r.complexidade::complexidade, r.horas, r.valor, r.margem, r.versao,
           ((SELECT d FROM _ref) - r.dia_emissao + TIME '11:00') AT TIME ZONE 'America/Sao_Paulo'
    FROM _orc r
    ORDER BY r.cod, r.versao
    RETURNING id, oportunidade_id, versao
)
UPDATE _orc r SET id = n.id FROM novos n
WHERE n.oportunidade_id = r.oportunidade_id AND n.versao = r.versao;

-- Status: toda versão nasce 'pendente' (exceto O20, sem nenhum evento,
-- para exercitar o padrão "sem evento = pendente"). A versão 1 vai para
-- 'em_revisao' quando sai a 2. A versão final vira 'aprovada' (ganha) ou
-- 'recusada' (perdida). O19 (aberta, em negociação) fica 'em_revisao'.
INSERT INTO orcamento_eventos (orcamento_id, status, ocorrido_em)
SELECT id, status::status_orcamento, ((SELECT d FROM _ref) - dia + TIME '12:00') AT TIME ZONE 'America/Sao_Paulo'
FROM (
    SELECT r.id, 'pendente' AS status, r.dia_emissao AS dia FROM _orc r WHERE r.cod <> 'O20'
    UNION ALL
    SELECT r.id, 'em_revisao', o.criado - 4 * o.passo FROM _orc r JOIN _opp o ON o.cod = r.cod
    WHERE (r.versao = 1 AND o.tem_v2) OR (r.cod = 'O19')
    UNION ALL
    SELECT r.id, CASE o.desfecho WHEN 'ganha' THEN 'aprovada' ELSE 'recusada' END, o.criado - o.alcance_ordem * o.passo
    FROM _orc r JOIN _opp o ON o.cod = r.cod
    WHERE r.final AND o.desfecho IS NOT NULL
) s
ORDER BY dia DESC, id;

-- ---------------------------------------------------------------------
-- 4. PROJETOS e VENDAS
-- ---------------------------------------------------------------------
-- Um projeto por oportunidade ganha, ligado ao orçamento final
-- (projetos.orcamento_id) e criado no dia seguinte ao fechamento.
CREATE TEMP TABLE _proj ON COMMIT DROP AS
SELECT * FROM (VALUES
 -- cod   nome                       tipo          status          limite_tokens
 ('O01', 'Site Institucional',      'web',        'concluido',      400000),
 ('O02', 'Portal do Cliente',       'web',        'em_andamento',  3800000),
 ('O03', 'App de Agendamento',      'mobile',     'em_andamento',  3000000),
 ('O04', 'API de Rastreamento',     'api',        'em_andamento',  1800000),
 ('O05', 'Integração ERP',          'integracao', 'pausado',       1500000)
) v(cod, nome, tipo, status, limite_tokens);

INSERT INTO projetos (cliente_id, orcamento_id, nome, tipo, status, horas_orcadas, limite_tokens, criado_em)
SELECT c.id, r.id, p.nome, p.tipo::tipo_projeto, p.status::status_projeto, r.horas, p.limite_tokens,
       ((SELECT d FROM _ref) - (o.criado - o.alcance_ordem * o.passo) + 1 + TIME '09:00') AT TIME ZONE 'America/Sao_Paulo'
FROM _proj p
JOIN _opp o ON o.cod = p.cod
JOIN clientes c ON c.nome = o.cliente
JOIN _orc r ON r.cod = p.cod AND r.final
ORDER BY o.criado DESC;

-- Uma venda por projeto, no valor do orçamento final, fechada no dia
-- do evento 'ganha'.
INSERT INTO vendas (projeto_id, orcamento_id, valor, fechada_em, criado_em)
SELECT pr.id, r.id, r.valor, t.fechada, t.fechada
FROM _proj p
JOIN _opp o ON o.cod = p.cod
JOIN _orc r ON r.cod = p.cod AND r.final
JOIN projetos pr ON pr.orcamento_id = r.id
CROSS JOIN LATERAL (
    SELECT ((SELECT d FROM _ref) - (o.criado - o.alcance_ordem * o.passo) + TIME '16:30') AT TIME ZONE 'America/Sao_Paulo' AS fechada
) t
ORDER BY t.fechada;

-- ---------------------------------------------------------------------
-- 5. SPRINTS: 6 por time, de 14 dias
-- ---------------------------------------------------------------------
-- Sprint 5 = atual (de hoje − 6 até hoje + 7); 1 a 4 = passadas; 6 = futura.
INSERT INTO sprints (time_id, numero, inicio, fim, criado_em)
SELECT t.id, n,
       (SELECT d FROM _ref) - 62 + 14 * (n - 1),
       (SELECT d FROM _ref) - 62 + 14 * (n - 1) + 13,
       -- planejada 2 dias antes do início; a futura, há 4 dias (nada no
       -- futuro e antes das atividades dela, criadas há 3 e 2 dias)
       (least((SELECT d FROM _ref) - 64 + 14 * (n - 1), (SELECT d FROM _ref) - 4) + TIME '08:00') AT TIME ZONE 'America/Sao_Paulo'
FROM times t CROSS JOIN generate_series(1, 6) n
ORDER BY n, t.nome;

-- ---------------------------------------------------------------------
-- 6. ATIVIDADES
-- ---------------------------------------------------------------------
-- Colunas de apoio (só do script):
--   criado / inicio / fim: dias atrás (inicio nulo = nunca começou;
--     fim nulo = não concluída)
--   fator: horas reais ÷ estimativa (acima de 1 = estouro). Nas não
--     concluídas, é o quanto já foi gasto e a etapa de testes não tem horas.
--   intensidade: tokens de IA por hora trabalhada (0 = sem IA)
--   eventos: false = nenhum evento (fica 'aberta' por padrão)
CREATE TEMP TABLE _atv ON COMMIT DROP AS
SELECT * FROM (VALUES
 -- projeto                  time        sprint titulo                                    tipo                   complex.   e_ana e_dev e_tes criado inicio fim  fator intens. eventos
 -- ---- Time Alfa ----
 ('Portal do Cliente',      'Time Alfa', 1, 'Levantamento de requisitos do portal',  'analise',             'media',    12,  0,  2,  64,  62,   56,  1.00,  3000, true),
 ('Site Institucional',     'Time Alfa', 1, 'Layout das páginas do site',            'nova_funcionalidade', 'pequena',   2, 10,  3,  63,  61,   55,  0.90,  2000, true),
 ('Site Institucional',     'Time Alfa', 1, 'Formulário de contato',                 'nova_funcionalidade', 'pequena',   1,  6,  2,  63,  58,   52,  1.10,  5000, true),
 ('Portal do Cliente',      'Time Alfa', 1, 'Login e cadastro de clientes',          'nova_funcionalidade', 'grande',    6, 28,  8,  63,  57,   47,  1.25,  8000, true),
 ('Site Institucional',     'Time Alfa', 2, 'Publicação e SEO do site',              'melhoria',            'media',     2,  8,  4,  49,  47,   41,  0.80,  1500, true),
 ('Portal do Cliente',      'Time Alfa', 2, 'Catálogo de produtos',                  'nova_funcionalidade', 'grande',    6, 32, 10,  49,  46,   36,  1.40, 12000, true),
 ('Portal do Cliente',      'Time Alfa', 2, 'Correção no envio de e-mails',          'correcao',            'pequena',   1,  4,  2,  45,  44,   42,  1.60,  6000, true),
 ('Portal do Cliente',      'Time Alfa', 2, 'Carrinho de compras',                   'nova_funcionalidade', 'grande',    4, 24,  8,  48,  43,   25,  1.10,  9000, true),
 ('Portal do Cliente',      'Time Alfa', 3, 'Pagamento com Pix',                     'nova_funcionalidade', 'media',     3, 14,  5,  35,  33,   24,  0.95,  7000, true),
 ('API de Rastreamento',    'Time Alfa', 3, 'Modelagem dos eventos de rastreamento', 'analise',             'media',    10,  2,  2,  35,  33,   28,  1.00,  4000, true),
 ('Portal do Cliente',      'Time Alfa', 3, 'Ajuste de performance na listagem',     'melhoria',            'pequena',   1,  6,  2,  30,  29,   26,  0.70,  3000, true),
 ('Portal do Cliente',      'Time Alfa', 4, 'Área do cliente: pedidos',              'nova_funcionalidade', 'media',     3, 16,  5,  21,  19,   10,  1.15,  6000, true),
 ('Portal do Cliente',      'Time Alfa', 4, 'Correção de cálculo de frete',          'correcao',            'media',     2,  8,  4,  18,  17,   13,  1.50, 10000, true),
 ('API de Rastreamento',    'Time Alfa', 4, 'Webhook de status de entrega',          'nova_funcionalidade', 'media',     2, 12,  4,  21,  18,    9,  1.05,  9000, true),
 ('Portal do Cliente',      'Time Alfa', 4, 'Relatório de vendas',                   'nova_funcionalidade', 'grande',    4, 20,  6,  21,  16, NULL,  0.60,  5000, true),
 ('Integração ERP',         'Time Alfa', 5, 'Mapeamento das tabelas do ERP',         'analise',             'media',    12,  0,  3,   8,   6,    2,  1.00,  3000, true),
 ('Integração ERP',         'Time Alfa', 5, 'Sincronização de estoque',              'nova_funcionalidade', 'grande',    4, 30,  8,   7,   4, NULL,  0.40,  8000, true),
 ('Portal do Cliente',      'Time Alfa', 5, 'Cupom de desconto',                     'nova_funcionalidade', 'pequena',   1,  6,  2,   7,   5,    1,  1.20,  4000, true),
 ('Portal do Cliente',      'Time Alfa', 5, 'Correção no filtro de busca',           'correcao',            'pequena',   1,  3,  2,   4,   3, NULL,  0.50,  2000, true),
 ('Integração ERP',         'Time Alfa', 5, 'Integração de pedidos com o ERP',       'nova_funcionalidade', 'media',     2, 14,  4,   7, NULL, NULL, 0.00,     0, true),
 ('Integração ERP',         'Time Alfa', 6, 'Conciliação financeira',                'nova_funcionalidade', 'grande',    4, 24,  8,   3, NULL, NULL, 0.00,     0, false),
 ('Portal do Cliente',      'Time Alfa', NULL, 'Programa de fidelidade',             'outro',               'grande',    8, 30,  8,  20, NULL, NULL, 0.00,     0, false),
 -- ---- Time Beta ----
 ('App de Agendamento',     'Time Beta', 1, 'Protótipo das telas do app',            'analise',             'media',    14,  4,  2,  64,  62,   54,  1.10,  6000, true),
 ('App de Agendamento',     'Time Beta', 1, 'Cadastro de pacientes',                 'nova_funcionalidade', 'media',     2, 14,  4,  63,  58,   50,  1.20,  9000, true),
 ('App de Agendamento',     'Time Beta', 2, 'Agenda de consultas',                   'nova_funcionalidade', 'grande',    6, 30, 10,  49,  47,   37,  1.30, 14000, true),
 ('App de Agendamento',     'Time Beta', 2, 'Notificações push',                     'nova_funcionalidade', 'media',     2, 12,  4,  48,  44,   38,  0.90, 11000, true),
 ('App de Agendamento',     'Time Beta', 3, 'Correção de fuso nos horários',         'correcao',            'pequena',   1,  5,  2,  33,  32,   30,  1.80, 15000, true),
 ('API de Rastreamento',    'Time Beta', 3, 'Autenticação da API',                   'nova_funcionalidade', 'media',     2, 12,  4,  35,  33,   25,  1.00, 12000, true),
 ('App de Agendamento',     'Time Beta', 3, 'Tela de histórico',                     'melhoria',            'media',     2, 10,  4,  34,  31,   23,  0.85,  7000, true),
 ('API de Rastreamento',    'Time Beta', 4, 'Endpoint de posição dos veículos',      'nova_funcionalidade', 'grande',    4, 28,  8,  21,  19,    8,  1.20, 20000, true),
 ('App de Agendamento',     'Time Beta', 4, 'Pagamento de consultas',                'nova_funcionalidade', 'grande',    4, 24,  8,  21,  18, NULL,  0.70, 12000, true),
 ('API de Rastreamento',    'Time Beta', 4, 'Documentação da API',                   'outro',               'pequena',   1,  4,  3,  15,  12,    9,  1.00,  8000, true),
 ('API de Rastreamento',    'Time Beta', 5, 'Limite de requisições (rate limit)',    'melhoria',            'media',     2, 10,  4,   7,   5,    1,  1.10, 16000, true),
 ('API de Rastreamento',    'Time Beta', 5, 'Correção de duplicidade de eventos',    'correcao',            'media',     2,  8,  4,   6,   4, NULL,  0.60, 18000, true),
 ('App de Agendamento',     'Time Beta', 5, 'Avaliação do atendimento',              'nova_funcionalidade', 'pequena',   1,  6,  2,   7, NULL, NULL, 0.00,     0, true),
 ('App de Agendamento',     'Time Beta', 6, 'Modo offline',                          'nova_funcionalidade', 'grande',    6, 30,  8,   2, NULL, NULL, 0.00,     0, false),
 ('API de Rastreamento',    'Time Beta', NULL, 'Painel de monitoramento',            'outro',               'media',     4, 10,  4,  10, NULL, NULL, 0.00,     0, false)
) v(projeto, time_nome, sprint_num, titulo, tipo, complexidade, e_ana, e_dev, e_tes, criado, inicio, fim, fator, intensidade, eventos);

ALTER TABLE _atv ADD COLUMN id BIGINT;
ALTER TABLE _atv ADD COLUMN projeto_id BIGINT;
ALTER TABLE _atv ADD COLUMN time_id BIGINT;
ALTER TABLE _atv ADD COLUMN rn INT;
UPDATE _atv a SET projeto_id = p.id FROM projetos p WHERE p.nome = a.projeto;
UPDATE _atv a SET time_id = t.id FROM times t WHERE t.nome = a.time_nome;
UPDATE _atv a SET rn = x.rn FROM (SELECT titulo, row_number() OVER (ORDER BY criado DESC, titulo) AS rn FROM _atv) x WHERE x.titulo = a.titulo;

-- A sprint da atividade é do MESMO time (chave composta sprint_id, time_id)
INSERT INTO atividades (projeto_id, time_id, sprint_id, titulo, tipo, complexidade,
                        estimativa_analise, estimativa_desenvolvimento, estimativa_testes, criado_em)
SELECT a.projeto_id, a.time_id, s.id, a.titulo, a.tipo::tipo_atividade, a.complexidade::complexidade,
       a.e_ana, a.e_dev, a.e_tes,
       ((SELECT d FROM _ref) - a.criado + TIME '09:00') AT TIME ZONE 'America/Sao_Paulo'
FROM _atv a
LEFT JOIN sprints s ON s.time_id = a.time_id AND s.numero = a.sprint_num
ORDER BY a.rn;

UPDATE _atv a SET id = at.id FROM atividades at WHERE at.titulo = a.titulo AND at.projeto_id = a.projeto_id;

-- Eventos de status: 'aberta' na criação, 'em_andamento' no início e
-- 'concluida' no fim. Atividades com eventos = false ficam sem nenhum.
INSERT INTO atividade_eventos (atividade_id, status, ocorrido_em)
SELECT id, status::status_atividade, t
FROM (
    SELECT a.id, 'aberta' AS status, ((SELECT d FROM _ref) - a.criado + TIME '09:05') AT TIME ZONE 'America/Sao_Paulo' AS t
    FROM _atv a WHERE a.eventos
    UNION ALL
    SELECT a.id, 'em_andamento', ((SELECT d FROM _ref) - a.inicio + TIME '09:30') AT TIME ZONE 'America/Sao_Paulo'
    FROM _atv a WHERE a.inicio IS NOT NULL
    UNION ALL
    SELECT a.id, 'concluida', ((SELECT d FROM _ref) - a.fim + TIME '17:00') AT TIME ZONE 'America/Sao_Paulo'
    FROM _atv a WHERE a.fim IS NOT NULL
) e
ORDER BY t, id;

-- ---------------------------------------------------------------------
-- 7. HORAS LANÇADAS (registros_horas)
-- ---------------------------------------------------------------------
-- Para cada atividade iniciada e cada etapa com estimativa:
--   horas reais da etapa = estimativa da etapa × fator
--     (sem testes nas atividades ainda não concluídas)
--   janela de trabalho, do início até o fim (ou até ontem, se em
--   andamento): análise no primeiro quarto, desenvolvimento no meio e
--   testes no último quarto
--   as horas são divididas em lançamentos de até 6 h, espalhados pela
--   janela, alternando os desenvolvedores do time da atividade.
-- O projeto_id do lançamento é o da atividade (chave composta).
INSERT INTO registros_horas (projeto_id, atividade_id, desenvolvedor_id, etapa, horas, custo_hora, trabalhado_em, criado_em)
SELECT x.projeto_id, x.id, dv.id, x.etapa::etapa_trabalho, x.horas, dd.custo_hora, x.dia,
       (x.dia + TIME '18:00') AT TIME ZONE 'America/Sao_Paulo'
FROM (
    SELECT e.*,
           round(e.reais / e.n, 2) AS horas,
           e.j_ini + ((g.i - 1) * (e.j_fim - e.j_ini) / e.n) AS dia,
           (e.rn + e.ordem + g.i) % 3 AS dev_ordem
    FROM (
        SELECT a.id, a.projeto_id, a.time_nome, a.rn, et.etapa, et.ordem,
               round(et.est * a.fator * CASE WHEN a.fim IS NULL AND et.etapa = 'testes' THEN 0 ELSE 1 END, 2) AS reais,
               greatest(1, ceil(et.est * a.fator / 6.0))::int AS n,
               (SELECT d FROM _ref) - a.inicio + floor((a.inicio - coalesce(a.fim, 1)) * et.f_ini)::int AS j_ini,
               (SELECT d FROM _ref) - a.inicio + floor((a.inicio - coalesce(a.fim, 1)) * et.f_fim)::int AS j_fim
        FROM _atv a
        CROSS JOIN LATERAL (VALUES
            ('analise',         0, a.e_ana, 0.00, 0.25),
            ('desenvolvimento', 1, a.e_dev, 0.25, 0.75),
            ('testes',          2, a.e_tes, 0.75, 1.00)
        ) et(etapa, ordem, est, f_ini, f_fim)
        WHERE a.inicio IS NOT NULL AND et.est > 0
    ) e
    CROSS JOIN LATERAL generate_series(1, e.n) g(i)
    WHERE e.reais > 0
) x
JOIN _dev dd ON dd.time_nome = x.time_nome AND dd.ordem = x.dev_ordem
JOIN desenvolvedores dv ON dv.nome = dd.nome
ORDER BY x.dia, x.id, x.etapa;

-- ---------------------------------------------------------------------
-- 8. CONSUMO DE TOKENS (consumo_tokens)
-- ---------------------------------------------------------------------
-- Preço fictício por modelo, em R$ por milhão de tokens.
CREATE TEMP TABLE _modelo ON COMMIT DROP AS
SELECT * FROM (VALUES
    ('claude-sonnet', 18.00, 90.00),
    ('gemini-flash',   0.50,  2.00),
    ('gpt-4o-mini',    0.90,  3.60)
) v(modelo, preco_entrada, preco_saida);

-- 8.1 Consumo ligado à atividade: um por lançamento de horas, no mesmo
-- dia, mesma etapa e mesmo desenvolvedor. Tokens = horas × intensidade
-- da atividade (com variação). 75% entrada, 25% saída. Modelo conforme
-- a etapa: análise → claude-sonnet; desenvolvimento → alterna
-- claude-sonnet e gemini-flash; testes → gpt-4o-mini.
INSERT INTO consumo_tokens (projeto_id, atividade_id, desenvolvedor_id, etapa, modelo,
                            tokens_entrada, tokens_saida, custo, ocorrido_em, criado_em)
SELECT c.projeto_id, c.atividade_id, c.desenvolvedor_id, c.etapa, c.modelo,
       c.entrada, c.saida,
       round((c.entrada * m.preco_entrada + c.saida * m.preco_saida) / 1000000.0, 4),
       c.t, c.t
FROM (
    SELECT rh.projeto_id, rh.atividade_id, rh.desenvolvedor_id, rh.etapa,
           CASE
               WHEN rh.etapa = 'analise' THEN 'claude-sonnet'
               WHEN rh.etapa = 'testes'  THEN 'gpt-4o-mini'
               WHEN rh.id % 2 = 0        THEN 'claude-sonnet'
               ELSE 'gemini-flash'
           END AS modelo,
           round(rh.horas * a.intensidade * (0.7 + (rh.id % 7) / 10.0) * 0.75)::int AS entrada,
           round(rh.horas * a.intensidade * (0.7 + (rh.id % 7) / 10.0) * 0.25)::int AS saida,
           (rh.trabalhado_em + TIME '11:00' + (rh.id % 6) * INTERVAL '1 hour') AT TIME ZONE 'America/Sao_Paulo' AS t
    FROM registros_horas rh
    JOIN _atv a ON a.id = rh.atividade_id
    WHERE a.intensidade > 0
) c
JOIN _modelo m ON m.modelo = c.modelo
ORDER BY c.t;

-- 8.2 Consumo sem atividade (uso geral da IA no projeto: dúvidas,
-- documentação), a cada 2 dias, do início do trabalho no projeto até
-- ontem (ou até a última atividade, no projeto concluído).
INSERT INTO consumo_tokens (projeto_id, atividade_id, desenvolvedor_id, etapa, modelo,
                            tokens_entrada, tokens_saida, custo, ocorrido_em, criado_em)
SELECT c.projeto_id, NULL, c.dev_id, 'analise', 'gpt-4o-mini', c.entrada, c.saida,
       round((c.entrada * 0.90 + c.saida * 3.60) / 1000000.0, 4), c.t, c.t
FROM (
    SELECT p.projeto_id, dv.id AS dev_id,
           (4500 + (dia.n % 5) * 1100) AS entrada,
           (1500 + (dia.n % 5) * 400)  AS saida,
           ((SELECT d FROM _ref) - dia.n + TIME '15:30') AT TIME ZONE 'America/Sao_Paulo' AS t
    FROM (
        SELECT a.projeto_id, min(a.time_nome) AS time_nome,
               max(a.inicio) AS primeiro, min(coalesce(a.fim, 1)) AS ultimo
        FROM _atv a WHERE a.inicio IS NOT NULL
        GROUP BY a.projeto_id
    ) p
    CROSS JOIN LATERAL generate_series(p.ultimo, p.primeiro, 2) AS dia(n)
    JOIN _dev dd ON dd.time_nome = p.time_nome AND dd.ordem = 0
    JOIN desenvolvedores dv ON dv.nome = dd.nome
) c
ORDER BY c.t;

-- 8.3 Picos (para o card de alertas ia-12): dias com consumo bem acima
-- da média dos 7 dias anteriores. Sem atividade, modelo caro.
INSERT INTO consumo_tokens (projeto_id, atividade_id, desenvolvedor_id, etapa, modelo,
                            tokens_entrada, tokens_saida, custo, ocorrido_em, criado_em)
SELECT p.id, NULL, dv.id, 'desenvolvimento', 'claude-sonnet', v.entrada, v.saida,
       round((v.entrada * 18.00 + v.saida * 90.00) / 1000000.0, 4), t.t, t.t
FROM (VALUES
    ('API de Rastreamento', 'Diego Rocha', 3,  330000, 110000),
    ('App de Agendamento',  'Elisa Prado', 15, 240000,  80000)
) v(projeto, dev, dias_atras, entrada, saida)
JOIN projetos p ON p.nome = v.projeto
JOIN desenvolvedores dv ON dv.nome = v.dev
CROSS JOIN LATERAL (
    SELECT ((SELECT d FROM _ref) - v.dias_atras + TIME '16:45') AT TIME ZONE 'America/Sao_Paulo' AS t
) t;

-- ---------------------------------------------------------------------
-- 9. METAS DE RECEITA (parâmetro): mês atual e os 2 anteriores
-- ---------------------------------------------------------------------
INSERT INTO metas_receita (mes, valor)
SELECT (date_trunc('month', (SELECT d FROM _ref)) - make_interval(months => v.m))::date, v.valor
FROM (VALUES (2, 80000.00), (1, 90000.00), (0, 100000.00)) v(m, valor);

COMMIT;

-- Conferência rápida (opcional): quantidades por tabela
-- SELECT 'clientes', count(*) FROM clientes UNION ALL
-- SELECT 'oportunidades', count(*) FROM oportunidades UNION ALL
-- SELECT 'projetos', count(*) FROM projetos UNION ALL
-- SELECT 'atividades', count(*) FROM atividades UNION ALL
-- SELECT 'registros_horas', count(*) FROM registros_horas UNION ALL
-- SELECT 'consumo_tokens', count(*) FROM consumo_tokens;

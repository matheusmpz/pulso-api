-- =====================================================================
-- Pulso — seed inicial
-- Rodar DEPOIS do schema_v2.sql, em um banco vazio.
--
-- Conteúdo: dashboards, cards (com a descrição do modal de informação),
-- probabilidades do funil, perfis, permissões por perfil e dois
-- usuários de desenvolvimento.
--
-- Os usuários abaixo são só para desenvolvimento: troque as senhas
-- (e gere novos hashes bcrypt) antes de qualquer uso fora da sua máquina.
-- =====================================================================

BEGIN;

-- ---------------------------------------------------------------------
-- 1. DASHBOARDS
-- ---------------------------------------------------------------------

INSERT INTO dashboards (chave, titulo, descricao) VALUES
    ('projetos', 'Projetos',
     'Andamento dos projetos em desenvolvimento: progresso, horas por etapa, sprints e eficiência, por projeto, time ou sprint.'),
    ('vendas', 'Vendas',
     'Funil comercial, receita, propostas e comparação entre o que foi orçado e o que foi realizado.'),
    ('ia', 'Custo de IA',
     'Consumo de tokens e custo de IA no desenvolvimento, por projeto, time, sprint, etapa, modelo e atividade.');

-- ---------------------------------------------------------------------
-- 2. CARDS
-- O codigo liga o card à função de cálculo na API. A descricao é o texto
-- exibido no modal de informação: o que o card mostra e como é calculado.
-- ---------------------------------------------------------------------

INSERT INTO cards (dashboard_id, codigo, titulo, descricao, tipo)
SELECT d.id, v.codigo, v.titulo, v.descricao, v.tipo::tipo_card
FROM (VALUES
    -- ---- Projetos ----
    ('projetos', 'proj-01', 'Progresso', 'Mostra quanto do trabalho planejado já foi concluído. Cálculo: soma da estimativa das atividades concluídas dividida pela soma da estimativa de todas as atividades, em percentual. Atividades maiores pesam mais do que as menores.', 'numero'),
    ('projetos', 'proj-02', 'Horas', 'Mostra quanto das horas estimadas já foi consumido. Cálculo: horas lançadas dividido pela soma das horas estimadas das atividades. Acima de 100% significa que o trabalho passou do previsto.', 'numero'),
    ('projetos', 'proj-03', 'Custo de IA', 'Mostra o consumo de IA frente à quota. Cálculo: tokens consumidos (entrada mais saída) dividido pela soma dos limites de tokens dos projetos do filtro. O valor em reais do período aparece em destaque.', 'numero'),
    ('projetos', 'proj-04', 'Evolução do projeto (burn-up)', 'Compara o ritmo ideal com o real. Cada ponto é um dia: no eixo horizontal ficam as horas gastas acumuladas e, no vertical, o percentual de evolução acumulado. A linha ideal é a diagonal (100% de evolução ao gastar 100% das horas estimadas). A linha real abaixo da ideal indica que está sendo gasto mais tempo do que o previsto para entregar.', 'linhas'),
    ('projetos', 'proj-05', 'Issues por status', 'Mostra quantas atividades estão abertas, em andamento e concluídas. Cálculo: contagem das atividades pelo status atual, que é o do último evento de cada atividade.', 'donut'),
    ('projetos', 'proj-06', 'Burndown da sprint', 'Mostra o que ainda falta entregar na sprint, dia a dia. Cálculo: a cada dia, soma da estimativa das atividades da sprint que ainda não foram concluídas, comparada com uma reta ideal que vai do total inicial até zero no último dia da sprint.', 'area'),
    ('projetos', 'proj-07', 'Horas em análise', 'Compara as horas gastas com as estimadas na etapa de análise. Cálculo: horas lançadas na etapa dividido pela soma das estimativas de análise das atividades. O percentual consumido aparece no rótulo.', 'barras_agrupadas'),
    ('projetos', 'proj-08', 'Horas em desenvolvimento', 'Compara as horas gastas com as estimadas na etapa de desenvolvimento. Cálculo: horas lançadas na etapa dividido pela soma das estimativas de desenvolvimento das atividades. O percentual consumido aparece no rótulo.', 'barras_agrupadas'),
    ('projetos', 'proj-09', 'Horas em testes', 'Compara as horas gastas com as estimadas na etapa de testes. Cálculo: horas lançadas na etapa dividido pela soma das estimativas de testes das atividades. O percentual consumido aparece no rótulo.', 'barras_agrupadas'),
    ('projetos', 'proj-10', 'Tempo médio por complexidade', 'Mostra quanto tempo as atividades concluídas levam, por complexidade (pequena, média e grande). Cálculo: média das horas lançadas em cada grupo, ao lado da média das horas estimadas.', 'barras_agrupadas'),
    ('projetos', 'proj-11', 'Tempo médio por tipo de atividade', 'Mostra quanto tempo as atividades concluídas levam, por tipo (análise, nova funcionalidade, melhoria, correção e outros). Cálculo: média das horas lançadas em cada tipo, ao lado da média das horas estimadas.', 'barras_agrupadas'),
    ('projetos', 'proj-12', 'Atividades estourando a estimativa', 'Lista as atividades que mais passaram da estimativa. Cálculo: diferença entre as horas lançadas e as horas estimadas, considerando só atividades acima da estimativa, em ordem decrescente (as 5 maiores). O título informa o total de atividades estouradas.', 'barras'),
    ('projetos', 'proj-13', 'Histórico de sprints', 'Compara, nas últimas 6 sprints, o que foi planejado com o que foi concluído. Cálculo: soma da estimativa das atividades alocadas à sprint contra a soma da estimativa das atividades concluídas.', 'barras_agrupadas'),
    ('projetos', 'proj-14', 'Distribuição de horas', 'Mostra onde as horas foram gastas. Cálculo: soma das horas lançadas, agrupada por desenvolvedor na visão de projeto e por projeto nas visões de time e de sprint.', 'barras'),
    ('projetos', 'proj-15', 'Custo de IA no tempo', 'Mostra o consumo de IA acumulado ao longo do tempo frente ao limite. Cálculo: tokens acumulados por dia, com uma linha de referência igual à soma dos limites de tokens dos projetos do filtro.', 'area'),

    -- ---- Vendas ----
    ('vendas', 'ven-01', 'Receita fechada', 'Total vendido no período. Cálculo: soma do valor das vendas fechadas entre as datas do filtro, com a variação em relação ao período anterior de mesma duração.', 'numero'),
    ('vendas', 'ven-02', 'Taxa de conversão', 'Percentual de oportunidades que viraram venda. Cálculo: ganhas dividido por (ganhas mais perdidas) no período. Oportunidades ainda em aberto não entram na conta.', 'numero'),
    ('vendas', 'ven-03', 'Ticket médio', 'Valor médio de cada venda. Cálculo: receita fechada dividida pelo número de vendas no período.', 'numero'),
    ('vendas', 'ven-04', 'Ciclo médio de venda', 'Quantos dias, em média, uma venda leva do primeiro contato ao fechamento. Cálculo: média de dias entre a entrada do lead e a etapa ganha, nas oportunidades ganhas no período.', 'numero'),
    ('vendas', 'ven-05', 'Funil de vendas', 'Mostra em quais etapas as oportunidades avançam ou se perdem. Cálculo: número de oportunidades que chegaram a cada etapa no período. O rótulo traz o percentual de conversão em relação à etapa anterior.', 'barras'),
    ('vendas', 'ven-06', 'Pipeline em aberto', 'Mostra quanto dinheiro está em negociação. Cálculo: por etapa, a soma do valor das propostas em andamento (bruto) e esse valor multiplicado pela probabilidade fixa da etapa (ponderado).', 'barras_agrupadas'),
    ('vendas', 'ven-07', 'Motivos de perda', 'Mostra por que as oportunidades são perdidas. Cálculo: contagem das oportunidades perdidas por motivo de perda.', 'donut'),
    ('vendas', 'ven-08', 'Origem dos leads', 'Mostra de onde vêm os leads e quais canais convertem melhor. Cálculo: número de leads por origem. O rótulo de cada barra traz a taxa de conversão da origem.', 'barras'),
    ('vendas', 'ven-09', 'Reuniões até fechar', 'Mostra quantas reuniões são necessárias. Cálculo: média de reuniões por oportunidade, separada entre as ganhas e as perdidas.', 'barras'),
    ('vendas', 'ven-10', 'Receita ao longo do tempo', 'Acompanha a receita mês a mês contra a meta. Cálculo: soma do valor das vendas por mês de fechamento, com uma linha de referência igual à meta mensal de receita.', 'area'),
    ('vendas', 'ven-11', 'Propostas por status', 'Mostra a situação das propostas comerciais. Cálculo: contagem dos orçamentos pelo status atual (pendente, aprovada, recusada ou em revisão), que é o do último evento de cada orçamento.', 'donut'),
    ('vendas', 'ven-12', 'Margem prevista por complexidade', 'Mostra a margem que o motor de orçamentação está prevendo. Cálculo: média da margem prevista dos orçamentos por complexidade. O valor cobrado por hora (valor dividido pelas horas estimadas) aparece ao passar o mouse.', 'barras'),
    ('vendas', 'ven-13', 'Orçado x real (horas)', 'Mostra se o orçamento está acertando as horas. Cálculo: por tipo de projeto, a soma das horas estimadas nos orçamentos ganhos comparada com a soma das horas lançadas nos projetos vinculados a eles.', 'barras_agrupadas'),
    ('vendas', 'ven-14', 'Margem prevista x real', 'Mostra se a margem prevista se confirmou e quanto a IA consumiu dela. Cálculo: por tipo de projeto, uma barra com a margem prevista (valor vezes margem prevista) e uma pilha com a margem real e o custo de IA. A margem real é a receita menos o custo das horas menos o custo de IA, e a altura total da pilha é a margem real sem o custo de IA.', 'barras_agrupadas'),

    -- ---- IA ----
    ('ia', 'ia-01', 'Custo de IA', 'Quanto foi gasto com IA no período. Cálculo: soma do custo dos consumos de tokens no filtro, com a variação em relação ao período anterior.', 'numero'),
    ('ia', 'ia-02', 'Tokens consumidos', 'Volume de tokens usados. Cálculo: soma dos tokens de entrada e de saída. O detalhe mostra a divisão entre os dois.', 'numero'),
    ('ia', 'ia-03', 'Quota consumida', 'Quanto da quota de IA já foi usado. Cálculo: tokens consumidos dividido pela soma dos limites de tokens dos projetos do filtro.', 'radial'),
    ('ia', 'ia-04', 'Custo por atividade concluída', 'Quanto a IA custa, em média, por atividade entregue. Cálculo: custo de IA das atividades concluídas no período dividido pelo número de atividades concluídas.', 'numero'),
    ('ia', 'ia-05', 'Consumo diário e projeção', 'Mostra o ritmo de consumo e quando a quota deve acabar. Cálculo: tokens por dia, com média móvel de 7 dias. A projeção divide os tokens restantes da quota pela média diária dos últimos 7 dias e indica a data estimada de esgotamento.', 'area'),
    ('ia', 'ia-06', 'Custo por projeto', 'Mostra quais projetos mais consomem IA. Cálculo: os 10 projetos com maior custo de IA. Quando o filtro é um único projeto, o gráfico agrupa por sprint.', 'barras'),
    ('ia', 'ia-07', 'Custo por modelo', 'Mostra a participação de cada modelo de IA no custo. Cálculo: soma do custo por modelo. Os tokens aparecem ao passar o mouse.', 'donut'),
    ('ia', 'ia-08', 'Custo por etapa', 'Mostra em que fase do desenvolvimento a IA é mais usada. Cálculo: soma do custo de IA por etapa (análise, desenvolvimento e testes).', 'barras'),
    ('ia', 'ia-09', 'Custo por tipo e complexidade', 'Mostra quanto a IA custa por atividade, conforme o tipo e a complexidade. Cálculo: custo médio de IA por atividade, agrupado por tipo de atividade, com uma série para cada complexidade.', 'barras_agrupadas'),
    ('ia', 'ia-10', 'Intensidade de IA x desempenho', 'Mostra se usar mais IA melhora a entrega. Cálculo: as atividades concluídas são divididas em três faixas de intensidade (baixa, média e alta), medida em tokens por hora estimada, dentro de cada complexidade. Para cada faixa, o gráfico mostra a média do desvio (horas reais dividido pelas horas estimadas): abaixo de 100% significa entrega mais rápida do que o previsto. Faixas com menos de 5 atividades são marcadas como amostra pequena.', 'barras_agrupadas'),
    ('ia', 'ia-11', 'Peso da IA no custo', 'Mostra quanto da despesa de um projeto vem da IA. Cálculo: por projeto, custo das horas e custo de IA empilhados. O rótulo traz o percentual de IA, ou seja, o custo de IA dividido pela soma do custo das horas com o custo de IA.', 'barras_agrupadas'),
    ('ia', 'ia-12', 'Alertas de consumo', 'Lista o que merece atenção no consumo de IA. Cálculo: projetos com 80% ou mais da quota usada e dias com consumo acima de 3 vezes a média dos últimos 7 dias, ordenados por gravidade.', 'lista')
) AS v(dashboard, codigo, titulo, descricao, tipo)
JOIN dashboards d ON d.chave = v.dashboard;

-- ---------------------------------------------------------------------
-- 3. PARÂMETROS DO FUNIL (probabilidade fixa por etapa, usada em ven-06)
-- ---------------------------------------------------------------------

INSERT INTO funil_etapas (etapa, ordem, probabilidade) VALUES
    ('lead',             1, 0.05),
    ('qualificacao',     2, 0.10),
    ('reuniao',          3, 0.20),
    ('proposta_enviada', 4, 0.30),
    ('negociacao',       5, 0.60),
    ('ganha',            6, 1.00),
    ('perdida',          7, 0.00);

-- ---------------------------------------------------------------------
-- 4. PERFIS E PERMISSÕES POR CARD
-- ---------------------------------------------------------------------

INSERT INTO perfis (nome, descricao) VALUES
    ('diretoria',        'Acesso a todos os dashboards e cards, incluindo vendas.'),
    ('lider_tecnologia', 'Acesso aos cards de andamento dos projetos.');

-- Diretoria: todos os cards.
INSERT INTO perfil_cards (perfil_id, card_id)
SELECT p.id, c.id
FROM perfis p
CROSS JOIN cards c
WHERE p.nome = 'diretoria';

-- Líder de tecnologia: cards dos dashboards listados em IN (...).
-- Para liberar também o dashboard de IA, troque por IN ('projetos', 'ia').
INSERT INTO perfil_cards (perfil_id, card_id)
SELECT p.id, c.id
FROM perfis p
JOIN dashboards d ON d.chave IN ('projetos')
JOIN cards c ON c.dashboard_id = d.id
WHERE p.nome = 'lider_tecnologia';

-- ---------------------------------------------------------------------
-- 5. USUÁRIOS DE DESENVOLVIMENTO (senhas em hash bcrypt)
-- ---------------------------------------------------------------------

INSERT INTO usuarios (nome, email, senha_hash, perfil_id)
SELECT 'Diretoria (dev)', 'diretoria@pulso.local',
       '$2a$10$xb3iIjPWHOzgEyYUZAvNlu6d2bgJIJGJN5h8oyiqTLD3SD3fkgodC', p.id
FROM perfis p WHERE p.nome = 'diretoria';

INSERT INTO usuarios (nome, email, senha_hash, perfil_id)
SELECT 'Líder de Tecnologia (dev)', 'lider@pulso.local',
       '$2a$10$ZmVOlb5nJx56JWMOTI8Z3uUkJ/78ro9q3GykTq3f2HU/DPONr.Bua', p.id
FROM perfis p WHERE p.nome = 'lider_tecnologia';

COMMIT;

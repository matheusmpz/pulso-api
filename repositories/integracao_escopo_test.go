//go:build integration

// Confere a semântica dos conjuntos do escopo (seção 7.2) com um cenário
// pequeno inserido DENTRO DE UMA TRANSAÇÃO que sempre termina em ROLLBACK:
// nada fica gravado. Mesmo assim, as sequências (ids) avançam, então este
// teste só roda com PULSO_TESTE_FIXTURE=1, de preferência num banco
// descartável:
//
//	PULSO_TESTE_FIXTURE=1 go test -tags integration -run Semantica ./repositories/
package repositories

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"pulso-api/models"
	"pulso-api/utils"
)

// cenarioEscopo guarda os ids gerados pelo cenário.
type cenarioEscopo struct {
	timeAlfa, timeBeta int64
	p1, p2, p3         int64 // p1 web/em_andamento, p2 api/pausado, p3 mobile sem atividades
	sprintAlfa         int64
}

// montarCenario insere, na transação:
//
//	times Alfa e Beta; devs dA (Alfa) e dB (Beta)
//	P1 web em_andamento (quota 1000), P2 api pausado (500), P3 mobile (200)
//	sprint SA (Alfa), SB (Beta)
//	A1 P1/Alfa/SA, A2 P1/Beta/SB, A3 P2/Alfa/SA, A4 P2/Alfa criada no futuro
//	horas: uma linha em A1, A2 e A3
//	tokens: t1 P1/A1/dA m1, t2 P1/sem atividade/dB m2, t3 P2/A3/dA m1, t4 P3/sem atividade/dA m1
func montarCenario(t *testing.T, tx *sql.Tx) cenarioEscopo {
	t.Helper()
	ctx := context.Background()
	var c cenarioEscopo
	id := func(query string, args ...interface{}) int64 {
		var v int64
		if err := tx.QueryRowContext(ctx, query+" RETURNING id", args...).Scan(&v); err != nil {
			t.Fatalf("cenário: %v\n%s", err, query)
		}
		return v
	}

	cliente := id(`INSERT INTO clientes (nome) VALUES ('Cliente teste')`)
	c.timeAlfa = id(`INSERT INTO times (nome) VALUES ('Teste Alfa')`)
	c.timeBeta = id(`INSERT INTO times (nome) VALUES ('Teste Beta')`)
	devA := id(`INSERT INTO desenvolvedores (nome, time_id) VALUES ('dA', $1)`, c.timeAlfa)
	devB := id(`INSERT INTO desenvolvedores (nome, time_id) VALUES ('dB', $1)`, c.timeBeta)

	projeto := `INSERT INTO projetos (cliente_id, nome, tipo, status, horas_orcadas, limite_tokens) VALUES ($1, $2, $3, $4, 100, $5)`
	c.p1 = id(projeto, cliente, "P1", "web", "em_andamento", 1000)
	c.p2 = id(projeto, cliente, "P2", "api", "pausado", 500)
	c.p3 = id(projeto, cliente, "P3", "mobile", "em_andamento", 200)

	sprint := `INSERT INTO sprints (time_id, numero, inicio, fim) VALUES ($1, 1, CURRENT_DATE - 7, CURRENT_DATE + 7)`
	c.sprintAlfa = id(sprint, c.timeAlfa)
	sprintBeta := id(sprint, c.timeBeta)

	atividade := `INSERT INTO atividades (projeto_id, time_id, sprint_id, titulo, tipo, complexidade, estimativa_desenvolvimento, criado_em)
		VALUES ($1, $2, $3, $4, 'melhoria', 'media', 8, $5)`
	agora := time.Now()
	a1 := id(atividade, c.p1, c.timeAlfa, c.sprintAlfa, "A1", agora.Add(-48*time.Hour))
	a2 := id(atividade, c.p1, c.timeBeta, sprintBeta, "A2", agora.Add(-48*time.Hour))
	a3 := id(atividade, c.p2, c.timeAlfa, c.sprintAlfa, "A3", agora.Add(-48*time.Hour))
	id(atividade, c.p2, c.timeAlfa, nil, "A4 (futura)", agora.Add(10*24*time.Hour))

	horas := `INSERT INTO registros_horas (projeto_id, atividade_id, desenvolvedor_id, etapa, horas, custo_hora) VALUES ($1, $2, $3, 'desenvolvimento', 2, 100)`
	id(horas, c.p1, a1, devA)
	id(horas, c.p1, a2, devB)
	id(horas, c.p2, a3, devA)

	tokens := `INSERT INTO consumo_tokens (projeto_id, atividade_id, desenvolvedor_id, etapa, modelo, tokens_entrada, tokens_saida, custo) VALUES ($1, $2, $3, 'desenvolvimento', $4, 10, 5, 0.1)`
	id(tokens, c.p1, a1, devA, "m1")
	id(tokens, c.p1, nil, devB, "m2")
	id(tokens, c.p2, a3, devA, "m1")
	id(tokens, c.p3, nil, devA, "m1")
	return c
}

func TestIntegracaoSemanticaEscopo(t *testing.T) {
	if os.Getenv("PULSO_TESTE_FIXTURE") != "1" {
		t.Skip("defina PULSO_TESTE_FIXTURE=1 para rodar (insere e desfaz dados numa transação)")
	}
	db := abrirBancoTeste(t)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback() // nada fica gravado

	c := montarCenario(t, tx)
	hoje := utils.Hoje()
	ptr := func(v int64) *int64 { return &v }

	// Todos os casos filtram por um id do cenário, então as contagens não
	// dependem de outros dados que o banco tenha.
	casos := []struct {
		nome       string
		f          models.Filtros
		p, a, h, t int64
		quota      int64
	}{
		{"projeto P1", models.Filtros{Escopo: "projeto", ID: ptr(c.p1)}, 1, 2, 2, 2, 1000},
		{"projeto P3 (sem atividades, com consumo)", models.Filtros{Escopo: "projeto", ID: ptr(c.p3)}, 1, 0, 0, 1, 200},
		{"time Alfa (A4 futura fica fora)", models.Filtros{Escopo: "time", ID: ptr(c.timeAlfa)}, 2, 2, 2, 2, 1500},
		{"time Beta", models.Filtros{Escopo: "time", ID: ptr(c.timeBeta)}, 1, 1, 1, 1, 1000},
		{"sprint Alfa", models.Filtros{Escopo: "sprint", ID: ptr(c.sprintAlfa)}, 2, 2, 2, 2, 1500},
		{"time Alfa + modelo m2 (nenhum consumo)", models.Filtros{Escopo: "time", ID: ptr(c.timeAlfa), Modelo: []string{"m2"}}, 2, 2, 2, 0, 1500},
		{"projeto P1 + status pausado (P1 sai)", models.Filtros{Escopo: "projeto", ID: ptr(c.p1), StatusProjeto: []string{"pausado"}}, 0, 0, 0, 0, 0},
		{"time Alfa + tipo api (só P2)", models.Filtros{Escopo: "time", ID: ptr(c.timeAlfa), TipoProjeto: []string{"api"}}, 1, 1, 1, 1, 500},
	}

	for _, cs := range casos {
		t.Run(cs.nome, func(t *testing.T) {
			f := cs.f
			f.De, f.Ate = utils.SomarDias(hoje, -29), hoje
			sqlc := &ConsultaSQL{}
			query := "WITH " + CTEsEscopo(sqlc, f) + `
				SELECT (SELECT COUNT(*) FROM p), (SELECT COUNT(*) FROM a),
				       (SELECT COUNT(*) FROM h), (SELECT COUNT(*) FROM t),
				       (SELECT valor FROM quota)`
			var p, a, h, tk, quota int64
			if err := tx.QueryRowContext(context.Background(), query, sqlc.Args()...).Scan(&p, &a, &h, &tk, &quota); err != nil {
				t.Fatalf("consulta falhou: %v", err)
			}
			if p != cs.p || a != cs.a || h != cs.h || tk != cs.t || quota != cs.quota {
				t.Errorf("esperado p=%d a=%d h=%d t=%d quota=%d; obtido p=%d a=%d h=%d t=%d quota=%d",
					cs.p, cs.a, cs.h, cs.t, cs.quota, p, a, h, tk, quota)
			}
		})
	}
}

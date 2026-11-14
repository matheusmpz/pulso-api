//go:build integration

// Testes de integração: rodam as consultas no PostgreSQL de verdade.
//
//	go test -tags integration ./repositories/
//
// Usam as variáveis DB_* do ambiente (ou do .env na raiz) e são pulados
// se elas não existirem. Só fazem SELECT.
package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"pulso-api/models"
	"pulso-api/utils"
)

func abrirBancoTeste(t *testing.T) *sql.DB {
	t.Helper()
	_ = godotenv.Load("../.env")
	if os.Getenv("DB_HOST") == "" || os.Getenv("DB_USER") == "" {
		t.Skip("DB_HOST/DB_USER não definidos: teste de integração pulado")
	}
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"), utils.EnvOuPadrao("DB_PORT", "5432"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), utils.EnvOuPadrao("DB_NAME", "pulso_db"), utils.EnvOuPadrao("DB_SSLMODE", "disable"))
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("banco indisponível: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestIntegracaoCTEsEscopo(t *testing.T) {
	db := abrirBancoTeste(t)
	id := int64(1)
	hoje := utils.Hoje()
	base := models.Filtros{De: utils.SomarDias(hoje, -29), Ate: hoje}

	casos := map[string]models.Filtros{}
	for _, escopo := range []string{"projeto", "time", "sprint"} {
		f := base
		f.Escopo, f.ID = escopo, &id
		casos[escopo+" com id"] = f
		if escopo != "sprint" {
			f.ID = nil
			casos[escopo+" sem id"] = f
		}
	}
	comListas := base
	comListas.Escopo = "projeto"
	comListas.TipoProjeto = []string{"api", "web"}
	comListas.StatusProjeto = []string{"em_andamento"}
	comListas.Modelo = []string{"qualquer"}
	casos["projeto com listas"] = comListas

	for nome, f := range casos {
		t.Run(nome, func(t *testing.T) {
			c := &ConsultaSQL{}
			query := "WITH " + CTEsEscopo(c, f) + `
				SELECT (SELECT COUNT(*) FROM p), (SELECT COUNT(*) FROM a),
				       (SELECT COUNT(*) FROM h), (SELECT COUNT(*) FROM t),
				       (SELECT valor FROM quota)`
			var np, na, nh, nt, quota int64
			if err := db.QueryRowContext(context.Background(), query, c.Args()...).Scan(&np, &na, &nh, &nt, &quota); err != nil {
				t.Fatalf("consulta falhou: %v", err)
			}
			t.Logf("p=%d a=%d h=%d t=%d quota=%d", np, na, nh, nt, quota)
		})
	}
}

func TestIntegracaoFiltroRepository(t *testing.T) {
	db := abrirBancoTeste(t)
	repo := NewFiltroRepository(db)
	ctx := context.Background()

	if _, err := repo.Origens(ctx); err != nil {
		t.Errorf("Origens: %v", err)
	}
	if _, err := repo.Modelos(ctx); err != nil {
		t.Errorf("Modelos: %v", err)
	}
	if _, err := repo.Projetos(ctx); err != nil {
		t.Errorf("Projetos: %v", err)
	}
	if _, err := repo.Times(ctx); err != nil {
		t.Errorf("Times: %v", err)
	}
	sprints, err := repo.Sprints(ctx)
	if err != nil {
		t.Fatalf("Sprints: %v", err)
	}
	if _, err := repo.BuscarSprint(ctx, -1); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("sprint inexistente deveria dar sql.ErrNoRows, obtido %v", err)
	}
	if len(sprints) > 0 {
		s, err := repo.BuscarSprint(ctx, sprints[0].ID)
		if err != nil || s.Inicio.String() != sprints[0].Inicio.String() {
			t.Errorf("BuscarSprint inconsistente: %v %+v", err, s)
		}
	}
}

func TestIntegracaoCardRepository(t *testing.T) {
	db := abrirBancoTeste(t)
	repo := NewCardRepository(db)
	ctx := context.Background()

	codigos, err := repo.ListarCodigosAtivos(ctx)
	if err != nil {
		t.Fatalf("ListarCodigosAtivos: %v", err)
	}
	t.Logf("%d cards ativos", len(codigos))

	if _, err := repo.BuscarAcesso(ctx, "nao-existe", 1); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("card inexistente deveria dar sql.ErrNoRows, obtido %v", err)
	}
	if len(codigos) > 0 {
		a, err := repo.BuscarAcesso(ctx, codigos[0], 1)
		if err != nil || a.Codigo != codigos[0] || !a.Ativo || a.DashboardChave == "" {
			t.Errorf("BuscarAcesso inconsistente: %v %+v", err, a)
		}
	}
}

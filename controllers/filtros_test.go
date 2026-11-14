package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"pulso-api/middlewares"
	"pulso-api/models"
)

type repoFiltroFalso struct {
	errEm   string // nome do método que deve falhar
	vazio   bool
	sprints []models.Sprint
}

func (r *repoFiltroFalso) falha(metodo string) error {
	if r.errEm == metodo {
		return errors.New("caiu")
	}
	return nil
}

func (r *repoFiltroFalso) Origens(context.Context) ([]string, error) {
	if r.vazio {
		return []string{}, r.falha("origens")
	}
	return []string{"indicacao", "site"}, r.falha("origens")
}

func (r *repoFiltroFalso) Modelos(context.Context) ([]string, error) {
	return []string{"modelo-a"}, r.falha("modelos")
}

func (r *repoFiltroFalso) Projetos(context.Context) ([]models.ProjetoOpcao, error) {
	return []models.ProjetoOpcao{{ID: 1, Nome: "Portal", Tipo: "web", Status: "em_andamento"}}, r.falha("projetos")
}

func (r *repoFiltroFalso) Times(context.Context) ([]models.TimeOpcao, error) {
	return []models.TimeOpcao{{ID: 1, Nome: "Alfa"}}, r.falha("times")
}

func (r *repoFiltroFalso) Sprints(context.Context) ([]models.Sprint, error) {
	return r.sprints, r.falha("sprints")
}

func (r *repoFiltroFalso) BuscarSprint(context.Context, int64) (models.Sprint, error) {
	return models.Sprint{}, nil
}

func dia(s string) models.Data {
	t, _ := time.Parse(models.FormatoData, s)
	return models.Data{Time: t}
}

func TestMarcarSprintAtual(t *testing.T) {
	sprints := []models.Sprint{
		{ID: 1, Inicio: dia("2026-10-20"), Fim: dia("2026-10-31")},
		{ID: 2, Inicio: dia("2026-11-03"), Fim: dia("2026-11-14")},
		{ID: 3, Inicio: dia("2026-11-17"), Fim: dia("2026-11-28")},
	}
	casos := map[string][]bool{
		"2026-11-14": {false, true, false}, // último dia conta
		"2026-11-03": {false, true, false}, // primeiro dia conta
		"2026-11-15": {false, false, false},
		"2026-10-25": {true, false, false},
	}
	for hoje, esperado := range casos {
		opcoes := marcarSprintAtual(sprints, dia(hoje))
		for i, o := range opcoes {
			if o.Atual != esperado[i] {
				t.Errorf("hoje=%s sprint %d: atual esperado %v, obtido %v", hoje, o.ID, esperado[i], o.Atual)
			}
		}
	}
}

func TestOpcoesFiltros(t *testing.T) {
	gin.SetMode(gin.TestMode)
	token := tokenDoPerfil(t, 2, "lider_tecnologia")

	casos := []struct {
		nome   string
		repo   *repoFiltroFalso
		status int
	}{
		{"sucesso", &repoFiltroFalso{sprints: []models.Sprint{{ID: 2, TimeID: 1, TimeNome: "Alfa", Numero: 5, Inicio: dia("2026-11-03"), Fim: dia("2026-11-14")}}}, http.StatusOK},
		{"banco sem dados", &repoFiltroFalso{vazio: true, sprints: []models.Sprint{}}, http.StatusOK},
		{"erro nas sprints", &repoFiltroFalso{errEm: "sprints"}, http.StatusInternalServerError},
		{"erro nas origens", &repoFiltroFalso{errEm: "origens"}, http.StatusInternalServerError},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			ctrl := &FiltroController{Repo: c.repo, Hoje: func() models.Data { return dia("2026-11-14") }}
			r := gin.New()
			r.GET("/filtros/opcoes", middlewares.AuthMiddleware(cfgJWTTeste), ctrl.Opcoes)

			status, resp := executar(r, http.MethodGet, "/filtros/opcoes", "", token)
			if status != c.status {
				t.Fatalf("status esperado %d, obtido %d", c.status, status)
			}
			if c.status != http.StatusOK {
				return
			}

			// Confere o JSON bruto: listas nunca null e datas AAAA-MM-DD
			var bruto map[string]json.RawMessage
			if err := json.Unmarshal(resp.Dados, &bruto); err != nil {
				t.Fatal(err)
			}
			for _, campo := range []string{"tipos_projeto", "status_projeto", "origens", "modelos", "projetos", "times", "sprints"} {
				if v, ok := bruto[campo]; !ok || string(v) == "null" {
					t.Errorf("%s ausente ou null", campo)
				}
			}
			if len(c.repo.sprints) > 0 {
				esperado := `[{"id":2,"time_id":1,"time_nome":"Alfa","numero":5,"inicio":"2026-11-03","fim":"2026-11-14","atual":true}]`
				if string(bruto["sprints"]) != esperado {
					t.Errorf("sprints:\nesperado %s\nobtido   %s", esperado, bruto["sprints"])
				}
			}
			if string(bruto["tipos_projeto"]) == "" || len(bruto["tipos_projeto"]) < 10 {
				t.Error("tipos_projeto deveria trazer os 5 tipos com rótulo")
			}
		})
	}

	// Sem token
	ctrl := &FiltroController{Repo: &repoFiltroFalso{}}
	r := gin.New()
	r.GET("/filtros/opcoes", middlewares.AuthMiddleware(cfgJWTTeste), ctrl.Opcoes)
	if status, _ := executar(r, http.MethodGet, "/filtros/opcoes", "", ""); status != http.StatusUnauthorized {
		t.Errorf("sem token: esperado 401, obtido %d", status)
	}
}

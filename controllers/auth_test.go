package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"pulso-api/middlewares"
	"pulso-api/models"
	"pulso-api/utils"
)

// repoUsuarioFalso simula o banco: um mapa de e-mail (minúsculo) → usuário.
type repoUsuarioFalso struct {
	usuarios map[string]*models.Usuario
	err      error
}

func (r *repoUsuarioFalso) BuscarAtivoPorEmail(_ context.Context, email string) (*models.Usuario, error) {
	if r.err != nil {
		return nil, r.err
	}
	if u, ok := r.usuarios[strings.ToLower(email)]; ok && u.Ativo {
		return u, nil
	}
	return nil, sql.ErrNoRows
}

func (r *repoUsuarioFalso) BuscarAtivoPorID(_ context.Context, id string) (*models.Usuario, error) {
	if r.err != nil {
		return nil, r.err
	}
	for _, u := range r.usuarios {
		if u.ID == id && u.Ativo {
			return u, nil
		}
	}
	return nil, sql.ErrNoRows
}

var cfgJWTTeste = utils.ConfigJWT{Segredo: []byte("segredo-de-teste"), Duracao: time.Hour}

func novoRepoTeste(t *testing.T) *repoUsuarioFalso {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("senha-certa"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return &repoUsuarioFalso{usuarios: map[string]*models.Usuario{
		"diretoria@pulso.local": {
			ID: "uuid-dir", Nome: "Diretoria", Email: "diretoria@pulso.local",
			SenhaHash: string(hash), PerfilID: 1, PerfilNome: "diretoria",
			PerfilDescricao: "Tudo", Ativo: true,
		},
		"inativo@pulso.local": {
			ID: "uuid-ina", Nome: "Inativo", Email: "inativo@pulso.local",
			SenhaHash: string(hash), PerfilID: 1, PerfilNome: "diretoria", Ativo: false,
		},
	}}
}

func novoRouterAuth(repo *repoUsuarioFalso) *gin.Engine {
	gin.SetMode(gin.TestMode)
	ctrl := &AuthController{Repo: repo, JWT: cfgJWTTeste}
	r := gin.New()
	r.POST("/auth/login", ctrl.Login)
	r.GET("/auth/me", middlewares.AuthMiddleware(cfgJWTTeste), ctrl.Me)
	return r
}

type respostaTeste struct {
	Sucesso bool            `json:"sucesso"`
	Dados   json.RawMessage `json:"dados"`
	Erro    string          `json:"erro"`
}

func executar(r *gin.Engine, metodo, caminho, corpo, token string) (int, respostaTeste) {
	req := httptest.NewRequest(metodo, caminho, strings.NewReader(corpo))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var resp respostaTeste
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w.Code, resp
}

func TestLogin(t *testing.T) {
	casos := []struct {
		nome     string
		corpo    string
		repoErr  error
		status   int
		mensagem string
	}{
		{"sucesso", `{"email":"diretoria@pulso.local","senha":"senha-certa"}`, nil, http.StatusOK, ""},
		{"e-mail com maiúsculas e espaços", `{"email":"  Diretoria@Pulso.Local ","senha":"senha-certa"}`, nil, http.StatusOK, ""},
		{"senha errada", `{"email":"diretoria@pulso.local","senha":"errada"}`, nil, http.StatusUnauthorized, msgLoginInvalido},
		{"e-mail inexistente", `{"email":"ninguem@pulso.local","senha":"senha-certa"}`, nil, http.StatusUnauthorized, msgLoginInvalido},
		{"usuário inativo", `{"email":"inativo@pulso.local","senha":"senha-certa"}`, nil, http.StatusUnauthorized, msgLoginInvalido},
		{"corpo inválido", `{email`, nil, http.StatusBadRequest, "Informe e-mail e senha."},
		{"senha vazia", `{"email":"diretoria@pulso.local","senha":""}`, nil, http.StatusBadRequest, "Informe e-mail e senha."},
		{"e-mail só com espaços", `{"email":"   ","senha":"x"}`, nil, http.StatusBadRequest, "Informe e-mail e senha."},
		{"erro no banco", `{"email":"diretoria@pulso.local","senha":"senha-certa"}`, errors.New("conexão caiu"), http.StatusInternalServerError, msgErroInterno},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			repo := novoRepoTeste(t)
			repo.err = c.repoErr
			status, resp := executar(novoRouterAuth(repo), http.MethodPost, "/auth/login", c.corpo, "")

			if status != c.status {
				t.Fatalf("status esperado %d, obtido %d (%s)", c.status, status, resp.Erro)
			}
			if c.status != http.StatusOK {
				if resp.Sucesso || resp.Erro != c.mensagem {
					t.Errorf("erro esperado %q, obtido %q", c.mensagem, resp.Erro)
				}
				return
			}

			var dados models.LoginResposta
			if err := json.Unmarshal(resp.Dados, &dados); err != nil {
				t.Fatalf("dados inválidos: %v", err)
			}
			if dados.Usuario.ID != "uuid-dir" || dados.Usuario.Perfil != "diretoria" || dados.Token == "" {
				t.Errorf("resposta inesperada: %+v", dados)
			}
			if strings.Contains(string(resp.Dados), "senha") {
				t.Error("a resposta não pode conter dados de senha")
			}
			claims, err := utils.ValidarToken(cfgJWTTeste, dados.Token)
			if err != nil || claims.UsuarioID != "uuid-dir" || claims.PerfilID != 1 {
				t.Errorf("token devolvido inválido: %v %+v", err, claims)
			}
		})
	}
}

func TestMe(t *testing.T) {
	tokenDir, _, _ := utils.GerarToken(cfgJWTTeste, "uuid-dir", 1, "diretoria", "diretoria@pulso.local", time.Now())
	tokenInativo, _, _ := utils.GerarToken(cfgJWTTeste, "uuid-ina", 1, "diretoria", "inativo@pulso.local", time.Now())

	casos := []struct {
		nome    string
		token   string
		repoErr error
		status  int
	}{
		{"sucesso", tokenDir, nil, http.StatusOK},
		{"sem token", "", nil, http.StatusUnauthorized},
		{"usuário desativado depois do login", tokenInativo, nil, http.StatusUnauthorized},
		{"erro no banco", tokenDir, errors.New("conexão caiu"), http.StatusInternalServerError},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			repo := novoRepoTeste(t)
			repo.err = c.repoErr
			status, resp := executar(novoRouterAuth(repo), http.MethodGet, "/auth/me", "", c.token)

			if status != c.status {
				t.Fatalf("status esperado %d, obtido %d (%s)", c.status, status, resp.Erro)
			}
			if c.status != http.StatusOK {
				return
			}

			var dados models.MeResposta
			if err := json.Unmarshal(resp.Dados, &dados); err != nil {
				t.Fatalf("dados inválidos: %v", err)
			}
			esperado := models.MeResposta{
				ID: "uuid-dir", Nome: "Diretoria", Email: "diretoria@pulso.local",
				Perfil: models.PerfilResumo{ID: 1, Nome: "diretoria", Descricao: "Tudo"},
			}
			if dados != esperado {
				t.Errorf("esperado %+v, obtido %+v", esperado, dados)
			}
		})
	}
}

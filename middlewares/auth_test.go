package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"pulso-api/utils"
)

var cfgTeste = utils.ConfigJWT{Segredo: []byte("segredo-de-teste"), Duracao: time.Hour}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	valido, _, _ := utils.GerarToken(cfgTeste, "uuid-1", 7, "diretoria", "d@pulso.local", time.Now())
	expirado, _, _ := utils.GerarToken(cfgTeste, "uuid-1", 7, "diretoria", "d@pulso.local", time.Now().Add(-2*time.Hour))

	casos := []struct {
		nome      string
		cabecalho string
		status    int
	}{
		{"sem cabeçalho", "", http.StatusUnauthorized},
		{"sem prefixo Bearer", valido, http.StatusUnauthorized},
		{"outro esquema", "Basic " + valido, http.StatusUnauthorized},
		{"token inválido", "Bearer abc", http.StatusUnauthorized},
		{"token expirado", "Bearer " + expirado, http.StatusUnauthorized},
		{"token válido", "Bearer " + valido, http.StatusOK},
		{"prefixo em minúsculas", "bearer " + valido, http.StatusOK},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			var usuarioID, perfil string
			var perfilID int64

			r := gin.New()
			r.GET("/protegida", AuthMiddleware(cfgTeste), func(ctx *gin.Context) {
				usuarioID = ctx.GetString(ChaveUsuarioID)
				perfilID = ctx.GetInt64(ChavePerfilID)
				perfil = ctx.GetString(ChavePerfil)
				ctx.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/protegida", nil)
			if c.cabecalho != "" {
				req.Header.Set("Authorization", c.cabecalho)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != c.status {
				t.Fatalf("status esperado %d, obtido %d", c.status, w.Code)
			}
			if c.status == http.StatusOK && (usuarioID != "uuid-1" || perfilID != 7 || perfil != "diretoria") {
				t.Errorf("contexto inesperado: usuario=%q perfil_id=%d perfil=%q", usuarioID, perfilID, perfil)
			}
		})
	}
}

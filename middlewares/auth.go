package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"pulso-api/utils"
)

// Chaves gravadas no contexto do Gin pelo middleware.
const (
	ChaveUsuarioID = "usuario_id"
	ChavePerfilID  = "perfil_id"
	ChavePerfil    = "perfil"
)

// AuthMiddleware lê "Authorization: Bearer <token>", valida o JWT e
// injeta usuario_id, perfil_id e perfil no contexto. Sem token ou com
// token inválido, responde 401 e interrompe a requisição.
func AuthMiddleware(cfg utils.ConfigJWT) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, ok := extrairBearer(c.GetHeader("Authorization"))
		if !ok {
			utils.RespostaErro(c, http.StatusUnauthorized, "Token não informado.")
			c.Abort()
			return
		}

		claims, err := utils.ValidarToken(cfg, token)
		if err != nil {
			utils.RespostaErro(c, http.StatusUnauthorized, "Token inválido ou expirado.")
			c.Abort()
			return
		}

		c.Set(ChaveUsuarioID, claims.UsuarioID)
		c.Set(ChavePerfilID, claims.PerfilID)
		c.Set(ChavePerfil, claims.Perfil)
		c.Next()
	}
}

// extrairBearer separa o token do cabeçalho "Bearer <token>"
// (o prefixo não diferencia maiúsculas).
func extrairBearer(cabecalho string) (string, bool) {
	partes := strings.Fields(cabecalho)
	if len(partes) != 2 || !strings.EqualFold(partes[0], "Bearer") {
		return "", false
	}
	return partes[1], true
}

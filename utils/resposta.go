package utils

import "github.com/gin-gonic/gin"

// RespostaSucesso devolve { "sucesso": true, "dados": ... }
func RespostaSucesso(c *gin.Context, status int, dados interface{}) {
	c.JSON(status, gin.H{
		"sucesso": true,
		"dados":   dados,
	})
}

// RespostaErro devolve { "sucesso": false, "erro": "mensagem" }
func RespostaErro(c *gin.Context, status int, mensagem string) {
	c.JSON(status, gin.H{
		"sucesso": false,
		"erro":    mensagem,
	})
}

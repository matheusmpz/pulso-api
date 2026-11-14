package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RegisterSwagger serve o Swagger UI em /swagger/index.html (público).
// A documentação vem do pacote docs, gerado pelo swag (ver README).
func RegisterSwagger(r *gin.Engine) {
	// /swagger sozinho redireciona para a página do Swagger UI
	r.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.PersistAuthorization(true), // mantém o token do "Authorize" ao recarregar
		ginSwagger.DocExpansion("list"),       // abre a lista de rotas, sem expandir cada uma
	))
}

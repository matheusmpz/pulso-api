// Pulso API — ponto de entrada.
//
// Carrega o .env, valida as variáveis obrigatórias, abre o pool do banco,
// sobe o Gin com CORS e as rotas e encerra de forma graciosa.
// Autenticação, cálculos dos cards e tempo real entram nas próximas etapas.
//
// @title          Pulso API
// @version        1.0
// @description    API do Pulso, o dashboard de BI da Fábrica de Software.
// @BasePath       /
//
// @securityDefinitions.apikey BearerAuth
// @in             header
// @name           Authorization
// @description    Informe o valor no formato "Bearer <token>".
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata" // embute a base de fusos no binário (não depende do sistema)

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"pulso-api/db"
	"pulso-api/routes"
	"pulso-api/utils"
)

// origemPadrao é usada quando CORS_ORIGINS está vazio (o gin-contrib/cors
// não aceita lista vazia).
const origemPadrao = "http://localhost:3000"

// tempoEncerramento é o prazo para as requisições em andamento terminarem
// depois de um SIGINT/SIGTERM.
const tempoEncerramento = 10 * time.Second

func main() {
	// Carrega o .env se existir. Em produção as variáveis vêm do ambiente
	// (12-factor), então a ausência do arquivo não é erro.
	if err := godotenv.Load(); err != nil {
		log.Println("Arquivo .env não encontrado; usando as variáveis do ambiente")
	}

	// Falha logo na inicialização se faltar variável obrigatória
	if err := utils.ValidarAmbiente(); err != nil {
		log.Fatal("Configuração inválida: ", err)
	}

	// Segredo e validade do JWT
	jwtCfg, err := utils.CarregarConfigJWT()
	if err != nil {
		log.Fatal("Configuração inválida: ", err)
	}

	// Fuso usado nos limites de dia e nos timestamps das respostas
	if err := utils.ConfigurarFuso(utils.EnvOuPadrao("APP_TIMEZONE", utils.FusoPadrao)); err != nil {
		log.Fatal("Configuração inválida: ", err)
	}

	// Inicializa o banco e o pool antes de receber qualquer requisição
	database := db.GetDB()

	// Modo do Gin via variável de ambiente ou padrão release
	modo := os.Getenv("GIN_MODE")
	if modo == "" {
		modo = gin.ReleaseMode
	}
	gin.SetMode(modo)

	// Inicia o Gin
	r := gin.Default()
	r.SetTrustedProxies(nil)

	r.Use(cors.New(cors.Config{
		AllowOrigins:     lerOrigens(os.Getenv("CORS_ORIGINS")),
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Registra todas as rotas passando o banco e a configuração do JWT
	routes.Register(r, database, jwtCfg)

	// Porta via variável de ambiente ou padrão 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Sobe o servidor em uma goroutine para o main poder esperar o sinal
	go func() {
		log.Printf("Pulso API ouvindo na porta %s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("Erro ao iniciar o servidor: ", err)
		}
	}()

	// Espera SIGINT (Ctrl+C) ou SIGTERM e encerra de forma graciosa
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Println("Encerrando o servidor...")
	ctxEncerrar, cancelar := context.WithTimeout(context.Background(), tempoEncerramento)
	defer cancelar()

	if err := srv.Shutdown(ctxEncerrar); err != nil {
		log.Println("Erro ao encerrar o servidor: ", err)
	}
	if err := database.Close(); err != nil {
		log.Println("Erro ao fechar o banco: ", err)
	}
	log.Println("Servidor encerrado")
}

// lerOrigens transforma CORS_ORIGINS (separado por vírgula) em lista,
// ignorando espaços e itens vazios. Sem nenhuma origem, usa origemPadrao.
func lerOrigens(valor string) []string {
	var origens []string
	for _, o := range strings.Split(valor, ",") {
		if o = strings.TrimSpace(o); o != "" {
			origens = append(origens, o)
		}
	}
	if len(origens) == 0 {
		origens = []string{origemPadrao}
	}
	return origens
}

package controllers

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"pulso-api/middlewares"
	"pulso-api/models"
	"pulso-api/repositories"
	"pulso-api/utils"
)

// Mensagens fixas, para não revelar se o e-mail existe ou se a senha errou.
const (
	msgLoginInvalido  = "E-mail ou senha inválidos"
	msgSessaoInvalida = "Sessão inválida. Faça login novamente."
	msgErroInterno    = "Erro interno. Tente novamente mais tarde."
)

// AuthController trata login e dados do usuário logado.
type AuthController struct {
	Repo repositories.UsuarioRepository
	JWT  utils.ConfigJWT
	// Agora devolve o horário atual (substituível nos testes).
	Agora func() time.Time
}

// LoginRespostaDoc é o envelope de sucesso do login (só para o Swagger).
type LoginRespostaDoc struct {
	Sucesso bool                 `json:"sucesso" example:"true"`
	Dados   models.LoginResposta `json:"dados"`
}

// MeRespostaDoc é o envelope de sucesso do /auth/me (só para o Swagger).
type MeRespostaDoc struct {
	Sucesso bool              `json:"sucesso" example:"true"`
	Dados   models.MeResposta `json:"dados"`
}

// ErroDoc é o envelope de erro (só para o Swagger).
type ErroDoc struct {
	Sucesso bool   `json:"sucesso" example:"false"`
	Erro    string `json:"erro" example:"mensagem de erro"`
}

// Login godoc
//
// POST /auth/login — autentica com e-mail e senha e devolve o JWT
//
// @Summary      Login
// @Description  Autentica com e-mail e senha de um usuário ativo e devolve um JWT (HS256). Qualquer falha de credencial devolve 401 com a mesma mensagem genérica.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      models.LoginInput  true  "Credenciais"
// @Success      200   {object}  LoginRespostaDoc
// @Failure      400   {object}  ErroDoc  "Corpo inválido ou campos vazios"
// @Failure      401   {object}  ErroDoc  "E-mail ou senha inválidos"
// @Failure      500   {object}  ErroDoc
// @Router       /auth/login [post]
func (h *AuthController) Login(c *gin.Context) {
	var input models.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil || strings.TrimSpace(input.Email) == "" {
		utils.RespostaErro(c, http.StatusBadRequest, "Informe e-mail e senha.")
		return
	}

	usuario, err := h.Repo.BuscarAtivoPorEmail(c.Request.Context(), strings.TrimSpace(input.Email))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Compara com um hash qualquer para o tempo de resposta ser
			// parecido com o de um e-mail existente.
			_ = bcrypt.CompareHashAndPassword(hashFicticio(), []byte(input.Senha))
			utils.RespostaErro(c, http.StatusUnauthorized, msgLoginInvalido)
			return
		}
		log.Println("Login: erro ao buscar usuário: ", err)
		utils.RespostaErro(c, http.StatusInternalServerError, msgErroInterno)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(usuario.SenhaHash), []byte(input.Senha)); err != nil {
		utils.RespostaErro(c, http.StatusUnauthorized, msgLoginInvalido)
		return
	}

	token, expiraEm, err := utils.GerarToken(h.JWT, usuario.ID, usuario.PerfilID, usuario.PerfilNome, usuario.Email, h.agora())
	if err != nil {
		log.Println("Login: erro ao gerar token: ", err)
		utils.RespostaErro(c, http.StatusInternalServerError, msgErroInterno)
		return
	}

	utils.RespostaSucesso(c, http.StatusOK, models.LoginResposta{
		Token:    token,
		ExpiraEm: expiraEm.In(utils.FusoApp()),
		Usuario: models.UsuarioResumo{
			ID:     usuario.ID,
			Nome:   usuario.Nome,
			Email:  usuario.Email,
			Perfil: usuario.PerfilNome,
		},
	})
}

// Me godoc
//
// GET /auth/me — dados do usuário logado e do perfil
//
// @Summary      Usuário logado
// @Description  Devolve os dados do usuário dono do token e do perfil dele.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200  {object}  MeRespostaDoc
// @Failure      401  {object}  ErroDoc  "Sem token, token inválido ou usuário inativo"
// @Failure      500  {object}  ErroDoc
// @Security     BearerAuth
// @Router       /auth/me [get]
func (h *AuthController) Me(c *gin.Context) {
	usuarioID := c.GetString(middlewares.ChaveUsuarioID)
	if usuarioID == "" {
		utils.RespostaErro(c, http.StatusUnauthorized, msgSessaoInvalida)
		return
	}

	usuario, err := h.Repo.BuscarAtivoPorID(c.Request.Context(), usuarioID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// O token é válido, mas o usuário foi desativado
			utils.RespostaErro(c, http.StatusUnauthorized, msgSessaoInvalida)
			return
		}
		log.Println("Me: erro ao buscar usuário: ", err)
		utils.RespostaErro(c, http.StatusInternalServerError, msgErroInterno)
		return
	}

	utils.RespostaSucesso(c, http.StatusOK, models.MeResposta{
		ID:    usuario.ID,
		Nome:  usuario.Nome,
		Email: usuario.Email,
		Perfil: models.PerfilResumo{
			ID:        usuario.PerfilID,
			Nome:      usuario.PerfilNome,
			Descricao: usuario.PerfilDescricao,
		},
	})
}

func (h *AuthController) agora() time.Time {
	if h.Agora != nil {
		return h.Agora()
	}
	return time.Now()
}

var (
	hashFicticioOnce  sync.Once
	hashFicticioValor []byte
)

// hashFicticio é um hash bcrypt de um valor aleatório, gerado uma vez,
// usado só para igualar o tempo de resposta do login.
func hashFicticio() []byte {
	hashFicticioOnce.Do(func() {
		h, err := bcrypt.GenerateFromPassword([]byte(time.Now().String()), bcrypt.DefaultCost)
		if err == nil {
			hashFicticioValor = h
		}
	})
	return hashFicticioValor
}

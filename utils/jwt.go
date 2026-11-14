package utils

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// emissorToken vai no claim "iss" e é conferido na validação.
const emissorToken = "pulso-api"

// duracaoPadraoToken é usada quando JWT_EXPIRA_HORAS não é informado.
const duracaoPadraoToken = 24 * time.Hour

// ConfigJWT reúne o segredo e a validade dos tokens.
type ConfigJWT struct {
	Segredo []byte
	Duracao time.Duration
}

// Claims são os dados gravados no token.
type Claims struct {
	UsuarioID string `json:"usuario_id"`
	PerfilID  int64  `json:"perfil_id"`
	Perfil    string `json:"perfil"`
	Email     string `json:"email"`
	jwt.RegisteredClaims
}

// CarregarConfigJWT lê JWT_SECRET e JWT_EXPIRA_HORAS do ambiente.
// Devolve erro se o segredo faltar ou se a validade for inválida.
func CarregarConfigJWT() (ConfigJWT, error) {
	segredo := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if segredo == "" {
		return ConfigJWT{}, errors.New("JWT_SECRET não definido")
	}
	duracao, err := ParseHorasExpiracao(os.Getenv("JWT_EXPIRA_HORAS"))
	if err != nil {
		return ConfigJWT{}, err
	}
	return ConfigJWT{Segredo: []byte(segredo), Duracao: duracao}, nil
}

// ParseHorasExpiracao converte JWT_EXPIRA_HORAS (inteiro positivo) em
// duração. Vazio usa o padrão de 24 h.
func ParseHorasExpiracao(valor string) (time.Duration, error) {
	valor = strings.TrimSpace(valor)
	if valor == "" {
		return duracaoPadraoToken, nil
	}
	horas, err := strconv.Atoi(valor)
	if err != nil || horas <= 0 {
		return 0, fmt.Errorf("JWT_EXPIRA_HORAS inválido (%q): use um número inteiro de horas maior que zero", valor)
	}
	return time.Duration(horas) * time.Hour, nil
}

// GerarToken cria um JWT HS256 para o usuário e devolve o token e a
// data de expiração.
func GerarToken(cfg ConfigJWT, usuarioID string, perfilID int64, perfil, email string, agora time.Time) (string, time.Time, error) {
	// O JWT guarda datas em segundos; trunca para a resposta bater com o token
	agora = agora.Truncate(time.Second)
	expiraEm := agora.Add(cfg.Duracao)
	claims := Claims{
		UsuarioID: usuarioID,
		PerfilID:  perfilID,
		Perfil:    perfil,
		Email:     email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   usuarioID,
			Issuer:    emissorToken,
			IssuedAt:  jwt.NewNumericDate(agora),
			ExpiresAt: jwt.NewNumericDate(expiraEm),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(cfg.Segredo)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiraEm, nil
}

// ValidarToken confere assinatura (só HS256), emissor e expiração e
// devolve os claims. Qualquer problema vira erro.
func ValidarToken(cfg ConfigJWT, tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims,
		func(*jwt.Token) (interface{}, error) { return cfg.Segredo, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(emissorToken),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	if !token.Valid || claims.UsuarioID == "" {
		return nil, errors.New("token inválido")
	}
	return claims, nil
}

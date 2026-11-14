# =====================================================================
# Dockerfile da Pulso API — multi-stage
#
# "Multi-stage" = duas imagens em sequência no mesmo arquivo:
#   1. "build": imagem grande, com o Go instalado, só para compilar.
#   2. final:   imagem pequena (alpine), só com o binário pronto.
# A imagem final não leva o compilador nem o código-fonte, então fica
# leve (dezenas de MB em vez de ~1 GB) e com menos coisa para atacar.
# =====================================================================

# ---------------------------------------------------------------------
# Estágio 1: compilação
# ---------------------------------------------------------------------
# "AS build" dá um nome ao estágio, para o estágio 2 copiar daqui.
FROM golang:1.25 AS build

# Pasta de trabalho dentro do container (criada se não existir).
WORKDIR /src

# Copia SÓ go.mod e go.sum primeiro e baixa as dependências.
# O Docker guarda cada instrução em cache: enquanto esses dois arquivos
# não mudarem, o download é reaproveitado e o build fica bem mais rápido.
COPY go.mod go.sum ./
RUN go mod download

# Agora copia o resto do código (o .dockerignore define o que fica de fora).
COPY . .

# Compila um binário estático:
#   CGO_ENABLED=0 → sem dependência de bibliotecas C, roda em qualquer Linux
#   GOOS=linux    → compila para Linux, mesmo que o host seja Windows/Mac
#   -trimpath     → tira os caminhos da sua máquina do binário
#   -ldflags      → "-s -w" remove símbolos de debug (binário menor)
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/pulso-api ./cmd/api

# ---------------------------------------------------------------------
# Estágio 2: execução
# ---------------------------------------------------------------------
FROM alpine:3.21

# ca-certificates → certificados para conexões TLS (ex.: banco com SSL)
# tzdata          → base de fusos horários (America/Sao_Paulo)
# --no-cache      → não guarda o índice do apk, a imagem fica menor
RUN apk add --no-cache ca-certificates tzdata

# Cria um usuário comum, sem privilégio de root. Se alguém explorar uma
# falha na API, ele não terá poder de administrador dentro do container.
#   -S → usuário/grupo de sistema (sem senha, sem shell de login)
RUN addgroup -S pulso && adduser -S -G pulso pulso

WORKDIR /app

# Traz do estágio "build" apenas o binário compilado.
COPY --from=build /out/pulso-api /app/pulso-api

# Valores padrão (o docker-compose e o .env podem sobrescrever).
ENV PORT=8080 \
    GIN_MODE=release

# Documenta a porta em que a API escuta. Quem publica a porta para fora
# do container é o "ports" do docker-compose (ou "docker run -p").
EXPOSE 8080

# A partir daqui tudo roda como o usuário sem privilégio.
USER pulso

# Healthcheck: a cada 30 s o Docker chama GET /health dentro do container.
# Se falhar 3 vezes seguidas, o container fica "unhealthy".
#   --start-period → tempo de tolerância enquanto a API sobe
#   wget já vem no alpine (busybox); ${PORT} é lido na hora da checagem
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD wget -qO- "http://localhost:${PORT}/health" > /dev/null || exit 1

# Comando que inicia o container.
ENTRYPOINT ["/app/pulso-api"]

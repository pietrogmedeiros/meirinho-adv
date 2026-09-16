# Um Dockerfile para todos os binários: os serviços compartilham o mesmo módulo
# e as mesmas dependências, então build separado por serviço só multiplicaria
# cache frio. CMD_PATH escolhe qual binário sai.
FROM golang:1.26-alpine AS build

WORKDIR /src

# go.mod/go.sum primeiro: a camada de dependências só invalida quando elas
# mudam, não a cada alteração de código.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG CMD_PATH=./services/auth-service
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/servico ${CMD_PATH}

FROM alpine:3.21

# ca-certificates: as chamadas à API do Claude e ao storage são HTTPS.
# tzdata: prazo processual é contado em horário de Brasília.
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 meirinho
ENV TZ=America/Sao_Paulo

COPY --from=build /out/servico /usr/local/bin/servico

USER meirinho
ENTRYPOINT ["/usr/local/bin/servico"]

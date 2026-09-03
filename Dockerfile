# Imagem de build: compila o binário estático usando Go sobre Alpine.
FROM golang:1.24-alpine AS builder

# Define o diretório de trabalho durante a compilação.
WORKDIR /src

# Copia o código-fonte para dentro da imagem de build.
COPY main.go .

# Gera um executável Linux sem dependências de CGO e reduz seus metadados.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/app main.go

# Imagem final mínima, sem shell ou bibliotecas desnecessárias.
FROM scratch

# Copia somente o executável produzido na etapa de build.
COPY --from=builder /out/app /app

# Executa o processo com um usuário sem privilégios de root.
USER 65532:65532

# Documenta a porta escutada pelo processo dentro do container.
EXPOSE 8080

# Define o executável iniciado na criação do container.
ENTRYPOINT ["/app"]
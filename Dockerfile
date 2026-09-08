FROM golang:1.27.1 AS construcao
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /azamon ./cmd/azamon

# Binário estático: nada além dele no contêiner.
FROM scratch
COPY --from=construcao /azamon /azamon
EXPOSE 8080
ENTRYPOINT ["/azamon"]

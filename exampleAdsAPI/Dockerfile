FROM golang:1.26-bookworm

WORKDIR /app

RUN go install github.com/air-verse/air@v1.67.4

COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 8081

CMD ["air", "-c", ".air.toml"]

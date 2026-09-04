FROM golang:1.26-bookworm

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENTRYPOINT ["go", "run", "-mod=vendor", "."]

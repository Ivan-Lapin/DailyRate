FROM golang:latest AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o currency_service ./currency/cmd/currency

FROM golang:latest 

WORKDIR /app

COPY --from=builder /app/currency_service .

COPY ./currency/internal/config/config.example.yaml ./currency/internal/config/config.example.yaml

EXPOSE 8888

ENTRYPOINT [ "./currency_service" ]
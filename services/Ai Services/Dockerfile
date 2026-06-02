FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/ai-services .

FROM alpine:3.20

WORKDIR /app

RUN addgroup -S app && adduser -S app -G app

COPY --from=builder /out/ai-services /app/ai-services
COPY prompts /app/prompts

RUN mkdir -p /app/response_csv /app/response_json /app/base_level_tables \
    && chown -R app:app /app

ENV SERVER_PORT=8888

EXPOSE 8888

USER app

CMD ["./ai-services"]

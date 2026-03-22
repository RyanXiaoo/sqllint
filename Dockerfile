FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /sqllint ./cmd/sqllint

FROM scratch
COPY --from=builder /sqllint /sqllint
ENTRYPOINT ["/sqllint"]

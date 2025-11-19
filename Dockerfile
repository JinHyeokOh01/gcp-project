# 1단계: 빌드
FROM golang:1.24 AS builder

WORKDIR /app

# go.mod / go.sum 은 프로젝트 루트에 있다고 가정
COPY go.mod go.sum ./
RUN go mod download

# backend / frontend 코드 복사
COPY backend ./backend
COPY frontend ./frontend

# backend/main.go 를 빌드
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./backend

# 2단계: 런타임
FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/frontend ./frontend

ENV PORT=8080
EXPOSE 8080

CMD ["./server"]
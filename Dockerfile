# syntax=docker/dockerfile:1

# --- Stage 1: Backend ---
FROM golang:1.21-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY --exclude=frontend . .
RUN ls

RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o keenetic-go-vpn ./cmd/keenetic-go-vpn

# --- Stage 2: Frontend ---
FROM node:22-alpine AS frontend
WORKDIR /frontend

COPY frontend/package*.json ./
RUN npm install

COPY frontend/ .
RUN npm run build

# --- Stage 3: Final ---
FROM alpine:latest
WORKDIR /root/

COPY --from=backend /app/keenetic-go-vpn .
COPY --from=frontend /frontend/dist ./static

EXPOSE 8000
CMD ["./keenetic-go-vpn"]
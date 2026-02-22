# syntax=docker/dockerfile:1

# --- Stage 1: Backend ---
FROM golang:1.21-alpine AS backend
WORKDIR /app

# 1. Cache dependencies (This layer won't change if you edit frontend)
COPY go.mod go.sum ./
RUN go mod download

# 2. Copy source code (Includes main.go, keenetic/, etc.)
# We use COPY . . to support any future folders you create.
COPY --exclude=frontend . .

RUN go mod tidy

# 3. Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o keenetic-manager .

# --- Stage 2: Frontend ---
FROM node:22-alpine AS frontend
WORKDIR /frontend

# 1. Cache Node dependencies
COPY frontend/package*.json ./
RUN npm install

# 2. Copy Frontend source and Build
COPY frontend/ .
RUN npm run build

# --- Stage 3: Final ---
FROM alpine:latest
WORKDIR /root/

# Copy binary from backend stage
COPY --from=backend /app/keenetic-manager .
# Copy static assets from frontend stage
COPY --from=frontend /frontend/dist ./static

RUN chmod +x /root/keenetic-manager

EXPOSE 800
CMD ["./keenetic-manager"]
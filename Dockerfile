FROM node:22-bookworm-slim AS web-builder
WORKDIR /src/web

COPY web/package.json web/package-lock.json ./
RUN npm ci --no-fund --no-audit

COPY web/ ./
RUN npm run build

FROM golang:1.24-bookworm AS go-builder
WORKDIR /src
ARG BUILD_VERSION=dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=web-builder /src/web/dist ./web/dist

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags "-X main.buildVersion=${BUILD_VERSION}" -o /out/flux-board .

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app

COPY --from=go-builder /out/flux-board ./flux-board

EXPOSE 8080

ENTRYPOINT ["/app/flux-board"]

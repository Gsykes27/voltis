FROM golang:1.22-bookworm AS voltis-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY voltis ./voltis
RUN CGO_ENABLED=0 go build -o /out/voltis ./cmd/voltis

FROM node:22-bookworm AS app-builder
WORKDIR /app
COPY examples/basic/voltis.config.json ./voltis.config.json
COPY examples/basic/public ./public
COPY examples/basic/app ./app
WORKDIR /app/app
RUN npm install
COPY --from=voltis-builder /out/voltis /usr/local/bin/voltis
WORKDIR /app
RUN voltis build

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=voltis-builder /out/voltis /usr/local/bin/voltis
COPY --from=app-builder /app/dist ./dist
COPY --from=app-builder /app/public ./public
COPY --from=app-builder /app/voltis.config.json ./voltis.config.json
EXPOSE 3000
ENTRYPOINT ["/usr/local/bin/voltis", "start", "-addr", ":3000"]

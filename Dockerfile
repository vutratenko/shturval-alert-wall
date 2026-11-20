FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.24-alpine AS go
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
COPY config/samples /src/config/samples
COPY --from=web /src/cmd/server/static ./cmd/server/static
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=go /out/server /server
COPY --from=go /src/config/samples /config/samples
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/server"]
FROM golang:1.27.1-alpine AS build
WORKDIR /src
COPY go.* ./
RUN go mod download
COPY . .
ARG VERSION=development
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/server ./cmd/server

FROM scratch
LABEL org.opencontainers.image.source="https://github.com/vimalyad/devops-devsecops-demo"
COPY --from=build /out/server /server
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/server"]

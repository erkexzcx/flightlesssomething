FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ARG version
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GOARM=${TARGETVARIANT#v} go build -a -ldflags "-w -s -X main.version=$version -extldflags '-static'" -o fm ./cmd/flightlesssomething/main.go

FROM --platform=$BUILDPLATFORM gcr.io/distroless/static
COPY --from=builder /app/fm /fm
ENTRYPOINT ["/fm"]

VERSION 0.7
FROM golang:1.22-bookworm
WORKDIR /workspace

generate:
  FROM +tools
  COPY proto/ ./proto
  RUN mkdir -p gen
  RUN protoc -I=proto/ \
    --go_out=gen \
    --go_opt=paths=source_relative \
    --connect-go_out=gen \
    --connect-go_opt=paths=source_relative \
    proto/telemetry/v1alpha1/telemetry.proto
  SAVE ARTIFACT gen AS LOCAL gen

tidy:
  LOCALLY
  RUN go mod tidy
  RUN go fmt ./...

lint:
  FROM golangci/golangci-lint:v1.59.1
  WORKDIR /workspace
  COPY . .
  RUN golangci-lint run --timeout 5m ./...

test:
  COPY go.mod go.sum .
  RUN go mod download
  COPY . .
  RUN go test -coverprofile=coverage.out -v ./...
  SAVE ARTIFACT coverage.out AS LOCAL coverage.out

tools:
  RUN apt update
  RUN apt install -y protobuf-compiler
  RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  RUN go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest
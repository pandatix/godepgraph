# Build stage
FROM golang:1.24.5@sha256:ef5b4be1f94b36c90385abd9b6b4f201723ae28e71acacb76d00687333c17282 AS builder

WORKDIR /go/src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install github.com/bufbuild/buf/cmd/buf@v1.56.0 && \
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.27.1 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc && \
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.27.1 && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.7
RUN make buf

RUN apt update && apt install zip unzip -y
RUN make update-swagger

ENV CGO_ENABLED=0
RUN go build -cover -o /go/bin/godepgraph cmd/godepgraph/main.go



# Prod stage
FROM golang:1.24.5@sha256:ef5b4be1f94b36c90385abd9b6b4f201723ae28e71acacb76d00687333c17282
COPY --from=builder /go/bin/godepgraph /godepgraph/godepgraph
COPY --from=builder /go/src/gen /godepgraph/gen
ENTRYPOINT [ "/godepgraph/godepgraph" ]

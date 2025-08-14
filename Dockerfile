# Build stage
FROM golang:1.24.4@sha256:10c131810f80a4802c49cab0961bbe18a16f4bb2fb99ef16deaa23e4246fc817 AS builder

WORKDIR /go/src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install github.com/bufbuild/buf/cmd/buf@v1.48.0 && \
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.25.1 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc && \
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.25.1 && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.1
RUN make buf

RUN apt update && apt install zip unzip -y
RUN make update-swagger

ENV CGO_ENABLED=0
RUN go build -cover -o /go/bin/godepgraph cmd/godepgraph/main.go



# Prod stage
FROM golang:1.24.4@sha256:10c131810f80a4802c49cab0961bbe18a16f4bb2fb99ef16deaa23e4246fc817
COPY --from=builder /go/bin/godepgraph /godepgraph/godepgraph
COPY --from=builder /go/src/gen /godepgraph/gen
ENTRYPOINT [ "/godepgraph/godepgraph" ]

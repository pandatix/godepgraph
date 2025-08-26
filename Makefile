.PHONY: buf
buf:
	rm -rf gen
	buf format -w
	buf build
	buf generate

.PHONY: update-swagger
update-swagger:
	./hack/update-swagger.sh

TAG?=dev
.PHONY: docker
docker:
	docker build -t $(REGISTRY)pandatix/godepgraph:$(TAG) . && docker push $(REGISTRY)pandatix/godepgraph:$(TAG)

.PHONY: build
build:
	mkdir bin/ | true
	go build -o bin/godepgraph     cmd/godepgraph/main.go
	go build -o bin/godepgraph-cli cmd/godepgraph-cli/main.go

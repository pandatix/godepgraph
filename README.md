# GoDepGraph

GoDepGraph is a toolbox created to support "Reconstructing Systems-of-Systems architectures towards analyzing cascading attacks" research work.

It has 3 capabilities:
- `CDN` to construct the Call-graph Dependency Network of a Go codebase
- `RDG` to produce a Resource Deployment Graph by parsing Pulumi states
- `SIG` to build a System Interaction Graph out of OpenTelemetry traces

## Local setup

Each command is made to export data to Neo4J database. To run one locally, you can do the following.

```bash
docker run -p 7474:7474 -p 7687:7687 -e NEO4J_AUTH=none neo4j:5.22.0
```

docker run --rm --name jaeger \
  -p 16686:16686 \
  -p 4317:4317 \
  -p 4318:4318 \
  -p 5778:5778 \
  -p 9411:9411 \
  jaegertracing/jaeger:2.5.0

## PAC

https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/
https://kyverno.io/docs/introduction/admission-controllers/
https://medium.com/@platform.engineers/building-custom-admission-controllers-in-go-for-kubernetes-271168ec56b5

## SIG Builder

https://roosma.dev/p/first-opentelemetry-exporter/
https://opentelemetry.io/docs/collector/building/receiver/

## Alg4

## Vuln Looker

https://neo4j.com/docs/cypher-manual/current/patterns/shortest-paths/

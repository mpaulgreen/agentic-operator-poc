FROM --platform=$BUILDPLATFORM golang:1.22 AS builder
ARG TARGETARCH

WORKDIR /workspace
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/

RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

FROM registry.access.redhat.com/ubi9/ubi-micro:latest

LABEL name="elasticsearch-operator" \
      vendor="Example" \
      version="0.1.0" \
      summary="ElasticsearchCluster Operator" \
      description="Manages ElasticsearchCluster resources on OpenShift" \
      io.k8s.display-name="ElasticsearchCluster Operator" \
      io.k8s.description="Manages ElasticsearchCluster resources on OpenShift" \
      io.openshift.tags="operator,search,elasticsearch"

WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532
ENTRYPOINT ["/manager"]

# Build the manager binary
FROM golang:1.19 as builder

WORKDIR /workspace

# Copy the Go Modules manifests, plus the source
COPY . ./

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Build
ENV PKG=github.com/aws/aws-cloud-map-mcs-controller-for-k8s/pkg/version
RUN GIT_TAG=$(git describe --tags --dirty --always) && \
    GIT_COMMIT=$(git describe --dirty --always) && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build \
     -ldflags="-s -w -X ${PKG}.GitVersion=${GIT_TAG} -X ${PKG}.GitCommit=${GIT_COMMIT}" -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]

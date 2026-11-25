# Build the manager binary
FROM golang:1.19 as builder

# TARGETOS/TARGETARCH are set automatically by BuildKit (--platform); defaults keep the old linux/amd64 build
ARG TARGETOS=linux
ARG TARGETARCH=amd64

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
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build \
     -ldflags="-s -w -X ${PKG}.GitVersion=${GIT_TAG} -X ${PKG}.GitCommit=${GIT_COMMIT}" -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]

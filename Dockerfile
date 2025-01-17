# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM crmirror.lcpu.dev/gcr.io/distroless/static:nonroot
# FROM crmirror.lcpu.dev/docker.io/library/ubuntu:24.04
ARG TARGETOS
ARG TARGETARCH

WORKDIR /
COPY ./build/manager-${TARGETARCH} manager
USER 65532:65532

ENTRYPOINT ["/manager"]

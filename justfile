all: build

build-utility:
    go build -o ./build/utility ./cmd/utility

build-manager:
    CGO_ENABLED=0 GOARCH=amd64 go build -o ./build/manager-amd64 ./cmd/manager
    CGO_ENABLED=0 GOARCH=arm64 go build -o ./build/manager-arm64 ./cmd/manager

build-manager-amd64:
    go build -o ./build/manager-amd64 ./cmd/manager

build: build-utility build-manager

build-image: build-manager
    docker build . -t crmirror.lcpu.dev/xtlsoft/hpcgame-judger:v0.1.0 --platform=linux/amd64,arm64
    docker push crmirror.lcpu.dev/xtlsoft/hpcgame-judger:v0.1.0

deploy:
    kubectl apply -f ./manifests

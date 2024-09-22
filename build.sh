#!/bin/bash

set -e

DIR=$(realpath "$(dirname "${BASH_SOURCE[0]}")")

mkdir -p "$DIR/dist"

cd "$DIR/client"
npm ci
npm run build

cd "$DIR"
rm -rv "$DIR/dist/*" || true
(
    export GOOS=linux;
    export GOARCH=amd64;
    export CGO_ENABLED=0
    mkdir -p "$DIR/dist/linux-amd64";
    go build -trimpath -ldflags=-buildid= -o "$DIR/dist/linux-amd64/myfans" ./cmd/myfans;
    cp "$DIR/config.sample.yaml" "$DIR/dist/linux-amd64/config.yaml";
    cp "$DIR/auth.sample.yaml" "$DIR/dist/linux-amd64/auth.yaml";
    (cd "$DIR/dist" && tar -cvzf linux-amd64.tar.gz linux-amd64);
)
(
    export GOOS=linux;
    export GOARCH=arm64;
    export CGO_ENABLED=0
    mkdir -p "$DIR/dist/linux-arm64";
    go build -trimpath -ldflags=-buildid= -o "$DIR/dist/linux-arm64/myfans" ./cmd/myfans;
    cp "$DIR/config.sample.yaml" "$DIR/dist/linux-arm64/config.yaml";
    cp "$DIR/auth.sample.yaml" "$DIR/dist/linux-arm64/auth.yaml";
    (cd "$DIR/dist" && tar -cvzf linux-arm64.tar.gz linux-arm64);
)
(
    export GOOS=linux;
    export GOARCH=arm;
    export GOARM=7;
    export CGO_ENABLED=0
    mkdir -p "$DIR/dist/linux-armv7";
    go build -trimpath -ldflags=-buildid= -o "$DIR/dist/linux-armv7/myfans" ./cmd/myfans;
    cp "$DIR/config.sample.yaml" "$DIR/dist/linux-armv7/config.yaml";
    cp "$DIR/auth.sample.yaml" "$DIR/dist/linux-armv7/auth.yaml";
    (cd "$DIR/dist" && tar -cvzf linux-armv7.tar.gz linux-armv7);
)
(
    export GOOS=darwin;
    export GOARCH=arm64;
    export CGO_ENABLED=0
    mkdir -p "$DIR/dist/darwin-arm64";
    go build -trimpath -ldflags=-buildid= -o "$DIR/dist/darwin-arm64/myfans" ./cmd/myfans;
    cp "$DIR/config.sample.yaml" "$DIR/dist/darwin-arm64/config.yaml";
    cp "$DIR/auth.sample.yaml" "$DIR/dist/darwin-arm64/auth.yaml";
    (cd "$DIR/dist" && tar -cvzf darwin-arm64.tar.gz darwin-arm64);
)
(
    export GOOS=darwin;
    export GOARCH=amd64;
    export CGO_ENABLED=0
    mkdir -p "$DIR/dist/darwin-amd64";
    go build -trimpath -ldflags=-buildid= -o "$DIR/dist/darwin-amd64/myfans" ./cmd/myfans;
    cp "$DIR/config.sample.yaml" "$DIR/dist/darwin-amd64/config.yaml";
    cp "$DIR/auth.sample.yaml" "$DIR/dist/darwin-amd64/auth.yaml";
    (cd "$DIR/dist" && tar -cvzf darwin-amd64.tar.gz darwin-amd64);
)
(
    export GOOS=windows;
    export GOARCH=arm64;
    export CGO_ENABLED=0
    mkdir -p "$DIR/dist/windows-arm";
    go build -trimpath -ldflags=-buildid= -o "$DIR/dist/windows-arm/myfans.exe" ./cmd/myfans;
    cp "$DIR/config.sample.yaml" "$DIR/dist/windows-arm/config.yaml";
    cp "$DIR/auth.sample.yaml" "$DIR/dist/windows-arm/auth.yaml";
    (cd "$DIR/dist" && zip -r windows-arm.zip windows-arm);
)
(
    export GOOS=windows;
    export GOARCH=amd64;
    export CGO_ENABLED=0
    mkdir -p "$DIR/dist/windows-x64";
    go build -trimpath -ldflags=-buildid= -o "$DIR/dist/windows-x64/myfans.exe" ./cmd/myfans;
    cp "$DIR/config.sample.yaml" "$DIR/dist/windows-x64/config.yaml";
    cp "$DIR/auth.sample.yaml" "$DIR/dist/windows-x64/auth.yaml";
    (cd "$DIR/dist" && zip -r windows-x64.zip windows-x64);
)


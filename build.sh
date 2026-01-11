#!/bin/bash

# Настройки сборки
LDFLAGS="-s -w"
TAGS="nomsgpack"
SRC="./cmd/main.go"

echo "Starting compilation for all platforms..."

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -tags=$TAGS -ldflags="$LDFLAGS" -o dist/tunsgo-aarch64 $SRC
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath -tags=$TAGS -ldflags="$LDFLAGS" -o dist/tunsgo-armv7 $SRC
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=5 go build -trimpath -tags=$TAGS -ldflags="$LDFLAGS" -o dist/tunsgo-armv5 $SRC
CGO_ENABLED=0 GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -trimpath -tags=$TAGS -ldflags="$LDFLAGS" -o dist/tunsgo-mipsle $SRC
CGO_ENABLED=0 GOOS=linux GOARCH=mips GOMIPS=softfloat go build -trimpath -tags=$TAGS -ldflags="$LDFLAGS" -o dist/tunsgo-mips $SRC
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -tags=$TAGS -ldflags="$LDFLAGS" -o dist/tunsgo-x64 $SRC
CGO_ENABLED=0 GOOS=linux GOARCH=386 go build -trimpath -tags=$TAGS -ldflags="$LDFLAGS" -o dist/tunsgo-x86 $SRC

echo "Compilation finished. Check the dist folder."
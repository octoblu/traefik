#!/bin/bash

dependencies(){
  # glide up --quick
  echo "Skipping dependencies update"
}

generate(){
  go generate
}

copy() {
  mkdir -p dist
  cp traefik dist/
}

compile(){
  env \
    GOOS=linux \
    GOARCH=amd64 \
    CGO_ENABLED=0 \
    go build \
      -a \
      -ldflags '-s'
}

package() {
  docker build --tag local/traefik:entrypoint .
  docker tag -f local/traefik:entrypoint local/traefik:latest
  docker tag -f local/traefik:entrypoint quay.io/octoblu/traefik:v1.0.alpha.429
}

main(){
  export GO15VENDOREXPERIMENT=1

  dependencies \
    && generate \
    && compile \
    && copy \
    && package
}
main

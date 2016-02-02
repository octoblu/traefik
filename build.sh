#!/bin/bash

dependencies(){
  # glide up --quick
  echo "Skipping dependencies update"
}

generate(){
  go generate
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

copy() {
  cp traefik entrypoint/
}

package() {
  docker build --tag local/traefik:entrypoint entrypoint
  docker tag -f local/traefik:entrypoint local/traefik:latest
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

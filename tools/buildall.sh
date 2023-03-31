#!/bin/bash



for os in linux darwin; do 
    for arch in amd64 arm64; do
        osarch="$os-$arch"
        echo "Building $osarch"
        mkdir -p build/$osarch
        env GOOS=$os GOARCH=$arch go build -o build/$osarch/corediff
    done
done

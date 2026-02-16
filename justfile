test:
    go test ./...

mogo:
    #!/usr/bin/env bash
    tmp=$(mktemp)
    GOOS=linux GOARCH=amd64 go build -o "$tmp" ./cmd/corediff
    rsync "$tmp" mogo:/usr/local/bin/cd3
    rm "$tmp"

release:
    #!/usr/bin/env bash
    set -euo pipefail
    targets="darwin/arm64 linux/amd64 linux/arm64"
    remote="root@sansec-web:/data/downloads"
    for target in $targets; do
        os="${target%/*}"
        arch="${target#*/}"
        echo "Building $os/$arch ..."
        tmp=$(mktemp)
        GOOS=$os GOARCH=$arch go build -o "$tmp" ./cmd/corediff
        echo "Uploading to $remote/$os-$arch/corediff ..."
        rsync "$tmp" "$remote/$os-$arch/corediff"
        rm "$tmp"
    done
    echo "Done."

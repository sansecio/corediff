test:
    go test ./...

mogo:
    #!/usr/bin/env bash
    tmp=$(mktemp)
    GOOS=linux GOARCH=amd64 go build -o "$tmp" ./cmd/corediff
    rsync "$tmp" mogo:/usr/local/bin/cd3
    rm "$tmp"

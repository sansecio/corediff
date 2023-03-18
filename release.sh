#!/bin/bash

set -e

function chronic {
    set +e
    echo -n "   $*"
    ret=$($* 2>&1)
    # echo "Return code: $? and $!"
    if [ $? -ne 0 ]; then
        echo
        echo ">>>> Program $* failure:"
        echo ">>>> $ret"
        exit
    fi
    echo -n $'\r'
    printf " \e[1;32m✔\e[0m $\n*"
    set -e
}

targets="linux,amd64 linux,arm64 darwin,arm64 darwin,amd64" # linux,arm64
for x in $targets; do
    os=$(echo $x | cut -d, -f1)
    arch=$(echo $x | cut -d, -f2)

    fn="corediff-$os-$arch"
    echo Building $fn
    chronic env GOOS=$os GOARCH=$arch go build -o build/$fn &&
    chronic rsync  build/$fn ssweb:/data/downloads/$os-$arch/corediff
done


>corediff.bin
chronic corediff -d corediff.bin -m \
    db/m1ce.db \
    db/m1ee.db \
    db/m2ce*.db \
    db/m2ee*.db

chronic rsync corediff.bin ssweb:/data/ecomscan/downloads/corediff.bin

echo
echo 'Finished! Run:'
echo
echo "  curl https://sansec.io/downloads/$(uname -sm | tr 'LD ' 'ld-')/corediff -O && chmod 755 corediff"
echo

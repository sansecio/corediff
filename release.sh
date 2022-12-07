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

# if different arch
chronic go build -o ~/bin/corediff

(
	GOARC=amd64
	GOOS=linux
	chronic go build -o /tmp/corediff
	chronic rsync /tmp/corediff ssweb:/data/ecomscan/downloads
)
#chronic upx -qq ~/bin/corediff

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
echo '  curl https://api.sansec.io/downloads/corediff -O && chmod 755 corediff'
echo

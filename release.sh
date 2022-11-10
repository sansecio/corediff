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
    echo -e " \e[1;32m✔\e[0m $*"
    set -e
}

chronic go build -o ~/bin/corediff
chronic upx -qq ~/bin/corediff
chronic rsync ~/bin/corediff ssweb:/data/ecomscan/downloads

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

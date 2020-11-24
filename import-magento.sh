#/bin/bash

set -e

function import {

    echo "Importing $1 .."
    src=./$1

    if [[ ! -d $src ]]; then
        git clone --quiet $2 $src
    else
        ( cd $src && git clean -f -d && git fetch --quiet --tags )
    fi

    tags=$(cd $src && git tag -l)
    for v in $tags; do
        echo $PWD $v
        ( cd $src && git clean -f -d && git checkout --quiet $v )
        ./corediff.py add $src
    done
}

import magento1 https://github.com/OpenMage/magento-mirror.git
import magento2 https://github.com/magento/magento2.git
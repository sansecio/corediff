#/bin/bash

set -e

function import {

    name="$1"
    url="$2"
    echo "Importing $name .."
    src=./$name

    if [[ ! -d $src ]]; then
        git clone --quiet $url $src
    else
        ( cd $src && git clean -f -d && git fetch --quiet --tags )
    fi

    tags=$(cd $src && git tag -l)
    for v in $tags; do
        echo $PWD $v
        ( cd $src && git clean -f -d && git checkout --quiet $v )
        # ensure that it looks like a proper magento root
        # touch $src/wp-config.php
        corediff --database=$name.db --add $src
    done
}


#import magento1 https://github.com/OpenMage/magento-mirror.git
import magento2 https://github.com/magento/magento2.git
#/bin/bash

set -e
#set -x 

export COMPOSER_HOME=$PWD/.composer
export COMPOSER_IGNORE_PLATFORM_REQS=1
mkdir -p $COMPOSER_HOME

function import {

    name="$1"
    filter="$2"
    url="$3"

    echo "Importing $name .."
    src=./git/$name

    if [[ ! -d $src ]]; then
        git clone --quiet $url $src
    else
        ( cd $src && git clean -f -d && git fetch --quiet --tags )
    fi
    tags=$(cd $src && git tag -l | sort -rn | egrep -v $filter)
    for v in $tags; do
        dbname="db/$name-git-$v.db"
        if test -e $dbname; then
            echo "Skipping $v"
            continue
        fi
        echo Indexing $v ...
        ( 
            echo "- Resetting git ..."
            cd $src && 
            git clean -qfdx && 
            git reset --hard --quiet && 
            git checkout --quiet $v 

            #composer install --dry-run --no-plugins --ignore-platform-reqs |& grep ' - Installing'  | awk '{print $3}' >> ../composer.$name
            if test -e composer.json; then
                echo "- Installing composer dependencies ..."
                chronic composer install --no-plugins --ignore-platform-reqs --no-interaction
            fi
            #composer show --no-interaction --no-plugins | awk '{print $1}' >> ../composer.$name
        )
        # ensure that it looks like a proper magento root
        # touch $src/wp-config.php

        echo "- Indexing with corediff ..."
        chronic corediff --no-cms --database=$dbname --add $src
    done
}


#import magento1 https://github.com/OpenMage/magento-lts.git
import magento2 '^(2\.3\.3$|2\.0\.9$|0\.|1\.)' https://github.com/magento/magento2.git

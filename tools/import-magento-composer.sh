#!/bin/bash

# Requires Magento access keys in $PWD/.composer/auth.json
# set -x 


export COMPOSER_HOME=$PWD/.composer

mkdir -p db

for edition in enterprise community; do
    echo "Finding all available versions for $edition ...."
    for ver in $(composer show --no-interaction --no-plugins magento/project-$edition-edition -a --format=json 2>/dev/null | jq -r '.versions[]' 2>/dev/null); do
        echo "magento/product-$edition-edition $ver"
        dbname="db/magento-$edition-$ver.db"
        if test -e $dbname; then
            continue
        fi
        (
            rm -rf tmp
            mkdir -p tmp
            cd tmp
            echo "- Installing magento/product-$edition-edition $ver"
            chronic composer create-project --no-interaction --no-plugins --ignore-platform-reqs magento/project-$edition-edition:$ver .
        )
        echo "- Indexing with corediff ..."
        chronic corediff --no-cms --database=$dbname --add tmp
    done
    rm -rf tmp

done


corediff -d magento-composer.db -m db/*.db

# composer config allow-plugins true
#
# packagist api:
# curl https://repo.packagist.org/p2/[vendor]/[package].json
# curl https://repo.packagist.org/p2/symfony/mime.json
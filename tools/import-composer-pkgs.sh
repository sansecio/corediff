#!/bin/bash

# set -x 

export COMPOSER_HOME=$PWD/.composer
export COMPOSER_AUTH="{\"http-basic\": {\"repo.magento.com\": {\"username\": \"87d8f716552861ad1e17627f08cfef6c\", \"password\": \"82ae49fd65a2a8ff376ec51f55cde4bb\"}}}"

mkdir -p db

cat composer.magento2 | while read pkg; do
    echo $pkg
    mkdir -p tmp
    for ver in $(composer show --no-interaction --no-plugins $pkg -a --format=json 2>/dev/null | jq -r '.versions[]' 2>/dev/null); do
        echo $pkg $ver
        dbname="db/$(echo "$pkg $ver" | sed -r 's#[^a-zA-Z0-9]#_#g')"
        if test -e $dbname; then
            # echo "Skipping $dbname"
            continue
        fi
        (
            cd tmp
            rm -f composer.lock
            chronic composer require --no-interaction --no-plugins --ignore-platform-reqs "$pkg" "$ver" 
            # sleep 5
        )


        chronic corediff --no-cms --database=$dbname --add tmp
        # composer require --no-interaction --no-plugins --ignore-platform-reqs $pkg:$ver
        # corediff --database=magento2.db --add $pkg
    done
    rm -rf tmp
    # corediff --database=magento2.db --add $pkg
done



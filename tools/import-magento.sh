#/bin/bash

set -e
#set -x 

export COMPOSER_HOME=$PWD/.composer

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
    truncate -s0 composer.all
    tags=$(cd $src && git tag -l | sort -rn)
    for v in $tags; do
        echo $PWD $v
        ( 
            cd $src && 
            git clean -qfdx && 
            git reset --hard --quiet && 
            git checkout --quiet $v 

            #composer install --dry-run --no-plugins --ignore-platform-reqs |& grep ' - Installing'  | awk '{print $3}' >> ../composer.$name
            if test -e composer.json; then
                chronic composer install --no-plugins --ignore-platform-reqs --no-interaction
            fi
            #composer show --no-interaction --no-plugins | awk '{print $1}' >> ../composer.$name
        )
        # ensure that it looks like a proper magento root
        # touch $src/wp-config.php
        chronic corediff --no-cms --database=$name.db --add $src
    done

    cat composer.$name | sort | uniq > composer.uniq
    mv composer.uniq composer.$name

}


#import magento1 https://github.com/OpenMage/magento-lts.git
import magento2 https://github.com/magento/magento2.git

exit

# git reset --hard && git clean -f -d -x
# composer install --ignore-platform-req=php --no-interaction

cat composer.all | while read pkg; do
    for ver in $(composer show -q --no-interaction --no-plugins $pkg -a --format=json | jq -r '.versions[]' 2>/dev/null); do
        echo $pkg $ver
        # composer require --no-interaction --no-plugins --ignore-platform-reqs $pkg:$ver
        # corediff --database=magento2.db --add $pkg
    done
    # corediff --database=magento2.db --add $pkg
done

# 
composer create-project --repository-url=https://repo.magento.com/ magento/project-community-edition:2.4.5

composer create-project --repository-url=https://repo.magento.com/ 
magento/project-enterprise-edition:2.4.5
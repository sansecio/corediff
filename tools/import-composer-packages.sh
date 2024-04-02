#!/bin/bash

# set -x
maxpkgs=150


if [ -e /dev/shm ]; then
    tempdir=$(mktemp -d -p /dev/shm)
else   
    tempdir=$(mktemp -d)
fi
echo "Using tempdir $tempdir"


cleanup() {
  echo "Cleaning up..."
  rm -rf "$tempdir"
  echo "Cleanup done."
  exit
}
trap cleanup EXIT INT TERM

export COMPOSER_HOME=$PWD/.composer

mkdir -p db
cat composer.base composer.pkgs | while read pkg; do
    echo $pkg
    for ver in $(composer show --no-interaction --no-plugins $pkg -a --format=json 2>/dev/null | jq -r '.versions[]' 2>/dev/null | head -n $maxpkgs); do
        rm -rf $tempdir
        mkdir -p $tempdir
        flag="processed/$pkg/$(echo "$pkg $ver" | sed -r 's#\W#_#g')"
        mkdir -p $(dirname $flag)
        dbname="db/$(echo "$pkg" | sed -r 's#\W#_#g').db"
        echo "- $flag"
        if test -e $flag; then
            # echo "Skipping $flag"
            continue
        fi
        (
            cd $tempdir
            if [[ "$pkg" == "magento/project"* ]]; then
                composer create-project --no-interaction --ignore-platform-reqs $pkg:$ver .
            else
                composer require --no-interaction --no-plugins --update-no-dev --no-audit --no-progress --ignore-platform-reqs "$pkg" "$ver"
            fi
            # sleep 5
        )>$flag 2>&1 || {
            echo "Failed to install $pkg $ver"
            continue
        }
        chronic corediff --no-cms --database=$dbname --add $tempdir && touch "$flag"
    done
done

corediff -d m2.db -m db/*.db
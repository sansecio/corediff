#!/bin/bash

set -e

tools/buildall.sh

(
    cd build
    for x in */corediff; do
        echo "Uploading $x"
        rsync -a $x sansec-web:/data/downloads/$x;
    done
)

echo 'Finished! Run:'
echo
echo "  curl https://sansec.io/downloads/$(uname -sm | tr 'LD ' 'ld-')/corediff -O && chmod 755 corediff"
echo

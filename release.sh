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
chronic rsync ~/bin/corediff mogo:/data/www/ecomscan

echo
echo 'Finished! Run:'
echo
echo '	curl https://mageintel.com/ecomscan/corediff -O && chmod 755 corediff'
echo

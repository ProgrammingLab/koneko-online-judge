#!/bin/sh

cd /tmp
rm -rf ./koj-workspace

for var in `ls -1 ./input/*`
do
    echo ${var}
    sudo -u nobody chmod 777 -R /tmp >/dev/null 2>/dev/null
    sudo -u nobody rm -rf /tmp/* >/dev/null 2>/dev/null
    sudo -u nobody chmod 777 -R /var/tmp >/dev/null 2>/dev/null
    sudo -u nobody rm -rf /var/tmp/* >/dev/null 2>/dev/null
    sudo -u nobody chmod 777 -R /run/lock >/dev/null 2>/dev/null
    sudo -u nobody rm -rf /run/lock/* >/dev/null 2>/dev/null

    cp ${var} input
    chmod 744 input

    echo -n $1`basename ${var}`$1
    /usr/bin/time -f "$1%e %M" -- timeout $2 /usr/bin/sudo -u nobody -- /bin/bash -c "$3 <input.txt; echo -n $1$?"
    rm -rf input
done

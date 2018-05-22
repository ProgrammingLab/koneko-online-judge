#!/bin/sh

cd /tmp >/dev/null 2>/dev/null
mv ./koj-workspace/$4 . >/dev/null 2>/dev/null
chmod 755 $4 >/dev/null 2>/dev/null
rm -rf ./koj-workspace
chmod 700 $0 >/dev/null 2>/dev/null
chmod 700 -R ./input >/dev/null 2>/dev/null

for var in `ls -1 ./input/*.txt`
do
    sudo -u nobody chmod 777 -R /tmp >/dev/null 2>/dev/null
    sudo -u nobody rm -rf /tmp/* >/dev/null 2>/dev/null
    sudo -u nobody chmod 777 -R /var/tmp >/dev/null 2>/dev/null
    sudo -u nobody rm -rf /var/tmp/* >/dev/null 2>/dev/null
    sudo -u nobody chmod 777 -R /run/lock >/dev/null 2>/dev/null
    sudo -u nobody rm -rf /run/lock/* >/dev/null 2>/dev/null

    cp ${var} input.txt
    chmod 744 input.txt

    echo -n $1`basename ${var}`$1
    /usr/bin/time -f "$1%e %M" -- timeout $2 /usr/bin/sudo -u nobody -- /bin/bash -c "$3 <input.txt; echo -n $1$?"
    rm -rf input.txt
done

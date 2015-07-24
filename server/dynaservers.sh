#!/bin/sh

mv addrList $GOPATH/src/github.com/relab/smartMerge/server/
cd $GOPATH/src/github.com/relab/smartMerge/server/ 
while read p; do
	IFS=':' read -ra ADDR <<< "$p"
	if [ "${ADDR[0]}" != "127.0.0.1" ];
	then 
		echo host: ${ADDR[0]}
		echo Obs ssh not correctly implemented.
		echo Obs this script does not ensure remote as correct version.logout
		ssh ljehl@${ADDR[0]}.ux.uis.no 'mygo/src/github.com/relab/smartMerge/server/server -port ${ADDR[1]} &'
	else
		echo port: ${ADDR[1]}
		./server -gcoff -port ${ADDR[1]} -alg dyna&
	fi
done < addrList
cd -

echo "Running. Press enter to stop."

read && killall server

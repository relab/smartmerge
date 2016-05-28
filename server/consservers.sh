#!/bin/sh

mv addrList $GOPATH/src/github.com/relab/smartMerge/server/
cd $GOPATH/src/github.com/relab/smartMerge/server/ 
while read p; do
	IFS=':' read -ra ADDR <<< "$p"
	if [ "${ADDR[0]}" != "127.0.0.1" ];
	then 
		echo host: ${ADDR[0]}
		echo Obs ssh not implemented.
	else
		echo port: ${ADDR[1]}
		./server -gcoff -all-cores -port ${ADDR[1]} -alg cons&
	fi
done < addrList
cd -

echo "Running. Press enter to stop."

read && killall server

#!/bin/sh

mv addrList $GOPATH/src/github.com/relab/smartMerge/server/
cd $GOPATH/src/github.com/relab/smartMerge/server/ 
while read p; do
	IFS=':' read -ra ADDR <<< "$p"
	if [ "${ADDR[0]}" != "" ] 
	then 
		echo host: ${ADDR[0]}
		echo implement ssh to host: ${ADDR[0]}
	fi
	echo port: ${ADDR[1]}
	./server -port ${ADDR[1]} &
done < addrList
cd -

echo "Running. Press enter to stop."

read && killall server

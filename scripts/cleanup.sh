#!/bin/sh
export GST=$GOPATH/src/github.com/relab/gorums-stress-test

echo stopping readers

for Pi in 22 27 31 32 34
do
ssh pitter"$Pi" "cd $SM/client && killall client"
ssh pitter"$Pi" "rm /local/scratch/ljehl/*log*"
done

echo stopping servers
for Pi in 9 10 11 12 13 14 15 17 18 19
do
	echo stop pitter$Pi
if [ "$1" = "" ]; then
	ssh pitter"$Pi" "cd $SM/server && killall server && rm /local/scratch/ljehl/*log*"
else 
	ssh pitter"$Pi" "cd $SM/lserver && killall lserver && rm /local/scratch/ljehl/*log*"
fi
done

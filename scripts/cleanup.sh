#!/bin/sh
export GST=$GOPATH/src/github.com/relab/gorums-stress-test

echo stopping readers

for Pi in 13 14 15 17 18 19
do
ssh pitter"$Pi" "cd $GST/client && killall client"
ssh pitter"$Pi" "rm /local/scratch/ljehl/*log*"
done

echo stopping servers
for Pi in 9 10 11 12 13 14 15 17 18 19
do
	ssh pitter"$Pi" "cd $GST/server && killall server && rm /local/scratch/ljehl/*log*"
done

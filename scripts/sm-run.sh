#!/bin/sh

echo starting servers.
ssh pitter24 "nohup $HOME/mygo/src/github.com/relab/smartMerge/server/servers.sh > /dev/null 2>&1 &"
ssh pitter25 "nohup $HOME/mygo/src/github.com/relab/smartMerge/server/servers.sh > /dev/null 2>&1 &"
ssh pitter26 "nohup $HOME/mygo/src/github.com/relab/smartMerge/server/servers.sh > /dev/null 2>&1 &"

export SM=$HOME/mygo/src/github.com/relab/smartMerge

sleep 1



echo starting Writers
ssh pitter21 "nohup client -conf $SM/client/addrList -alg=sm -mode=bench -contW -size=4000 -nclients=5 -id=5 -initsize=12 -gc-off -all-cores > /local/scratch/ljehl/writerslog 2>&1 &"

sleep 3

echo starting Reconfigurers
client -conf client/addrList -alg=sm -mode=exp -rm -nclients="$*" -initsize=12 -gc-off -elog -all-cores > /local/scratch/ljehl/reconflog 2>&1

sleep 2
echo stopping Writers
ssh pitter21 "cd $SM && killall client/client"
ssh pitter21 "mv /local/scratch/ljehl/*.elog $SM/"
mv /local/scratch/ljehl/*.elog $SM/

ssh pitter24 "cd $SM/server && killall server" 
ssh pitter25 "cd $SM/server && killall server" 
ssh pitter26 "cd $SM/server && killall server" 


#!/bin/sh

echo starting servers.
#ssh pitter30 "nohup $HOME/mygo/src/github.com/relab/smartMerge/server/servers.sh > /dev/null 2>&1 &"
#ssh pitter31 "nohup $HOME/mygo/src/github.com/relab/smartMerge/server/servers.sh > /dev/null 2>&1 &"
#ssh pitter32 "nohup $HOME/mygo/src/github.com/relab/smartMerge/server/servers.sh > /dev/null 2>&1 &"
ssh pitter30 "nohup $HOME/mygo/src/github.com/relab/smartMerge/server/servers.sh > $SM/pi30servlog 2>&1 &"
ssh pitter31 "nohup $HOME/mygo/src/github.com/relab/smartMerge/server/servers.sh > $SM/pi31servlog 2>&1 &"
ssh pitter32 "nohup $HOME/mygo/src/github.com/relab/smartMerge/server/servers.sh > $SM/pi32servlog 2>&1 &"

export SM=$HOME/mygo/src/github.com/relab/smartMerge

sleep 1



echo starting Writers
#ssh pitter21 "$SM/scripts/wclients sm"
ssh pitter28 "nohup client -conf $SM/client/addrList -alg=sm -mode=bench -contW -size=4000 -nclients=5 -id=5 -initsize=12 -gc-off -all-cores > /local/scratch/ljehl/writerslog 2>&1 &"

sleep 3


echo starting Reconfigurers
if ! [ "$*" == "" ]; then
client -conf $SM/client/addrList -alg=sm -mode=exp -rm -nclients="$*" -initsize=12 -gc-off -elog -all-cores > /local/scratch/ljehl/reconflog 2>&1
else
	sleep 20
fi

sleep 1
echo stopping Writers
ssh pitter28 "killall client"
ssh pitter28 "mv /local/scratch/ljehl/*.elog $SM/"
ssh pitter28 "mv /local/scratch/ljehl/writerslog* $SM/"
mv /local/scratch/ljehl/*.elog $SM/
mv /local/scratch/ljehl/reconflog $SM/

ssh pitter30 "cd $SM/server && killall server" 
ssh pitter31 "cd $SM/server && killall server" 
ssh pitter32 "cd $SM/server && killall server" 


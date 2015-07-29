#!/bin/sh

echo starting servers.
ssh pitter24 "$HOME/mygo/src/github.com/relab/smartMerge/server/dynaservers.sh" &
ssh pitter25 "$HOME/mygo/src/github.com/relab/smartMerge/server/dynaservers.sh" &
ssh pitter26 "$HOME/mygo/src/github.com/relab/smartMerge/server/dynaservers.sh" &

export SM=$HOME/mygo/src/github.com/relab/smartMerge

sleep 1

cd $SM

echo starting Writers
(client/client -conf client/addrList -alg=dyna -mode=bench -contW -size=4000 -nclients=5 -id=5 -initsize=12 -gc-off -all-cores > logfile &)

echo starting Reconfigurers
client/client -conf client/addrList -alg=dyna -mode=exp -rm -nclients="$*" -initsize=12 -gc-off -elog -all-cores

sleep 1
echo stopping Writers
killall client/client 
 
ssh pitter24 "pkill -u ljehl"
ssh pitter25 "pkill -u ljehl"
ssh pitter26 "pkill -u ljehl"


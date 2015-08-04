#!/bin/sh
 
echo starting servers.
ssh pitter24 "nohup $HOME/mygo/src/github.com/relab/smartMerge/server/dynaservers.sh > /dev/null 2>&1 &"
ssh pitter25 "nohup $HOME/mygo/src/github.com/relab/smartMerge/server/dynaservers.sh > /dev/null 2>&1 &"
ssh pitter26 "nohup $HOME/mygo/src/github.com/relab/smartMerge/server/dynaservers.sh > /dev/null 2>&1 &"

export SM=$HOME/mygo/src/github.com/relab/smartMerge

sleep 1

cd $SM

echo starting Writers
ssh pitter21 "nohup $SM/client/client -conf $SM/client/addrList -alg=dyna -mode=bench -contW -size=4000 -nclients=5 -id=5 -initsize=12 -gc-off -all-cores > logfile 2>&1 &"

echo starting Reconfigurers
client/client -conf client/addrList -alg=dyna -mode=exp -rm -nclients="$*" -initsize=12 -gc-off -elog -all-cores

sleep 1
echo stopping Writers
ssh pitter21 "cd $SM && killall client/client"
#scp pitter21:$SM/*.elog . 
#ssh pitter21 "cd $SM && rm *.elog"
mv ~/*.elog $SM/
mv ~/logfile $SM/

ssh pitter24 "cd $SM/server && killall server" 
ssh pitter25 "cd $SM/server && killall server" 
ssh pitter26 "cd $SM/server && killall server" 


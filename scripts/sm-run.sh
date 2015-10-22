#!/bin/sh

export SM=$HOME/mygo/src/github.com/relab/smartMerge

echo starting servers.
for Pi in 9 10 11 12 13 14 15 17 18 19
do
	ssh pitter"$Pi" "nohup $SM/server/server -gcoff -all-cores -port 13000 > $SM/pi'$Pi'servlog 2>&1 &"
done

#ssh pitter24 "nohup $HOME/mygo/src/github.com/relab/smartMerge/server/servers.sh > /dev/null 2>&1 &"
#ssh pitter25 "nohup $HOME/mygo/src/github.com/relab/smartMerge/server/servers.sh > /dev/null 2>&1 &"
#ssh pitter26 "nohup $HOME/mygo/src/github.com/relab/smartMerge/server/servers.sh > /dev/null 2>&1 &"

sleep 1



echo starting Writers
#ssh pitter21 "$SM/scripts/wclients sm"
for Pi in {30..34}
do
ssh pitter"$Pi" "nohup client -conf $SM/scripts/newList -alg=sm -mode=bench -contW -size=4000 -nclients=1 -id=5 -initsize=10 -gc-off -all-cores > /local/scratch/ljehl/pi'$Pi'writerslog1 2>&1 &"

#ssh pitter"$Pi" "nohup client -conf $SM/scripts/newList -alg=sm -mode=bench -contW -size=4000 -nclients=1 -id=5 -initsize=7 -gc-off -all-cores > /local/scratch/ljehl/pi'$Pi'writerslog2 2>&1 &"
done

sleep 3


if ! [ "$*" == "" ]; then
	echo starting Reconfigurers
	client -conf $SM/scripts/newList -alg=sm -mode=exp -rm -nclients="$*" -initsize=10 -gc-off -elog -all-cores > /local/scratch/ljehl/reconflog 2>&1
else
	echo no reconfiguration, waiting 10 seconds
	sleep 10
fi

sleep 1
echo stopping Writers

for Pi in {30..34}
do
ssh pitter"$Pi" "killall client"
ssh pitter"$Pi" "mv /local/scratch/ljehl/*.elog $SM/"
ssh pitter"$Pi" "mv /local/scratch/ljehl/*writerslog* $SM/"
done
mv /local/scratch/ljehl/*.elog $SM/
mv /local/scratch/ljehl/reconflog $SM/

for Pi in 9 10 11 12 13 14 15 17 18 19
do
	ssh pitter"$Pi" "cd $SM/server && killall server" 
done

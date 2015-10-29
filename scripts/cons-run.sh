#!/bin/sh

export SM=$GOPATH/src/github.com/relab/smartMerge

SERVS=(9 10 11 12 13 14 15 17)
READS=(27 28 30 31 32)

cd $SM
mkdir exlogs || {
	echo "press enter to continue"
	read
}

echo starting servers.
for Pi in ${SERVS[@]}
do
	echo starting server on pitter$Pi
	ssh pitter"$Pi" "nohup $SM/server/server -all-cores -alg=cons -port 13000 -v=6 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/servlogpi'$Pi' 2>&1 &"
done


sleep 1


echo single write
$SM/client/client -conf $SM/scripts/newList -alg=cons -mode=bench -writes=1 -size=4000 -nclients=1 -id=5 -initsize=100 

echo starting Readers
for Pi in ${READS[@]}
do
	echo starting reader on pitter$Pi
ssh pitter"$Pi" "nohup $SM/client/client -conf $SM/scripts/newList -alg=cons -mode=bench -contR -nclients=1 -id='$Pi' -initsize=100 -all-cores -log_events -v=4 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/rlogpi'$Pi' 2>&1 &"
done

sleep 3


if ! [ "$*" == "" ]; then
	echo starting Reconfigurers
	$SM/client/client -conf $SM/scripts/newList -alg=cons -mode=exp -rm -nclients="$*" -initsize=100 -gc-off -elog -all-cores -v=6 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/reconflog 2>&1
else
	echo no reconfiguration, waiting 10 seconds
	sleep 10
fi

sleep 1
echo stopping Readers

for Pi in ${READS[@]}
do
	echo stopping reader on pitter$Pi
	ssh pitter"$Pi" "cd $SM/client && killall client" 
done

echo copy reader logs
for Pi in ${READS[@]}
do	
ssh pitter"$Pi" "mv /local/scratch/ljehl/*.elog $SM/exlogs"
ssh pitter"$Pi" "mv /local/scratch/ljehl/*log* $SM/exlogs"
done
mv /local/scratch/ljehl/*log* $SM/exlogs

for Pi in ${SERVS[@]}
do
	ssh pitter"$Pi" "cd $SM/server && killall server" 
	ssh pitter"$Pi" "mv /local/scratch/ljehl/*log* $SM/exlogs"
done

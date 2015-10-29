#!/bin/sh

export SM=$GOPATH/src/github.com/relab/smartMerge

SERVS=(9 10 11 12 13 14 15 17)

i=0
while read R; do
	READS[i]=$R
	i=$(($i+1))
	echo $i
done <$SM/scripts/readersList

#READS=(25 26 30 31 32)

cd $SM
mkdir exlogs || {
	echo "press enter to continue or Ctrl-C to abort"
	read
}

echo starting servers on
for Pi in ${SERVS[@]}
do
	echo -n pitter$Pi
	ssh pitter"$Pi" "nohup $SM/server/server -all-cores -port 13000 -v=6  -log_dir='/local/scratch/ljehl' > /dev/null 2>&1 &"
done

echo " "

sleep 1


echo single write
$SM/client/client -conf $SM/scripts/newList -alg=sm -mode=bench -writes=1 -size=4000 -nclients=1 -id=5 -initsize=100 

echo starting Readers on
for Pi in ${READS[@]}
do
	echo -n pitter$Pi
ssh pitter"$Pi" "nohup $SM/client/client -conf $SM/scripts/newList -alg=sm -mode=bench -contR -nclients=1 -id='$Pi' -initsize=100 -all-cores -log_events -v=5 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/rlogpi'$Pi' 2>&1 &"
done

echo " "

sleep 3


if ! [ "$*" == "" ]; then
	echo starting Reconfigurers
	$SM/client/client -conf $SM/scripts/newList -alg=sm -mode=exp -rm -nclients="$*" -initsize=100 -gc-off -elog -all-cores -v=6 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/reconflog 2>&1
else
	echo no reconfiguration, waiting 10 seconds
	sleep 10
fi

sleep 1
echo stopping Readers

for Pi in ${READS[@]}
do
	echo -n pitter$Pi
	ssh pitter"$Pi" "cd $SM/client && killall client" 
done

echo copy reader logs
for Pi in ${READS[@]}
do	
ssh pitter"$Pi" "mv /local/scratch/ljehl/*.elog $SM/exlogs"
ssh pitter"$Pi" "mv /local/scratch/ljehl/*log* $SM/exlogs"
done
mv /local/scratch/ljehl/*log* $SM/exlogs

echo stopping servers
for Pi in ${SERVS[@]}
do
	ssh pitter"$Pi" "cd $SM/server && killall server" 
	ssh pitter"$Pi" "mv /local/scratch/ljehl/*log* $SM/exlogs"
done

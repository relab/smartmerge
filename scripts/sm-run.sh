#!/bin/sh

#Arguements: $1: reader optimization $2:alg  $3: number of removals, $4: more reader option, e.g. -regular, 


export SM=$GOPATH/src/github.com/relab/smartMerge

SERVS=(9 10 11 12)

i=0
while read R; do
	READS[i]=$R
	i=$(($i+1))
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

if [ "$2" == "sm" -o "$1" == "norecontact" ]; then
	echo -n "sm-pitter$Pi "
	ssh pitter"$Pi" "nohup $SM/server/server -port 13000 -v=6  -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/pi'$Pi'servlog 2>&1 &"
	ssh pitter"$Pi" "nohup $SM/server/server -port 12000 -v=6  -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/pi'$Pi'2servlog 2>&1 &"
else
	echo -n "c-pitter$Pi "
	ssh pitter"$Pi" "nohup $SM/server/server -alg=cons -port 13000 -v=6  -log_dir='/local/scratch/ljehl' > /dev/null 2>&1 &"
	ssh pitter"$Pi" "nohup $SM/server/server -alg=cons -port 12000 -v=6  -log_dir='/local/scratch/ljehl' > /dev/null 2>&1 &"
fi
done

echo " "

sleep 1


echo single write
$SM/client/client -conf $SM/scripts/newList -alg=$2 -mode=bench -writes=1 -size=4000 -nclients=1 -id=5 -initsize=100 

echo starting Readers on
for Pi in ${READS[@]}
do
	echo -n "pitter$Pi "
ssh pitter"$Pi" "nohup $SM/client/client -conf $SM/scripts/newList -alg=$2 -opt=$1 $4 -mode=bench -contR -nclients=1 -id='$Pi' -initsize=100 -log_events -v=6 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/rlogpi'$Pi' 2>&1 &"

#ssh pitter"$Pi" "nohup $SM/client/client -conf $SM/scripts/newList -alg=sm -opt=$1 $3 -mode=bench -contR -gc-off -nclients=1 -id='1$Pi' -initsize=100 -log_events -v=6 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/rlogpi1'$Pi' 2>&1 &"
done

echo " "

sleep 1


if ! [ "$3" == "" ]; then
	echo starting Reconfigurers
	$SM/client/client -conf $SM/scripts/newList -alg=$2 -mode=exp -rm -nclients="$3" -initsize=100 -elog -v=6 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/reconflog 2>&1
else
	echo no reconfiguration, waiting 10 seconds
	sleep 10
fi

sleep 1
echo stopping Readers

for Pi in ${READS[@]}
do
	echo -n "pitter$Pi "
	ssh pitter"$Pi" "cd $SM/client && killall client" 
done

echo " "

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

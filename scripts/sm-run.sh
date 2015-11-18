#!/bin/sh

#Arguements: $1: reader optimization $2:alg  $3: number of removals, $4: more reader option, e.g. -regular, 

if [ "$1" == "help" ]; then
	echo Arguments:
	echo "1 reader optimization: no | doreconf"
	echo "2 alg: sm | cons"
	echo "3 cprov: normal | thrifty | norecontact"
	echo "4 reconfiguration: -rm -add -cont"
	echo "5 number of reconfiguration clients"
	echo "6 more reader options, e.g. -regular | -logThroughput"
	exit
fi

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

if [ "$3" == "norecontact" ]; then

	echo -n "sm-pitter$Pi "
	ssh pitter"$Pi" "nohup $SM/server/server -port 13000 -no-abort -v=6  -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/pi'$Pi'servlog 2>&1 &"
	ssh pitter"$Pi" "nohup $SM/server/server -port 12000 -no-abort -v=6  -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/pi'$Pi'2servlog 2>&1 &"


elif [ "$2" == "sm" ]; then

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
$SM/client/client -conf $SM/scripts/newList -alg=$2 -mode=bench -writes=1 -size=1000 -nclients=1 -id=5 -initsize=100 

echo starting Readers on
for Pi in ${READS[@]}
do
	echo -n "pitter$Pi "
ssh pitter"$Pi" "nohup $SM/client/client -conf $SM/scripts/newList -alg=$2 -opt=$1 -cprov=$3 $6 -mode=bench -contR -nclients=1 -id='$Pi' -initsize=100 -log_events -v=6 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/rlogpi'$Pi' 2>&1 &"

#ssh pitter"$Pi" "nohup $SM/client/client -conf $SM/scripts/newList -alg=sm -opt=$1 $3 -mode=bench -contR -gc-off -nclients=1 -id='1$Pi' -initsize=100 -log_events -v=6 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/rlogpi1'$Pi' 2>&1 &"
done

echo " "

sleep 1


if ! [ "$4" == "" ]; then
	echo starting Reconfigurers
	nohup $SM/client/client -conf $SM/scripts/newList -alg=$2 -mode=exp $4 -nclients="$5" -initsize=100 -elog -v=6 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/reconflog 2>&1 &
else
	echo no reconfiguration, waiting 10 seconds
	sleep 10
fi

if [ "$3" == "-cont" ]; then
	echo sleeping 30 seconds
	sleep 30
fi


sleep 1


echo stopping reconfigurers
cd $SM/client && killall client
cd -


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

echo safety stop reconfigurer:
cd $SM/client && killall -9 client
cd -

echo safety stop readers
for Pi in ${READS[@]}
do
	echo -n "pitter$Pi "
	ssh pitter"$Pi" "cd $SM/client && killall -9 client" 
done

echo safety stop servers
for Pi in ${SERVS[@]}
do
	ssh pitter"$Pi" "cd $SM/server && killall -9 server" 
done



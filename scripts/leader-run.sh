#!/bin/sh

#Arguements: $1: reader optimization $2:alg  $3: number of removals, $4: more reader option, e.g. -regular, 

if [ "$1" == "help" ]; then
	echo This script performs leader runs.
	echo Arguments:
	echo "1 reader optimization: no | doreconf"
	echo "2 alg: always using sm"
	echo "3 cprov: normal | thrifty | norecontact"
	echo "4 reconfiguration: -rm -add -repl -cont"
	echo "5 number of reconfiguration clients"
	echo "6 more reader options, e.g. -regular | -logThroughput"
	echo "7 V-level"
	exit
else 
	echo Performing a run with configuration leader. Arguments:
	echo "1 reader optimization: $1"
	echo "2 alg: sm"
	echo "3 cprov: $3"
	echo "4 reconfiguration: $4"
	echo "5 number of reconfiguration clients: $5"
	echo "6 more reader options, $6"
	echo "7 V-level: $7"
	
fi

export SM=$GOPATH/src/github.com/relab/smartMerge

SERVS=(9 10 11 12 14 17 15 19 21)

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

if [ "$3" = "norecontact" ]; then
if [ $Pi == 21 ]; then

	echo -n "leader-pitter$Pi "
	#ssh pitter"$Pi" "nohup $SM/lserver/lserver -port 13000 -no-abort -v=$7 -conf $SM/scripts/leaderList -cprov=$3 -initsize=8 -id=$((100+$Pi)) -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/pi'$Pi'servlog 2>&1 &"
	ssh pitter"$Pi" "nohup $SM/lserver/lserver -port 12000 -no-abort -v=$7 -conf $SM/scripts/leaderList -cprov=$3 -initsize=8 -id=$((100+$Pi)) -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/pi'$Pi'2servlog 2>&1 &"

else

	echo -n "no-pitter$Pi "
	ssh pitter"$Pi" "nohup $SM/server/server -port 13000 -no-abort -v=$7 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/pi'$Pi'servlog 2>&1 &"
	ssh pitter"$Pi" "nohup $SM/server/server -port 12000 -no-abort -v=$7 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/pi'$Pi'2servlog 2>&1 &"

fi

else
if [ $Pi == 21 ]; then

	echo -n "leader-pitter$Pi "
	ssh pitter"$Pi" "nohup $SM/lserver/lserver -port 13000 -v=$7 -conf $SM/scripts/leaderList -cprov=$3 -initsize=8 -id=$((100+$Pi)) -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/pi'$Pi'servlog 2>&1 &"
	ssh pitter"$Pi" "nohup $SM/lserver/lserver -port 12000 -v=$7 -conf $SM/scripts/leaderList -cprov=$3 -initsize=8 -id=$((100+$Pi)) -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/pi'$Pi'2servlog 2>&1 &"

else

	echo -n "sm-pitter$Pi "

	ssh pitter"$Pi" "nohup $SM/server/server -port 13000 -v=$7 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/pi'$Pi'servlog 2>&1 &"
	ssh pitter"$Pi" "nohup $SM/server/server -port 12000 -v=$7 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/pi'$Pi'2servlog 2>&1 &"

fi

fi
done

echo " "

sleep 3


echo single write
$SM/client/client -conf $SM/scripts/leaderList -alg=$2 -mode=bench -writes=1 -size=1000 -nclients=1 -id=5 -initsize=8 

echo starting Readers on
x=0
for Pi in ${READS[@]}
do
	echo -n "pitter$Pi-$x "
ssh pitter"$Pi" "nohup $SM/client/client -conf $SM/scripts/leaderList -alg=$2 -opt=$1 -cprov=$3 $6 -mode=bench -contR -nclients=1 -id=$x -initsize=8 -log_events -v=$7 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/rlogpi'$Pi' 2>&1 &"

	echo -n "pitter$Pi-$(($x+2))"
ssh pitter"$Pi" "nohup $SM/client/client -conf $SM/scripts/leaderList -alg=$2 -opt=$1 -cprov=$3 $6 -mode=bench -contR -nclients=1 -id='$(($x+2))' -initsize=8 -log_events -v=$7 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/rlogpix'$Pi' 2>&1 &"

	echo -n "pitter$Pi-$(($x+4))"
ssh pitter"$Pi" "nohup $SM/client/client -conf $SM/scripts/leaderList -alg=$2 -opt=$1 -cprov=$3 $6 -mode=bench -contR -nclients=1 -id='$(($x+4))' -initsize=8 -log_events -v=$7 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/rlogpiy'$Pi' 2>&1 &"

	echo -n "pitter$Pi-$(($x+6)) "
ssh pitter"$Pi" "nohup $SM/client/client -conf $SM/scripts/leaderList -alg=$2 -opt=$1 -cprov=$3 $6 -mode=bench -contR -nclients=1 -id='$(($x+6))' -initsize=8 -log_events -v=$7 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/rlogpiz'$Pi' 2>&1 &"
x=$(($x+1))
done

echo " "

sleep 1

if [ "$4" = "-cont" ]; then

	echo starting Reconfigurers
	nohup $SM/client/client -conf $SM/scripts/leaderList -useleader -alg=$2 -cprov=$3 -mode=exp $4 -nclients="$5" -initsize=8 -elog -all-cores -v=$7 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/reconflog 2>&1 &
	
	echo sleeping 30 seconds
	sleep 30

elif ! [ "$5" == 0 ]; then
	echo starting Reconfigurers
	$SM/client/client -conf $SM/scripts/leaderList -useleader -alg=$2 -cprov=$3 -mode=exp $4 -nclients="$5" -initsize=8 -elog -all-cores -v=$7 -log_dir='/local/scratch/ljehl' > /local/scratch/ljehl/reconflog 2>&1
else
	echo no reconfiguration, waiting 10 seconds
	sleep 10
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
#ssh pitter"$Pi" "mv /local/scratch/ljehl/*.elog $SM/exlogs"
ssh pitter"$Pi" "mv /local/scratch/ljehl/*log* $SM/exlogs"
done
mv /local/scratch/ljehl/*log* $SM/exlogs

echo stopping servers
for Pi in ${SERVS[@]}
do
	ssh pitter"$Pi" "cd $SM/lserver && killall lserver" 
	ssh pitter"$Pi" "cd $SM/server && killall server" 
	ssh pitter"$Pi" "mv /local/scratch/ljehl/*log* $SM/exlogs"
done

echo safety stop reconfigurer:
cd $SM/client && killall -9 client > /dev/null && echo -n "did kill something"
cd -

echo safety stop readers
for Pi in ${READS[@]}
do
	echo -n "pitter$Pi "
	ssh pitter"$Pi" "cd $SM/client && killall -9 client" > /dev/null && echo -n "did kill something"
done

echo safety stop servers
for Pi in ${SERVS[@]}
do
if [ $Pi == 21 ];then
	ssh pitter"$Pi" "cd $SM/lserver && killall -9 lserver" > /dev/null && echo -n "did kill something"
else 
	ssh pitter"$Pi" "cd $SM/server && killall -9 server" > /dev/null && echo -n "did kill something"
fi
done

echo "leader-run $1 $2 $3 $4 $5 $6" > exlogs/command

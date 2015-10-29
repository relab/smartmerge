#!/bin/sh

cd $SM/client
go build || exit
cd $SM/server
go build || exit


cd "$SM/sm_rm2$*" && echo "File sm_rm2$* exists already. Abort." && exit

for RMS in 1 2 5
do

cd $SM
echo "$RMS removal runs"
echo SmartMerge
mkdir "sm_rm$RMS$*"
mkdir "sm_rm$RMS$*"/events
for i in {1..20} 
do
	echo make run $i
	./scripts/sm-run.sh "$RMS"
	mv $SM/exlogs/*.elog $SM/"sm_rm$RMS$*"/events
	mv $SM/exlogs $SM/"sm_rm$RMS$*"/"run$i"
	echo sleeping 5 seconds
	sleep 5
done
cd $SM
cd "sm_rm$RMS$*"/events
echo 
$SM/scripts/analyzeall analysis

#echo DynaStore
#cd $SM
#for i in {1..20} 
#do
#	./dyna-run.sh "$RMS"
#	mv ../writerslog ../"wlog$i"
#	mv ../reconflog ../"rlog$i"
#done
#cd $SM
#mkdir "dyna_rm$RMS$*"
#mv *.elog "dyna_rm$RMS$*/" 
#mv wlog*  "dyna_rm$RMS$*/" 
#mv rlog*  "dyna_rm$RMS$*/" 
#cd "dyna_rm$RMS$*"
#echo Analyzing
#$SM/scripts/analyzeall "dyna_rm$RMS$*"

#echo Original DynaStore
#cd ../scripts
#for i in {1..20} 
#do
#	./orgd-run.sh "$RMS"
#	mv ../writerslog ../"wlog$i"
#	mv ../reconflog ../"rlog$i"
#done
#cd $SM
#mkdir "orgd_rm$RMS$*"
#mv *.elog "orgd_rm$RMS$*/" 
#mv wlog*  "orgd_rm$RMS$*/" 
#mv rlog*  "orgd_rm$RMS$*/" 
#cd "orgd_rm$RMS$*"
#echo Analyzing
#$SM/scripts/analyzeall "orgd_rm$RMS$*" 4

echo Consensus Based
cd $SM
mkdir "cons_rm$RMS$*"
mkdir "cons_rm$RMS$*"/events
for i in {1..20} 
do
	echo make run $i
	./scripts/cons-run.sh "$RMS"
	mv $SM/exlogs/*.elog $SM/"cons_rm$RMS$*"/events
	mv $SM/exlogs $SM/"cons_rm$RMS$*"/"run$i"
	echo sleeping 5 seconds
	sleep 5
done
cd $SM/"cons_rm$RMS$*"/events
echo Analyzing
$SM/scripts/analyzeall analysis


done

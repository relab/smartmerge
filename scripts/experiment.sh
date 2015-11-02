#!/bin/sh

cd $SM/client
go build || exit
cd $SM/server
go build || exit


cd "$SM/sm_rm2$*" && echo "File sm_rm2$* exists already. Abort." && exit

for RMS in 5 2
do

: <<'END'
cd $SM
echo "$RMS removal runs"
echo SmartMerge
mkdir "sm_rm$RMS$*"
for i in {1..19} 
do
	echo make run $i
	./scripts/sm-run.sh "$RMS"
	mv $SM/exlogs $SM/"sm_rm$RMS$*"/"run$i"
	echo sleeping 5 seconds
	sleep 5
done
cd "sm_rm$RMS$*"

echo checking
mkdir problem
for R in run*; do
	cd $R
	if ls ./*ERROR* > /dev/null 2>&1; then
		cd ..
		mv $R problem/
	fi
	cd $SM/"sm_rm$RMS$*"
done
for R in run*; do
	cd $R
	$SM/scripts/checkall debug || {
		cd ..  
		mv $R problem/
	}
	cd $SM/"sm_rm$RMS$*"
done
rmdir problem || echo some runs had problems		
echo analysing
$SM/scripts/analyzeallsub analysis
END

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
for i in {1..19} 
do
	echo make run $i
	./scripts/cons-run.sh "$RMS"
	mv $SM/exlogs $SM/"cons_rm$RMS$*"/"run$i"
	echo sleeping 5 seconds
	sleep 5
done
cd $SM/"cons_rm$RMS$*"

echo checking
mkdir problem
for R in run*; do
	cd $R
	if ls ./*ERROR* > /dev/null 2>&1; then
		cd ..
		mv $R problem/
	fi
	cd $SM/"cons_rm$RMS$*"
done

for R in run*; do
	cd $R
	$SM/scripts/checkall debug || {
		cd ..  
		mv $R problem/
	}
	cd $SM/"cons_rm$RMS$*"
done
rmdir problem || echo some runs had problems		
echo analysing
$SM/scripts/analyzeallsub analysis

done

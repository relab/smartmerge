#!/bin/sh

cd $SM/client
#go build || exit
cd $SM/server
#go build || exit


cd "$SM/sm_rm2$*" && echo "File sm_rm2$* exists already. Abort." && exit

for RMS in 4
do

echo "$RMS removal runs"

for Opt in "doreconf" "no" "norecontact"
do

#: <<'END'
cd $SM
echo SmartMerge with optimization $Opt
mkdir "sm_opt$Opt-rm$RMS$*"
for i in {1..20} 
do
	echo make run $i
	./scripts/sm-run.sh "$Opt" "sm" "$RMS"
	mv $SM/exlogs $SM/"sm_opt$Opt-rm$RMS$*"/"run$i"
	echo sleeping 5 seconds
	sleep 5
done
cd "sm_opt$Opt-rm$RMS$*"

echo checking
mkdir problem
for R in run*; do
	cd $R
	if ls ./*ERROR* > /dev/null 2>&1; then
		cd ..
		mv $R problem/
	fi
	cd $SM/"sm_opt$Opt-rm$RMS$*"
done
for R in run*; do
	$SM/scripts/checkall $R || mv $R problem/
done
rmdir problem || echo some runs had problems		
echo analysing
$SM/scripts/analyzeallsub analysis $RMS 5
#END

cd $SM
echo SmartMerge with optimization $Opt regular reads
mkdir "sm_regopt$Opt-rm$RMS"
for i in {1..20} 
do
	echo make run $i
	./scripts/sm-run.sh "$Opt" "sm" "$RMS" "-regular"
	mv $SM/exlogs $SM/"sm_regopt$Opt-rm$RMS$*"/"run$i"
	echo sleeping 5 seconds
	sleep 5
done
cd "sm_regopt$Opt-rm$RMS$*"

echo checking
mkdir problem
for R in run*; do
	cd $R
	if ls ./*ERROR* > /dev/null 2>&1; then
		cd ..
		mv $R problem/
	fi
	cd $SM/"sm_regopt$Opt-rm$RMS$*"
done
for R in run*; do
	$SM/scripts/checkall $R 1 || mv $R problem/
done
rmdir problem || echo some runs had problems		
echo analysing
$SM/scripts/analyzeallsub analysis $RMS 5 1


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

echo Consensus Based with optimization $Opt
cd $SM
mkdir "cons_opt$Opt-rm$RMS$*"
for i in {1..20} 
do
	echo make run $i
	./scripts/sm-run.sh "$Opt" "cons" "$RMS"
	mv $SM/exlogs $SM/"cons_opt$Opt-rm$RMS$*"/"run$i"
	echo sleeping 5 seconds
	sleep 5
done
cd $SM/"cons_opt$Opt-rm$RMS$*"

echo checking
mkdir problem
for R in run*; do
	cd $R
	if ls ./*ERROR* > /dev/null 2>&1; then
		cd ..
		mv $R problem/
	fi
	cd $SM/"cons_opt$Opt-rm$RMS$*"
done

for R in run*; do   
	$SM/scripts/checkall $R || mv $R problem/
done
rmdir problem || echo some runs had problems		
echo analysing
$SM/scripts/analyzeallsub analysis $RMS 5
END

echo Consensus Based with optimization $Opt regular
cd $SM
mkdir "cons_regopt$Opt-rm$RMS$*"
for i in {1..20} 
do
	echo make run $i
	./scripts/sm-run.sh "$Opt" "cons" "$RMS" "-regular"
	mv $SM/exlogs $SM/"cons_regopt$Opt-rm$RMS$*"/"run$i"
	echo sleeping 5 seconds
	sleep 5
done
cd $SM/"cons_regopt$Opt-rm$RMS$*"

echo checking
mkdir problem
for R in run*; do
	cd $R
	if ls ./*ERROR* > /dev/null 2>&1; then
		cd ..
		mv $R problem/
	fi
	cd $SM/"cons_regopt$Opt-rm$RMS$*"
done

for R in run*; do
	$SM/scripts/checkall $R 1 || mv $R problem/
done
rmdir problem || echo some runs had problems		
echo analysing
$SM/scripts/analyzeallsub analysis $RMS 5 1


done
done

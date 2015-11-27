#!/bin/sh


cd "$SM/sm_opt*" && echo "File sm_opt exists already. Abort." && exit

for RMS in 0
do

echo "$RMS replacement runs"

for Opt in "no"
do

for ALG in "sm" "cons" "dyna" "ssr"; do

cd $SM
echo Alg $ALG with optimization $Opt
mkdir "$ALG-opt$Opt-rm$RMS$*"
for i in {1..1} 
do
	echo make run $i
	./scripts/sm-run.sh "$Opt" $ALG thrifty -rm "$RMS" " " 0
	mv $SM/exlogs $SM/"$ALG-opt$Opt-rm$RMS$*"/"run$i"
	echo sleeping 3 seconds
	sleep 3
done
cd "$ALG-opt$Opt-rm$RMS$*"

echo checking
mkdir problem
for R in run*; do
	cd $R
	if ls ./*ERROR* > /dev/null 2>&1; then
		cd ..
		mv $R problem/
	fi
	cd $SM/"$ALG-opt$Opt-rm$RMS$*"
done
for R in run*; do
	$SM/scripts/checkall $R || mv $R problem/
done
rmdir problem || echo some runs had problems		
echo analysing
$SM/scripts/analyzeallsub analysis $RMS 12

: <<'END'
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
END

done #for ALG
done #for Opt
done #for RMS

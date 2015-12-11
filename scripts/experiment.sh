#!/bin/sh


cd "$SM/sm_opt*" && echo "File sm_opt exists already. Abort." && exit

for RMS in 5
do

echo "$RMS replacement runs"

for Opt in "no" #"doreconf" 
do

for CP in "thrifty" #"norecontact"  
do

for ALG in "ssr" "dyna" #"sm" "cons" #"ssr" "dyna"
do

if [ "$ALG" = "ssr" -a "$CP" = "norecontact" ]; then
	exit
	echo Skipping alg $ALG with optimization $Opt cp: $CP 
else


if [ "$Opt" = "no" -a "$CP" = "thrifty" -a $RMS == 3 ]; then
	echo Skip $ALG, $Opt, $CP.
else

cd $SM
echo Alg $ALG with optimization $Opt conf provider $CP


mkdir "$ALG-opt$Opt-cp$CP-repl$RMS$*"
for i in {1..20} 
do
	echo make run $i
	./scripts/sm-run.sh "$Opt" $ALG $CP "-repl" "$RMS" -logThroughput 0
	mv $SM/exlogs $SM/"$ALG-opt$Opt-cp$CP-repl$RMS$*"/"run$i"
	echo sleeping 3 seconds
	sleep 3
done
cd "$ALG-opt$Opt-cp$CP-repl$RMS$*"

echo checking
mkdir problem
for R in run*; do
	cd $R
	if ls ./*ERROR* > /dev/null 2>&1; then
		cd ..
		mv $R problem/
	fi
	cd $SM/"$ALG-opt$Opt-cp$CP-repl$RMS$*"
done
#for R in run*; do
#	$SM/scripts/checkall $R || mv $R problem/
#done
rmdir problem || echo some runs had problems		
echo analysing
$SM/scripts/analyzeallsub analysis
fi

: <<'END'
cd $SM
echo Alg $ALG with optimization $Opt regular conf prov $CP


mkdir "$ALG-regopt$Opt-cp$CP-repl$RMS$*"
for i in {1..40} 
do
	echo make run $i
	./scripts/sm-run.sh "$Opt" $ALG $CP -repl "$RMS" "-regular" 0
	mv $SM/exlogs $SM/"$ALG-regopt$Opt-cp$CP-repl$RMS$*"/"run$i"
	echo sleeping 3 seconds
	sleep 3
done
cd "$ALG-regopt$Opt-cp$CP-repl$RMS$*"

echo checking
mkdir problem
for R in run*; do
	cd $R
	if ls ./*ERROR* > /dev/null 2>&1; then
		cd ..
		mv $R problem/
	fi
	cd $SM/"$ALG-regopt$Opt-cp$CP-repl$RMS$*"
done
for R in run*; do
	$SM/scripts/checkall $R || mv $R problem/
done
rmdir problem || echo some runs had problems		
echo analysing
$SM/scripts/analyzeallsub analysis
END
fi

: <<'END'

END

done #for ALG
done #for CP
done #for Opt
done #for RMS

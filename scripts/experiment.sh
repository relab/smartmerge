#!/bin/sh


cd "$SM/sm_opt*" && echo "File sm_opt exists already. Abort." && exit

for RMS in 1
do

echo "$RMS replacement runs"

for Opt in "no" #"doreconf" 
do

for CP in "thrifty" #"norecontact"  #"thrifty" 
do

for ALG in  "dyna" #"sm" "cons" "ssr" 
do

if [ "$ALG" = "ssr" -a "$CP" = "norecontact" ]; then
	echo Skipping alg $ALG with optimization $Opt cp: $CP 
elif [ "$ALG" = "dyna" -a "$CP" = "norecontact" ]; then
	echo Skipping alg $ALG with optimization $Opt cp: $CP 
else



cd $SM
echo Alg $ALG with optimization $Opt conf provider $CP


mkdir "$ALG-reg-opt$Opt-cp$CP-repl$RMS$*"
for i in {41..60} 
do
	echo make run $i
	./scripts/sm-run.sh "$Opt" $ALG $CP "-repl" "$RMS" "-regular" 7
	mv $SM/locexlogs $SM/"$ALG-reg-opt$Opt-cp$CP-repl$RMS$*"/"run$i"
	echo sleeping 3 seconds
	sleep 3
done
cd "$ALG-reg-opt$Opt-cp$CP-repl$RMS$*"

echo checking
mkdir problem
for R in run*; do
	cd $R
	if ls ./*ERROR* > /dev/null 2>&1; then
		cd ..
		mv $R problem/
	fi
	cd $SM/"$ALG-reg-opt$Opt-cp$CP-repl$RMS$*"
done
for R in run*; do
	$SM/scripts/checkall $R || mv $R problem/
done
rmdir problem || echo some runs had problems		
echo analysing
$SM/scripts/analyzeallsub analysis 1960 1

fi

: <<'END'

END

done #for ALG
done #for CP
done #for Opt
done #for RMS

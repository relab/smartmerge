#!/bin/sh

#for i in {1..20} 
#do
#	./sm-run.sh
#done
#cd $SM
#mkdir sm_rm2
#mv *.elog sm_rm2/ 
#cd sm_rm2
#$SM/scripts/analyzeall sm_remove2

cd ../scripts
for i in {1..20} 
do
	./dyna-run.sh
done
cd $SM
mkdir dyna_rm2
mv *.elog dyna_rm2/ 
cd dyna_rm2
echo Analyzing
$SM/scripts/analyzeall dyna_remove2

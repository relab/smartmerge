#!/bin/sh

for i in {1..20} 
do
	./sm-run.sh 5
	continue
done
cd $SM
mkdir sm_rm5
mv *.elog sm_rm5/ 
cd sm_rm5
$SM/scripts/analyzeall sm_remove5

cd ../scripts
for i in {1..20} 
do
	./dyna-run.sh 5
done
cd $SM
mkdir dyna_rm5
mv *.elog dyna_rm5/ 
cd dyna_rm5
echo Analyzing
$SM/scripts/analyzeall dyna_remove5

exit

for i in {1..2} 
do
	./sm-run.sh 2
	continue
done
cd $SM
mkdir sm_rm2
mv *.elog sm_rm2/ 
cd sm_rm2
$SM/scripts/analyzeall sm_remove2

cd ../scripts
for i in {1..2} 
do
	./dyna-run.sh 2
done
cd $SM
mkdir dyna_rm2
mv *.elog dyna_rm2/ 
cd dyna_rm2
echo Analyzing
$SM/scripts/analyzeall dyna_remove2

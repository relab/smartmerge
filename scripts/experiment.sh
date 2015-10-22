#!/bin/sh

cd $SM/client
go install
cd -

cd "$SM/sm_rm2$*" && echo "File sm_rm2$* exists already. Abort." && exit

for RMS in 0 1 2 5
do

cd $SM/scripts
echo "$RMS removal runs"
echo SmartMerge
for i in {1..20} 
do
	./sm-run.sh "$RMS"
	mv ../writerslog ../"wlog$i"
	mv ../reconflog ../"rlog$i"
done
cd $SM
mkdir "sm_rm$RMS$*"
mv *.elog "sm_rm$RMS$*/" 
mv wlog*  "sm_rm$RMS$*/" 
mv rlog*  "sm_rm$RMS$*/" 
cd "sm_rm$RMS$*"
$SM/scripts/analyzeall "sm_rm$RMS$*"

echo DynaStore
cd ../scripts
for i in {1..20} 
do
	./dyna-run.sh "$RMS"
	mv ../writerslog ../"wlog$i"
	mv ../reconflog ../"rlog$i"
done
cd $SM
mkdir "dyna_rm$RMS$*"
mv *.elog "dyna_rm$RMS$*/" 
mv wlog*  "dyna_rm$RMS$*/" 
mv rlog*  "dyna_rm$RMS$*/" 
cd "dyna_rm$RMS$*"
echo Analyzing
$SM/scripts/analyzeall "dyna_rm$RMS$*"

echo Original DynaStore
cd ../scripts
for i in {1..20} 
do
	./orgd-run.sh "$RMS"
	mv ../writerslog ../"wlog$i"
	mv ../reconflog ../"rlog$i"
done
cd $SM
mkdir "orgd_rm$RMS$*"
mv *.elog "orgd_rm$RMS$*/" 
mv wlog*  "orgd_rm$RMS$*/" 
mv rlog*  "orgd_rm$RMS$*/" 
cd "orgd_rm$RMS$*"
echo Analyzing
$SM/scripts/analyzeall "orgd_rm$RMS$*" 4

echo Consensus Based
cd ../scripts
for i in {1..20} 
do
	./cons-run.sh "$RMS"
	mv ../writerslog ../"wlog$i"
	mv ../reconflog ../"rlog$i"
done
cd $SM
mkdir "cons_rm$RMS$*"
mv *.elog "cons_rm$RMS$*/" 
mv wlog*  "cons_rm$RMS$*/" 
mv rlog*  "cons_rm$RMS$*/" 
cd "cons_rm$RMS$*"
echo Analyzing
$SM/scripts/analyzeall "cons_rm$RMS$*"


done

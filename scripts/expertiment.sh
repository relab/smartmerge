#!/bin/sh

cd $SM/client
go build
cd -

cd "$SM/sm_rm2$*" && echo "File sm_rm2$* exists already. Abort." && exit

echo No Reconf runs
echo No Reconf SmartMerge
for i in {1..20} 
do
	./sm-run.sh 
done
cd $SM
mkdir "sm_nor$*"
mv *.elog "sm_nor$*/" 
cd "sm_nor$*"
$SM/scripts/analyzeall "sm_nor$*"

echo No Reconf DynaStore
cd ../scripts
for i in {1..20} 
do
	./dyna-run.sh
done
cd $SM
mkdir "dyna_nor$*"
mv *.elog "dyna_nor$*/" 
cd "dyna_nor$*"
echo Analyzing
$SM/scripts/analyzeall "dyna_nor$*"

echo 1 removal runs
echo 1 removal in SmartMerge
for i in {1..20} 
do
	./sm-run.sh 1 
done
cd $SM
mkdir "sm_nor$*"
mv *.elog "sm_nor$*/" 
cd "sm_nor$*"
$SM/scripts/analyzeall "sm_nor$*"

echo 1 removal in DynaStore
cd ../scripts
for i in {1..20} 
do
	./dyna-run.sh
done
cd $SM
mkdir "dyna_nor$*"
mv *.elog "dyna_nor$*/" 
cd "dyna_nor$*"
echo Analyzing
$SM/scripts/analyzeall "dyna_nor$*"

echo 2 Removals experiment
echo 2 Removals in SmartMerge
for i in {1..20} 
do
	./sm-run.sh 2
done
cd $SM
mkdir "sm_rm2$*"
mv *.elog "sm_rm2$*/" 
cd "sm_rm2$*"
$SM/scripts/analyzeall "sm_remove2$*"

echo 2 Removals in DynaStore
cd ../scripts
for i in {1..20} 
do
	./dyna-run.sh 2
done
cd $SM
mkdir "dyna_rm2$*"
mv *.elog "dyna_rm2$*/" 
cd "dyna_rm2$*"
echo Analyzing
$SM/scripts/analyzeall "dyna_remove2$*"

echo 5 Removals experiment
echo 5 Removals in SmartMerge
cd $SM/scripts
for i in {1..20} 
do
	./sm-run.sh 5
done
cd $SM
mkdir "sm_rm5$*"
mv *.elog "sm_rm5$*/" 
cd "sm_rm5$*"
$SM/scripts/analyzeall "sm_remove5$*"

echo 5 Removals in DynaStore
cd ../scripts
for i in {1..20} 
do
	./dyna-run.sh 5
done
cd $SM
mkdir "dyna_rm5$*"
mv *.elog "dyna_rm5$*/" 
cd "dyna_rm5$*"
echo Analyzing
$SM/scripts/analyzeall "dyna_remove5$*"



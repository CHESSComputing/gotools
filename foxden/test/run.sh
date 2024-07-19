#!/bin/bash
cdir=/Users/vk/Work/CHESS/FOXDEN
ddir=$cdir/DataBookkeeping
sdir=$cdir/SpecScansService
idir=$cdir/gotools/foxden/test/data
schema=ID3A

echo
echo "get write token"
token=`./foxden token create write | head -1`
export FOXDEN_WRITE_TOKEN=$token
# echo "FOXDEN_WRITE_TOKEN=$FOXDEN_WRITE_TOKEN"
echo
echo "get read token"
token=`./foxden token create read | head -1`
export FOXDEN_TOKEN=$token
# echo "FOXDEN_TOKEN=$FOXDEN_TOKEN"
echo
echo "view issued tokens"
./foxden token view

echo
echo "remove $ddir/dbs.db"
rm $ddir/dbs.db
sqlite3 $ddir/dbs.db < $ddir/static/schema/sqlite.sql

echo
echo "remove $sdir/motors.db"
rm $sdir/motors.db
sqlite3 $sdir/motors.db < $sdir/static/sql/create_tables.sql

echo
echo "+++ ADD NEW MetaData RECORDS"
echo
echo
echo "+++ Add new meta-data record $idir/ID3A-meta1-foxden.json"
./foxden meta add $schema $idir/ID3A-meta1-foxden.json

echo
echo "+++ Add new meta-data record $idir/ID3A-meta2-foxden.json"
./foxden meta add $schema $idir/ID3A-meta2-foxden.json

echo
echo "+++ Add new meta-data record $idir/ID3A-meta-wrong-foxden.json"
echo "+++ MUST GET ERROR during insertion of the record"
./foxden meta add $schema $idir/ID3A-meta-wrong-foxden.json 2>&1 1>& /tmp/foxden_error.txt
grep ERROR /tmp/foxden_error.txt

echo
echo "### MetaData records: via search {}"
./foxden search {}
echo
echo "### MetaData records: via meta ls"
./foxden meta ls

echo
echo "+++ ADD NEW DBS RECORDS"
echo
echo "+++ Add new dbs-data record $idir/ID3A-dbs1.json"
./foxden prov add $idir/ID3A-dbs1.json
echo
echo "+++ Add new dbs-data record $idir/ID3A-dbs2.json"
./foxden prov add $idir/ID3A-dbs2.json

echo
echo "+++ Add new dbs-data record $idir/ID3A-dbs3.json"
./foxden prov add $idir/ID3A-dbs3.json

echo
echo "+++ Add parent record"
./foxden prov add-parent $idir/parent.json

echo
echo "+++ ADD NEW SpecScan DATA RECORDS"
echo
echo "+++ Add new SpecScans data record $idir/ID3A-specscan-1.json"
./foxden spec add $idir/ID3A-specscan-1.json
echo "+++ Add new SpecScans data record $idir/ID3A-specscan-2.json"
./foxden spec add $idir/ID3A-specscan-2.json
echo "+++ Add new SpecScans data record $idir/ID3A-specscan-3.json"
./foxden spec add $idir/ID3A-specscan-3.json
echo
echo "View spec records"
did=`grep did test/data/ID3A-specscan-1.json | awk '{print $2}'`
./foxden spec view $did

echo
echo "+++ search for all records"
./foxden search {}

echo
echo "+++ view record /beamline=3a/btr=3731-b/cycle=2023-3"
./foxden view /beamline=3a/btr=3731-b/cycle=2023-3

echo
echo "+++ view records /beamline=3a,4b/btr=3731-b/cycle=2023-3/sample_name=test-1"
./foxden view /beamline=3a,4b/btr=3731-b/cycle=2023-3/sample_name=test-1

echo
echo "+++ view records /beamline=3a,4b/btr=3731-b/cycle=2023-3/sample_name=test-2"
./foxden view /beamline=3a,4b/btr=3731-b/cycle=2023-3/sample_name=test-2

echo
echo "+++ test dataset id updates"
echo "+++ adding empty dbs record with some did"
did=`cat test/data/ID3A-dbs1-empty.json | grep did | awk '{print $2}' | sed -e "s,\",,g"`
./foxden prov add test/data/ID3A-dbs1-empty.json
echo "+++ list files of our did"
./foxden prov ls files --did=$did
echo "+++ update dbs record with new files"
./foxden prov add test/data/ID3A-dbs1-empty-files.json
echo "+++ list files of our did"
./foxden prov ls files --did=$did
echo "+++ update dbs record with processing info"
./foxden prov add test/data/ID3A-dbs1-empty-proc.json
echo "+++ list files of our did"
./foxden prov ls files --did=$did
echo "+++ list parents of our did"
./foxden prov ls parents --did=$did
echo "+++ list children of our did"
./foxden prov ls child --did=$did


echo
export FOXDEN_WRITE_TOKEN=$FOXDEN_TOKEN
schema=test
wfile=/tmp/foxden.wrong.token
./foxden meta add $schema $idir/test-data.json 2>&1 1>& $wfile
invalid=`grep invalid $wfile`
if [ -z "$invalid" ]; then
    echo "+++ use read token for writing, test failed"
else
    echo "+++ use read token for writing, test success"
fi

#!/bin/bash
cdir=/Users/vk/Work/CHESS/FOXDEN
dbname=foxden
if [ "`hostname -s`" == "foxden-dev" ]; then
    export FOXDEN_CONFIG=$HOME/.foxden-dev.yaml
    export FOXDEN_MONGODB_PORT=27017
    cdir=/home/chessdata_svc/FOXDEN
    dbname=foxden-dev
fi
if [ "`hostname -s`" == "lnx15" ]; then
    echo "We are not suppose to run this test on lnx15 (foxden) node"
    exit 1
fi
ddir=$cdir/DataBookkeeping
sdir=$cdir/SpecScansService
idir=$cdir/gotools/foxden/test/data
schema=ID3A
mongoPort=${FOXDEN_MONGODB_PORT:-8230}

echo "MongoDB port: $mongoPort"
echo "MongoDB db  : $dbname"

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
echo "cleanup MetaData database chess.meta and chess.spec"
if [ -f /tmp/cleanup.js ]; then
    rm /tmp/cleanup.js
fi
echo "use $dnname" > /tmp/cleanup.js
cat >> /tmp/cleanup.js << EOF
db.meta.remove({});
db.meta.count();
db.specscans.remove({});
db.specscans.count();
EOF
mongo --port $mongoPort < /tmp/cleanup.js

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
echo "+++ Test wrong DID in meta-data record"
./foxden meta add $schema $idir/ID3A-meta1-foxden-did-nil.json 2>&1 1>& /tmp/foxden_error_did.txt
cat /tmp/foxden_error_did.txt

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
did=`grep did test/data/ID3A-specscan-1.json | awk '{print $2}'`
# echo "View spec records for did:$did"
# ./foxden spec view did:$did
echo "query spec service with more complex query, e.g. '{"btr": "3731-b", "motors": {"monu_pitch": 0.26898878}}'"
./foxden spec view '{"btr": "3731-b", "motors": {"monu_pitch": 0.26898878}}' | grep scan_number

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

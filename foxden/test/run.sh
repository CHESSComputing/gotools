#!/bin/bash
cdir=/Users/vk/Work/CHESS/FOXDEN
ddir=$cdir/DataBookkeeping
idir=$cdir/gotools/foxden/test/data
schema=ID3A

echo
echo "get write token"
token=`./foxden auth token write /tmp/krb5cc_502 | head -1`
export CHESS_WRITE_TOKEN=$token
# echo "CHESS_WRITE_TOKEN=$CHESS_WRITE_TOKEN"
echo
echo "get read token"
token=`./foxden auth token /tmp/krb5cc_502 | head -1`
export CHESS_TOKEN=$token
# echo "CHESS_TOKEN=$CHESS_TOKEN"
echo
echo "view issued tokens"
./foxden auth token view

echo
echo "remove dbs.db"
rm $ddir/dbs.db
sqlite3 $ddir/dbs.db < $ddir/static/schema/sqlite.sql

echo
echo "+++ Add new meta-data record $idir/ID3A-meta1.json"
./foxden meta add $schema $idir/ID3A-meta1.json

echo
echo "+++ Add new meta-data record $idir/ID3A-meta2.json"
./foxden meta add $schema $idir/ID3A-meta2.json

echo
echo "+++ Add new dbs-data record $idir/ID3A-dbs1.json"
./foxden prov add $idir/ID3A-dbs1.json
echo
echo "+++ Add new dbs-data record $idir/ID3A-dbs2.json"
./foxden prov add $idir/ID3A-dbs2.json

echo
echo "+++ search for all records"
./foxden search {}

echo
echo "+++ view record /beamline=3a/btr=3731-b/cycle=2023-3"
./foxden view /beamline=3a/btr=3731-b/cycle=2023-3

echo
echo "+++ view records xyz"
./foxden view xyz

echo
echo "+++ test read token for writing (must fail)"
export CHESS_WRITE_TOKEN=$CHESS_TOKEN
schema=test
./foxden meta add $schema $idir/test-data.json

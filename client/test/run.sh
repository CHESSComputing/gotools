#!/bin/bash
cdir=/Users/vk/Work/CHESS
ddir=$cdir/DataBookkeeping
idir=$cdir/gotools/client/test/data
schema=ID3A

echo
echo "get client token"
token=`./client auth token /tmp/krb5cc_502 | head -1`
export CHESS_WRITE_TOKEN=$token
echo "CHESS_WRITE_TOKEN=$CHESS_WRITE_TOKEN"
token=`./client auth token /tmp/krb5cc_502 | head -1`
export CHESS_TOKEN=$token
echo "CHESS_TOKEN=$CHESS_WRITE_TOKEN"

echo
echo "remove dbs.db"
rm $ddir/dbs.db
sqlite3 $ddir/dbs.db < $ddir/static/schema/sqlite-schema.sql

echo
echo "+++ Add new meta-data record $idir/ID3A-meta1.json"
./client meta add $schema $idir/ID3A-meta1.json
echo
echo "+++ Add new meta-data record $idir/ID3A-meta2.json"
./client meta add $schema $idir/ID3A-meta2.json

echo
echo "+++ Add new dbs-data record $idir/ID3A-dbs1.json"
./client dbs add $idir/ID3A-dbs1.json
echo
echo "+++ Add new dbs-data record $idir/ID3A-dbs2.json"
./client dbs add $idir/ID3A-dbs2.json

echo
echo "+++ search for all records"
./client search {}

echo
echo "+++ view record abc"
./client view abc

echo
echo "+++ view records xyz"
./client view xyz

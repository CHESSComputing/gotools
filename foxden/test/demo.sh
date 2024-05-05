#!/bin/bash
cdir=/Users/vk/Work/CHESS/FOXDEN
ddir=$cdir/DataBookkeeping
idir=$cdir/gotools/foxden/test/data
schema=ID3A

echo
echo "create write token"
echo "/foxden token create write"
token=`./foxden token create write | head -1`
export FOXDEN_WRITE_TOKEN=$token
# echo "FOXDEN_WRITE_TOKEN=$FOXDEN_WRITE_TOKEN"
echo
echo "create read token"
echo "/foxden token create read"
token=`./foxden token create read | head -1`
export FOXDEN_TOKEN=$token
# echo "FOXDEN_TOKEN=$FOXDEN_TOKEN"
echo
echo "view issued tokens"
./foxden token view

sleep 5

echo
echo "remove dbs.db"
rm $ddir/dbs.db
sqlite3 $ddir/dbs.db < $ddir/static/schema/sqlite.sql

echo
echo "clear up MongoDB"
cat > /tmp/cleanup.js << EOF
use chess;
db.meta.remove({});
EOF
mongo --port 8230 < /tmp/cleanup.js 2>&1 1>& /dev/null

echo
echo "+++ Add new meta-data record $idir/ID3A-meta1.json"
echo "./foxden meta add $schema $idir/ID3A-meta1.json"
./foxden meta add $schema $idir/ID3A-meta1.json

sleep 2

echo
echo "+++ Add new dbs-data record $idir/ID3A-dbs1.json"
echo "./foxden prov add $idir/ID3A-dbs1.json"
./foxden prov add $idir/ID3A-dbs1.json

sleep 2

echo
echo "+++ search for all records"
echo "./foxden search {}"
./foxden search {}

sleep 2

echo
echo "+++ view record /beamline=3a/btr=3731-b/cycle=2023-3"
echo "./foxden view /beamline=3a/btr=3731-b/cycle=2023-3"
./foxden view /beamline=3a/btr=3731-b/cycle=2023-3

### Update tool
This directory contains codebase for updating MetaData records in FOXDEN.
It has very specific logic, so far code provides how to add globus link
to all existing meta-data records. Therefore, if we'll need to update
further our records the code should be properly adjusted.

```
# compile the code
make

# update records
uri="mongodb://localhost:27017"
db=dbName
col=dbColl
./metaupdate -uri $uri -DBName $db -Collection $col
```

### Migrate tool
This directory contains codebase for migrate tool of porting
set of meta-data records from MetaData service to FOXDEN one.
During the transition a proper did is constructured based
on default attributes: beamline, btr, cycle, sample

```
#!/bin/bash

# example of migrating data from readUri to writeUri
# please adjust your uri, db name and collection accordingly to your use-case

readUri="mongodb://localhost:27017"
readDB=dbName
readCol=dbColl
writeUri=mongodb://localhost:27888
writeDB=dbName
writeCol=dbColl

./migrate -readUri "$readUri" -readDBName $readDB -readCollection $readCol \
        -writeUri "$writeUri" -writeDBName $writeDB -writeCollection $writeCol -verbose
```

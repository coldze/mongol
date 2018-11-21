# MongoDB + Liquibase = mongol

Liquibase-like tool for MongoDB.

## Docker-hub:

https://hub.docker.com/r/coldze/mongol/

## Functionality

* forward migrations
```
mongol migrate --path=/path/to/changelog.json --count=123
```

* backward migrations/rollbacks
```
mongol rollback --path=/path/to/changelog.json --count=123
```

## Test

```
go get -u github.com/coldze/mongol
```

Change mongo-db settings in test/changelog.json and run:
```
cd $GOPATH/src/github.com/coldze/mongol
mkdir build && cd build && go build .. && ./mongol migrate --path=../test/changelog.json
```

## Docker-way

* forward migrations
```
docker run --rm -v /path/to/src:/mongol/src coldze/mongol:latest mongol migrate --path=/mongol/src/changelog.json 
```

* backward migrations
```
docker run --rm -v /path/to/src:/mongol/src coldze/mongol:latest mongol rollback --path=/mongol/src/changelog.json 
```

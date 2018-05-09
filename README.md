# MongoDB + Liquibase = mongol

Liquibase-like tool for MongoDB.

## Functionality

Works only forward migration, no rollback, all migrations are applied at once, no checking of migration's history. Development at early stage.

## Test

```
go get -u github.com/coldze/mongol
```

Change mongo-db settings in test/changelog.json and run:
```
cd $GOPATH/src/github.com/coldze/mongol
mkdir build && cd build && go build .. && ./mongol --changelog=../test/changelog.json
```

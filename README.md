# MongoDB + Liquibase = mongol

Liquibase-like tool for MongoDB.

## Installation

### Docker-way:

* forward migrations
```
docker run --rm -v /path/to/src:/mongol/src coldze/mongol:latest mongol migrate --path=/mongol/src/changelog.json 
```

* backward migrations
```
docker run --rm -v /path/to/src:/mongol/src coldze/mongol:latest mongol rollback --path=/mongol/src/changelog.json 
```

### Compile from source code:
1. Install go: https://golang.org/doc/install
2. Don't forget to add `$GOBIN` to your `$PATH`
3. Run the following:
```
go get -u github.com/kardianos/govendor //govendor tool to handle dependecies, see vendor/vendor.json
go get github.com/coldze/mongol
cd $GOPATH/src/github.com/coldze/mongol
govendor sync
go install
mongol --help
``` 


## Docker-hub:

https://hub.docker.com/r/coldze/mongol/

## Functionality

* forward migrations:
```
mongol migrate --path=/path/to/changelog.json --count=123
```

* backward migrations/rollbacks. You must specify amount of migrations to rollback. `Count` takes into consideration *ALL* migrations, specified in `changelog.json`, whether they were applied or not. So if you have 10 migrations in total, but only 5 were applied and you want to rollback last 2, you will have to specify `count=7` (last 5 missing, 2 to rollback): 
```
mongol rollback --path=/path/to/changelog.json --count=7
```

* migrations must be in a valid `extended-json` format:
https://github.com/mongodb/specifications/blob/master/source/extended-json.rst

* migrations support MongoDB database commands:
https://docs.mongodb.com/manual/reference/command/

* pay attention, that different versions of MongoDB support different sets of commands.

* migrations support sets of commands, just follow this syntax:
```
{
  "cmds" [
    {
      //first command in extended json format
    },
    {
      //second command in extended json format
    }
    ...
  ]
}
```

* MongoDB supports JavaScript starting from 3.0 up to 3.6, in 4.0 it was deprecated, so you're able to use `eval` in migrations for MongoDB 3.x versions.

## Running a test

Change mongo-db settings in `$GOPATH/src/github.com/coldze/mongol/test/changelog.json` and execute:
```
mongol migrate --path=$GOPATH/src/github.com/coldze/mongol/test/changelog.json
```

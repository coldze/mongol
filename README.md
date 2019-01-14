# MongoDB + Liquibase = mongol

Liquibase-like tool for MongoDB. Manipulating structure and data of MongoDB in migration-based way.

##How-to

#### Structure:
- Mongol expects that `migrate` and `rollback` commands will be provided with a file (default ./changelog.json) in JSON format:

###### Main changelog file format:

```
{
	"connection":"mongodb://username:password@mymongodb:27017/myauthdb",
	"dbname":"mydbname",
	"migrations": [
		{
			"include": "some-folder/changelog.json",
			"relativeToChangelogFile": true
		}
	]
}
```

* **connection** - **Required**. Full connection string to your MongoDB database
* **dbname** - **Required**. Database name inside MongoDB to which migrations will be applied
* **migrations** - **Required**. List of migrations to apply. Several formats are acceptible (see below)
* **include** - **Required**. Path to migration changelog file. Full or relative to this changelog file, according to `relativeToChangelogFile`
* **relativeToChangelogFile** - *Optional*. Indicates whether `include` path should be treated as relative to this changelog file. Default: `true`

Following formats are acceptable:

* Migrations is of type string. `relativeToChangelogFile` is true, by default.
```
{
	"connection":"mongodb://username:password@mymongodb:27017/myauthdb",
	"dbname":"mydbname",
	"migrations": "some-folder/changelog.json"
}
```

* Migrations is of type array of strings. `relativeToChangelogFile` is true, by default.
```
{
	"connection":"mongodb://username:password@mymongodb:27017/myauthdb",
	"dbname":"mydbname",
	"migrations": ["some-folder/changelog.json", "another-folder/changelog.json"]
}
```

* Full format, ommiting `relativeToChangelogFile`.
```
{
	"connection":"mongodb://username:password@mymongodb:27017/myauthdb",
	"dbname":"mydbname",
	"migrations": [
		{
			"include": "some-folder/changelog.json"
		}
	]
}
```

* Full format.
```
{
	"connection":"mongodb://username:password@mymongodb:27017/myauthdb",
	"dbname":"mydbname",
	"migrations": [
		{
			"include": "some-folder/changelog.json",
			"relativeToChangelogFile": true
		}
	]
}
```

###### Migration changelog file format:

```
{
  "id": "migration_id",
  "changes": [
    {
      "migration": {
        "include": "00001_first_migration.json",
        "relativeToChangelogFile": true
      },
      "rollback": {
        "include": "00001_first_migration_rollback.json",
        "relativeToChangelogFile": true
      }
    }
  ]
}
```
* **id** - **required**. Migration's ID.
* **changes** - **required**. List of changes to apply. Contains an object with 2 fields `migration` - forward migration, that is applied by `migrate` command; `rollback` - backward migration, that is applied by `rollback` command.
* **migration** - **required**. Lists direct commands to apply during forward migration. Has the same format as `migrations` tag from main changelog file (see above).
* **rollback** - optional. Lists direct commands to apply during backward migration. Has the same format as `migration` tag.

Following formats are acceptable:

```
{
  "id": "20190101_00001_initial_migration",
  "changes": [
    {
      "migration": {
        "include": "00001_first_migration.json",
        "relativeToChangelogFile": true
      }
    }
  ]
}
```

```
{
  "id": "20190101_00001_initial_migration",
  "changes": [
    {
      "migration": {
        "include": "00001_first_migration.json",
      }
    }
  ]
}
```

```
{
  "id": "20190101_00001_initial_migration",
  "changes": [
    {
      "migration": "00001_first_migration.json"
    }
  ]
}
```

```
{
  "id": "20190101_00001_initial_migration",
  "changes": [
    {
      "migration": ["00001_first_migration.json", "00002_second_migration.json"]
    }
  ]
}
```

###### Migration file format:

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

Example (single command). Following migration creates collection `collection_name` with validator:
```
{
  "create": "collection_name",
  "validator":{
    "$jsonSchema":{
      "bsonType":"object",
      "required":["name"],
      "properties":{
        "name":{
          "bsonType":"string",
          "description":"must be a string and is required"
        }
      }
    }
  }
}
```

Example (several commands). Following migration creates collection `collection_name` with validator and fills it up with test value:

```
{
  "cmds" [
    {
      "create": "collection_name",
      "validator":{
        "$jsonSchema":{
          "bsonType":"object",
          "required":["name"],
          "properties":{
            "name":{
              "bsonType":"string",
              "description":"must be a string and is required"
            }
          }
        }
      }
    },
    {
      "insert": "collection_name",
      "documents":[
        {
          "name": "Test value",
          "bsonType":"object",
          "required":["name"],
          "properties":{
            "name":{
              "bsonType":"string",
              "description":"must be a string and is required"
            }
          }
        }
      ]
    }
  ]
}
```

## Example
Can be found [here](https://github.com/coldze/mongol/tree/master/test) 


## Run

### Docker-way:

* forward migrations
```
docker run --rm -v /path/to/src:/mongol/src coldze/mongol:latest mongol migrate --path=/mongol/src/changelog.json 
```

* backward migrations
```
docker run --rm -v /path/to/src:/mongol/src coldze/mongol:latest mongol rollback --path=/mongol/src/changelog.json 
```

### Compile from source code & run:
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

* MongoDB supports JavaScript starting from 3.0 up to 3.6, in 4.0 it was deprecated, so you're able to use `eval` in migrations for MongoDB 3.x versions.

## Sample

##### Folder structure

```
./changelog.json //contains information about connection to mongo, includes sub-changelogs
./migrations/20190101/changelog.json //sub-changelog with migrations created at 2019-01-01
./migrations/20190101/0001_fill_data.json
./migrations/20190101/0001_fill_data_rollback.json
```  

##### Migration file
##### Changelog file
##### Root changelog file

## Running a test

Change mongo-db settings in `$GOPATH/src/github.com/coldze/mongol/test/changelog.json` and execute:
```
mongol migrate --path=$GOPATH/src/github.com/coldze/mongol/test/changelog.json
```

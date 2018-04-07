package main

import (
	"bitbucket.org/4fit/mongol/common/logs"
	"bitbucket.org/4fit/mongol/decoding_2"
	"bitbucket.org/4fit/mongol/migrations"
	"bitbucket.org/4fit/mongol/primitives/custom_error"
	"context"
	"flag"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"io/ioutil"
	"reflect"
	"strings"
)

/*type Test struct {
	Value1 string `bson:""`
}*/

const (
	ADMIN_COMMAND_CREATE = "create"
)

func main() {
	log := logs.NewStdLogger()

	changelogConfigPath := flag.String("changelog", "./changelog.json", "Path to root changelog file")
	flag.Parse()

	changeLog, errValue := migrations.NewChangeLog(changelogConfigPath)
	if errValue != nil {
		log.Fatalf("Failed to load changelog. Error: %v", errValue)
		return
	}
	mongoClient, err := mongo.NewClient(changeLog.Connection)
	if err != nil {
		log.Fatalf("Failed to connect to mongo. Error: %v", custom_error.MakeError(err))
		return
	}
	db := mongoClient.Database(changeLog.DbName)

	els := []*bson.Value{}
	els = append(els, bson.VC.String("test01"), bson.VC.String("test02"))
	erv := bson.VC.ArrayFromValues(els...)
	consV := bson.EC.Interface("test", erv)
	log.Infof("%+v", consV.String())

	decoding_2.Decode(nil)
	inputFile := flag.String("input", "./test.json", "input file")
	flag.Parse()
	/*mongoClient, err := mongo.NewClient("mongodb://localhost:27030")
	if err != nil {
		log.Fatalf("Failed to connect to mongo. Error: %v", err)
		return
	}*/
	//db := mongoClient.Database("mongol")

	log.Debugf("DB Name: %v", db.Name())
	_, err = db.RunCommand(context.Background(), bson.NewDocument(
		bson.EC.String("drop", "std2"),
	))
	if err != nil {
		log.Warningf("Failed to drop collection. Error: %v", err)
	}
	_, err = db.RunCommand(context.Background(), bson.NewDocument(
		bson.EC.String("drop", "std2\""),
	))
	if err != nil {
		log.Warningf("Failed to drop collection. Error: %v", err)
	}
	_, err = db.RunCommand(context.Background(), bson.NewDocument(
		bson.EC.String("drop", "std3"),
	))
	if err != nil {
		log.Warningf("Failed to drop collection. Error: %v", err)
	}

	parsedInput := map[string]interface{}{}
	inputData, err := ioutil.ReadFile(*inputFile)
	if err != nil {
		log.Fatalf("Failed to open file. Error: %v", err)
		return
	}

	parsedDoc, errValue := decoding_2.Decode(inputData)
	if errValue != nil {
		log.Fatalf("Failed to parse. Error: %v", errValue)
		return
	}
	if parsedDoc.Type() != bson.TypeEmbeddedDocument {
		log.Fatalf("Parsed document has unexpected type. %v", parsedDoc.Type())
		return
	}
	log.Infof("PARSED!")

	//err = json.Unmarshal([]byte(inputData), &parsedInput)

	/*doc, errC := parsing.GenerateCommand(parsedInput)
	if errC != nil {
		log.Fatalf("Error: %v", errC)
	}
	log.Infof("Done: %v", doc.String())

	_, err = db.RunCommand(context.Background(), doc)
	if err != nil {
		log.Fatalf("Failed to run-command (gen). Error: %v", err)
		return
	}*/
	log.Infof("Completed")

	_, err = db.RunCommand(context.Background(), parsedDoc.MutableDocument())
	if err != nil {
		log.Fatalf("Failed to run-command. Error: %v", err)
		return
	}
	log.Infof("Completed")

	/*if err != nil {
		log.Fatalf("Failed to unmarshal. Error: %v", err)
		return
	}

	for k, v := range parsedInput {
		if v == nil {
			log.Infof("%v has type 'null'", k)
			continue
		}
		log.Infof("%v has type '%v'", k, reflect.TypeOf(v).String())
	}
	data, _ := json.Marshal(parsedInput)
	log.Infof("Marshaled: %v", string(data))*/

	return

	param, ok := parsedInput[ADMIN_COMMAND_CREATE]
	if !ok {
		log.Fatalf("Unknown command")
		return
	}
	if reflect.TypeOf(param).Name() != "string" {
		typeValue := reflect.TypeOf(param)
		log.Fatalf("Invalid type for command %v. Expected string, got %v", ADMIN_COMMAND_CREATE, typeValue.Name())
	}
	params, ok := parsedInput["parameters"]
	if !ok {
		return
	}
	log.Infof("TYPE contains: %v. type name: %v", strings.Contains(reflect.TypeOf(params).String(), "map["), reflect.TypeOf(params).String())
	/*
			db.createCollection("students", {
		   validator: {
		      $jsonSchema: {
		         bsonType: "object",
		         required: [ "name"],
		         properties: {
		            name: {
		               bsonType: "string",
		               description: "must be a string and is required"
		            }
		         }
		      }
		   }
		})
	*/

	/*doc := bson.NewDocument(bson.EC.String("v1", "value"))
	data, err := doc.MarshalBSON()
	if err != nil {
		log.Fatalf("Failed to marshal document. Error: %v", err)
		return
	}
	log.Debugf(string(data))*/
	/*mongoSession, err := mgo.Dial("localhost:27030")
	if err != nil {
		log.Fatalf("Failed to dial mongo. Error: %v", err)
		return
	}
	defer mongoSession.Close()*/
}

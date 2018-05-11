package main

import (
	"context"
	"flag"
	"github.com/coldze/mongol/common/logs"
	"github.com/coldze/mongol/engine"
	"github.com/coldze/mongol/primitives/custom_error"
	mongo2 "github.com/coldze/mongol/primitives/mongo"
	"github.com/mongodb/mongo-go-driver/mongo"
)

func main() {
	log := logs.NewStdLogger()

	changelogConfigPath := flag.String("changelog", "./changelog.json", "Path to root changelog file")
	flag.Parse()

	changeLog, errValue := engine.NewChangeLog(changelogConfigPath)
	if errValue != nil {
		log.Fatalf("Failed to load changelog. Error: %v", errValue)
		return
	}
	ctx := context.Background()
	mongoClient, err := mongo.Connect(ctx, changeLog.GetConnectionString(), nil)
	if err != nil {
		log.Fatalf("Failed to connect to mongo. Error: %v", custom_error.MakeError(err))
		return
	}
	defer mongoClient.Disconnect(ctx)
	db := mongoClient.Database(changeLog.GetDBName())

	validator, errValue := mongo2.NewMongoChangeSetValidator(db, engine.COLLECTION_NAME_MIGRATIONS_LOG)
	if errValue != nil {
		log.Fatalf("Failed to create validator. Error: %v", errValue)
		return
	}

	errValue = changeLog.Apply(validator)
	if errValue != nil {
		log.Fatalf("Failed to validate changelog. Error: %v", errValue)
		return
	}

	documentApplier := mongo2.NewDbChanger(db, context.Background())
	transactionRecFactory := engine.NewTransactionRecordFactory(engine.COLLECTION_NAME_MIGRATIONS_LOG)

	transactionFactory, errValue := engine.NewSimulatedTransactionFactory(documentApplier, transactionRecFactory, log)
	if errValue != nil {
		log.Fatalf("Failed to create transaction factory. Error: %v", errValue)
		return
	}
	if transactionFactory == nil {
		log.Fatalf("Empty Transaction-factory created.")
		return
	}
	applier, errValue := engine.NewChangeSetApplier(transactionFactory)
	if errValue != nil {
		log.Fatalf("Failed to create migration applier. Error: %v", errValue)
		return
	}
	if applier == nil {
		log.Fatalf("Empty migration applier created.")
		return
	}

	log.Infof("Length: %v", len(changeLog.GetChangeSets()))

	errValue = changeLog.Apply(applier)

	if errValue != nil {
		log.Fatalf("Failed to apply changes. Error: %v", errValue)
		return
	}
}

package commands

import (
	"context"

	"github.com/coldze/mongol/engine"
	"github.com/coldze/mongol/primitives/mongo"
	"github.com/coldze/primitives/custom_error"
	"github.com/coldze/primitives/logs"
	mgo "github.com/mongodb/mongo-go-driver/mongo"
)

func Migrate(path string, log logs.Logger) custom_error.CustomError {
	changeLog, errValue := engine.NewChangeLog(path)
	if errValue != nil {
		return custom_error.NewErrorf(errValue, "Failed to load changelog.")
	}
	ctx := context.Background()
	mongoClient, err := mgo.Connect(ctx, changeLog.GetConnectionString(), nil)
	if err != nil {
		return custom_error.MakeErrorf("Failed to connect to mongo. Error: %v", err)
	}
	defer mongoClient.Disconnect(ctx)
	db := mongoClient.Database(changeLog.GetDBName())

	notAppliedList := []*engine.ChangeSet{}

	notAppliedProcessing := func(change *engine.ChangeSet) custom_error.CustomError {
		notAppliedList = append(notAppliedList, change)
		return nil
	}

	appliedProcessing := func(change *engine.ChangeSet) custom_error.CustomError {
		length := len(notAppliedList)
		if length > 0 {
			return custom_error.MakeErrorf("Applied change-set with ID %v after non-applied with ID %v", change.ID, notAppliedList[length-1].ID)
		}
		log.Infof("Already applied change-set with ID: %v", change.ID)
		return nil
	}

	validator, errValue := mongo.NewMongoChangeSetValidator(db, engine.COLLECTION_NAME_MIGRATIONS_LOG, appliedProcessing, notAppliedProcessing)
	if errValue != nil {
		return custom_error.NewErrorf(errValue, "Failed to create validator.")
	}

	errValue = changeLog.Apply(validator)
	if errValue != nil {
		return custom_error.NewErrorf(errValue, "Changelog validation failed.")
	}

	documentApplier := mongo.NewDbChanger(db, context.Background())
	transactionRecFactory := engine.NewTransactionRecordFactory(engine.COLLECTION_NAME_MIGRATIONS_LOG)

	transactionFactory, errValue := engine.NewSimulatedTransactionFactory(documentApplier, transactionRecFactory, log)
	if errValue != nil {
		return custom_error.NewErrorf(errValue, "Failed to create transaction factory.")
	}
	if transactionFactory == nil {
		return custom_error.MakeErrorf("Empty Transaction-factory created.")
	}
	applier, errValue := engine.NewChangeSetApplier(transactionFactory)
	if errValue != nil {
		return custom_error.NewErrorf(errValue, "Failed to create migration applier.")
	}
	if applier == nil {
		return custom_error.MakeErrorf("Empty Migration-applier created.")
	}

	notAppliedChangeLog, customErr := engine.NewArrayChangeLog(notAppliedList)
	if customErr != nil {
		return custom_error.NewErrorf(customErr, "Failed to create not-applied-change-log.")
	}

	errValue = notAppliedChangeLog.Apply(applier)
	if errValue != nil {
		return custom_error.NewErrorf(errValue, "Failed to apply changes.")
	}
	return nil
}

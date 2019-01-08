package commands

import (
	"context"

	"github.com/coldze/mongol/engine"
	"github.com/coldze/mongol/primitives/mongo"
	"github.com/coldze/primitives/custom_error"
	"github.com/coldze/primitives/logs"
	mgo "github.com/mongodb/mongo-go-driver/mongo"
)

func Rollback(path string, limit int64, log logs.Logger) custom_error.CustomError {
	changeLog, errValue := engine.NewRollbackChangeLog(path)
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

	notAppliedList := map[string]struct{}{}
	appliedList := map[string]struct{}{}

	notAppliedProcessing := func(changeID string) custom_error.CustomError {
		_, ok := notAppliedList[changeID]
		if ok {
			return custom_error.MakeErrorf("Dublicated change id: %v", changeID)
		}
		_, ok = appliedList[changeID]
		if ok {
			return custom_error.MakeErrorf("Dublicated change id: %v", changeID)
		}
		notAppliedList[changeID] = struct{}{}
		return nil
	}

	appliedProcessing := func(changeID string) custom_error.CustomError {
		log.Infof("Already applied change with ID: %v", changeID)
		_, ok := notAppliedList[changeID]
		if ok {
			return custom_error.MakeErrorf("Dublicated change id: %v", changeID)
		}
		_, ok = appliedList[changeID]
		if ok {
			return custom_error.MakeErrorf("Dublicated change id: %v", changeID)
		}
		appliedList[changeID] = struct{}{}
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
	transactionRecFactory := engine.NewRollbackTransactionRecordFactory(engine.COLLECTION_NAME_MIGRATIONS_LOG)

	transactionFactory, errValue := engine.NewRollbackSimulatedTransactionFactory(documentApplier, transactionRecFactory, notAppliedList, limit, log)
	if errValue != nil {
		return custom_error.NewErrorf(errValue, "Failed to create transaction factory.")
	}
	if transactionFactory == nil {
		return custom_error.MakeErrorf("Empty Transaction-factory created.")
	}
	applier, errValue := engine.NewRollbackChangeSetApplier(transactionFactory)
	if errValue != nil {
		return custom_error.NewErrorf(errValue, "Failed to create migration applier.")
	}
	if applier == nil {
		return custom_error.MakeErrorf("Empty Migration-applier created.")
	}

	/*notAppliedChangeLog, customErr := engine.NewArrayChangeLog(notAppliedList)
	if customErr != nil {
		return custom_error.NewErrorf(customErr, "Failed to create not-applied-change-log.")
	}*/

	errValue = changeLog.Apply(applier)
	if errValue != nil {
		return custom_error.NewErrorf(errValue, "Failed to apply changes.")
	}
	return nil
}

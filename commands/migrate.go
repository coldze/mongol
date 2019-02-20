package commands

import (
	"context"

	"github.com/coldze/mongol/engine"
	"github.com/coldze/mongol/primitives/mongo"
	"github.com/coldze/primitives/custom_error"
	"github.com/coldze/primitives/logs"
)

func Migrate(path string, limit int64, log logs.Logger) custom_error.CustomError {
	changeLog, errValue := engine.NewChangeLog(path)
	if errValue != nil {
		return custom_error.NewErrorf(errValue, "Failed to load changelog.")
	}
	ctx := context.Background()

	mongoClient, err := newMgoClient(ctx, changeLog.GetConnectionString())
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to connect to mongo.")
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
		length := len(notAppliedList)
		if length > 0 {
			return custom_error.MakeErrorf("Applied changeSet-set with ID %v after not-applied changes", changeID)
		}
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
	transactionRecFactory := engine.NewTransactionRecordFactory(engine.COLLECTION_NAME_MIGRATIONS_LOG)

	transactionFactory, errValue := engine.NewSimulatedTransactionFactory(documentApplier, transactionRecFactory, appliedList, limit, log)
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

	errValue = changeLog.Apply(applier)
	if errValue != nil {
		return custom_error.NewErrorf(errValue, "Failed to apply changes.")
	}
	return nil
}

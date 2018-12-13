package engine

import (
	"github.com/coldze/primitives/custom_error"
	"log"
	"runtime/debug"
)

type changeSetApplierStrategy func(changes []*Change, transaction Transaction) custom_error.CustomError

type changeSetApplier struct {
	startTransaction TransactionFactory
	strategy         changeSetApplierStrategy
}

func extractError(r interface{}, changeID string) custom_error.CustomError {
	errValue, ok := r.(custom_error.CustomError)
	if ok {
		return errValue
	}
	err, ok := r.(error)
	if ok {
		return custom_error.MakeErrorf("Failed to apply changeset '%v'. Error: %v", changeID, err)
	}
	return custom_error.MakeErrorf("Failed to apply changeset '%v'. Unknown error: %+v", changeID, r)
}

func forwardChangeSetApplierStrategy(changes []*Change, transaction Transaction) custom_error.CustomError {
	for i := range changes {
		err := transaction.Apply(changes[i])
		if err != nil {
			return custom_error.NewErrorf(err, "Failed to apply change '%v'.", changes[i].ID)
		}
	}
	return nil
}

func backwardChangeSetApplierStrategy(changes []*Change, transaction Transaction) custom_error.CustomError {
	for i := len(changes) - 1; i >= 0; i-- {
		err := transaction.Apply(changes[i])
		if err != nil {
			return custom_error.NewErrorf(err, "Failed to apply change '%v'.", changes[i].ID)
		}
	}
	return nil
}

func (c *changeSetApplier) Process(changeSet *ChangeSet) (result custom_error.CustomError) {
	if changeSet == nil {
		return custom_error.MakeErrorf("Failed to apply changeset. Nil pointer provided.")
	}
	transaction, err := c.startTransaction(changeSet.ID)
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to start transaction")
	}
	defer func() {
		if result == nil {
			commitErr := transaction.Commit()
			if commitErr == nil {
				return
			}
		}
		log.Printf("Error occured. Rolling back. Error: %+v", result)
		rollbackErr := transaction.Rollback()
		if rollbackErr != nil {
			result = custom_error.NewErrorf(rollbackErr, "Failed to rollback changeset with ID '%v' after error during application. Application error: %v", result)
		}
	}()

	defer func() {
		r := recover()
		if r != nil {
			debug.PrintStack()
			result = extractError(r, changeSet.ID)
			return
		}
	}()
	err = c.strategy(changeSet.Changes, transaction)
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to apply change-set '%v'.", changeSet.ID)
	}
	return nil
	/*res := c.migrations.FindOne(context.Background(), bson.NewDocument(bson.EC.String("change_set_id", changeSet.ID)))
	if res == nil {
		return nil
	}
	changeSetRecord := ChangeSetRecord{}
	err := res.Decode(&changeSetRecord)
	if err != nil {
		return custom_error.MakeErrorf("Failed to get changeset. Error: %v", err)
	}
	if changeSetRecord.Hash == changeSet.Hash {
		return nil
	}
	return custom_error.MakeErrorf("Checksum failed for changeset '%v'. Was: %v Now: %v", changeSet.ID, changeSetRecord.Hash, changeSet.Hash)*/

}

func NewChangeSetApplier(startTransaction TransactionFactory) (ChangeSetProcessor, custom_error.CustomError) {
	return &changeSetApplier{
		startTransaction: startTransaction,
		strategy:         forwardChangeSetApplierStrategy,
	}, nil
}

func NewRollbackChangeSetApplier(startTransaction TransactionFactory) (ChangeSetProcessor, custom_error.CustomError) {
	return &changeSetApplier{
		startTransaction: startTransaction,
		strategy:         backwardChangeSetApplierStrategy,
	}, nil
}

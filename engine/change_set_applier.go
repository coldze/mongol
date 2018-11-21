package engine

import (
	"github.com/coldze/primitives/custom_error"
)

type changeSetApplier struct {
	startTransaction TransactionFactory
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
		rollbackErr := transaction.Rollback()
		if rollbackErr != nil {
			result = custom_error.NewErrorf(rollbackErr, "Failed to rollback changeset with ID '%v' after error during application. Application error: %v", result)
		}
	}()

	defer func() {
		r := recover()
		if r != nil {
			result = extractError(r, changeSet.ID)
			return
		}
	}()
	for i := range changeSet.Changes {
		err := transaction.Apply(changeSet.Changes[i])
		if err != nil {
			panic(custom_error.NewErrorf(err, "Failed to apply change '%v'. Change #: %v", changeSet.ID, i+1))
		}
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
	}, nil
}

package mongo

import (
	"bitbucket.org/4fit/mongol/migrations"
	"bitbucket.org/4fit/mongol/primitives/custom_error"
)

type changeSetApplier struct {
	startTransaction TransactionFactory
}

func (c *changeSetApplier) applyChangeFromFile(filename string) custom_error.CustomError {
	return custom_error.MakeErrorf("Not implemented")
}

func (c *changeSetApplier) Process(changeSet *migrations.ChangeSet) (result custom_error.CustomError) {
	if changeSet == nil {
		return custom_error.MakeErrorf("Failed to apply changeset. Nil pointer provided.")
	}
	transaction, err := c.startTransaction(changeSet.ID, changeSet.Hash)
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to start transaction")
	}
	defer func() {
		r := recover()
		if r == nil {
			transaction.Commit()
			return
		}
		transaction.Rollback()
		errValue, ok := r.(custom_error.CustomError)
		if ok {
			result = errValue
			return
		}
		err, ok := r.(error)
		if ok {
			result = custom_error.MakeErrorf("Failed to apply changeset '%v'. Error: %v", changeSet.ID, err)
			return
		}
		result = custom_error.MakeErrorf("Failed to apply changeset '%v'. Unknown error: %+v", changeSet.ID, r)
	}()
	for i := range changeSet.Changes {
		err := transaction.Apply(changeSet.Changes[i])
		if err != nil {
			panic(custom_error.NewErrorf(err, "Failed to apply change '%v'", changeSet.Changes[i]))
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

func NewMongoChangeSetApplier(startTransaction TransactionFactory) (migrations.ChangeSetProcessor, custom_error.CustomError) {
	return &changeSetApplier{
		startTransaction: startTransaction,
	}, nil
}

package engine

import (
	"github.com/coldze/primitives/custom_error"
	"github.com/coldze/primitives/logs"
)

type SimulatedTransaction struct {
	log                     logs.Logger
	dbChanger               DocumentApplier
	rollbacks               []Migration
	changeID                string
	appliedChanges          map[string]struct{}
	createTransactionRecord TransactionRecordFactory
	changeCount             int64
}

func (t *SimulatedTransaction) Commit() custom_error.CustomError {
	/*transactionRecord, err := t.createTransactionRecord(t.changeID, t.hashValue)
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to create transaction record. Change ID: %v. Hash: %v", t.changeID, t.hashValue)
	}
	err = t.dbChanger.Apply(transactionRecord)
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to save transaction record. Change ID: %v. Hash: %v", t.changeID, t.hashValue)
	}*/
	t.log.Infof("Transaction commit")
	return nil
}

func (t *SimulatedTransaction) Apply(change *Change) custom_error.CustomError {
	_, applied := t.appliedChanges[change.ID]
	if applied {
		t.log.Infof("Skipping already applied change with id: %v", change.ID)
		return nil
	}
	t.log.Infof("Applying change: %v.", change.ID)
	err := change.Forward.Apply(t.dbChanger)
	if err != nil {
		return err
	}

	/*** Into Migration.Apply ***/
	appliedMigrationRecord, err := t.createTransactionRecord(change.ID, change.Hash)
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to create transaction record. Change ID: %v. Hash: %v. Migration index: %v", t.changeID, change.Hash, t.changeCount)
	}
	t.dbChanger.Apply(appliedMigrationRecord)
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to save migration record. Change ID: %v. Hash: %v. Migration index: %v", t.changeID, change.Hash, t.changeCount)
	}
	/*** Into Migration.Apply ^^^^^ ***/

	t.rollbacks = append(t.rollbacks, change.Backward)
	t.changeCount++
	return nil
}

func (t *SimulatedTransaction) Rollback() custom_error.CustomError {
	t.log.Infof("Transaction rollback")
	for i := len(t.rollbacks) - 1; i >= 0; i-- {
		err := t.rollbacks[i].Apply(t.dbChanger)
		if err != nil {
			return custom_error.MakeErrorf("Failed to rollback. Error: %v", err)
		}
	}
	return nil
}

func NewSimulatedTransactionFactory(dbChanger DocumentApplier, transactionRecFactory TransactionRecordFactory, appliedChanges map[string]struct{}, log logs.Logger) (TransactionFactory, custom_error.CustomError) {
	return func(changeID string) (Transaction, custom_error.CustomError) {
		return &SimulatedTransaction{
			changeID:                changeID,
			log:                     log,
			changeCount:             int64(0),
			dbChanger:               dbChanger,
			rollbacks:               []Migration{},
			appliedChanges:          appliedChanges,
			createTransactionRecord: transactionRecFactory,
		}, nil
	}, nil
}

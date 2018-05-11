package engine

import (
	"github.com/coldze/mongol/common/logs"
	"github.com/coldze/primitives/custom_error"
)

type SimulatedTransaction struct {
	log                     logs.Logger
	dbChanger               DocumentApplier
	rollbacks               []Migration
	changeID                string
	hashValue               string
	createTransactionRecord TransactionRecordFactory
}

func (t *SimulatedTransaction) Commit() custom_error.CustomError {
	transactionRecord, err := t.createTransactionRecord(t.changeID, t.hashValue)
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to create transaction record. Change ID: %v. Hash: %v", t.changeID, t.hashValue)
	}
	err = t.dbChanger.Apply(transactionRecord)
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to save transaction record. Change ID: %v. Hash: %v", t.changeID, t.hashValue)
	}
	t.log.Infof("Transaction commit")
	return nil
}

func (t *SimulatedTransaction) Apply(change *Change) custom_error.CustomError {
	t.log.Infof("Applying change.")
	err := change.Forward.Apply(t.dbChanger)
	if err != nil {
		return err
	}
	t.rollbacks = append(t.rollbacks, change.Backward)
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

func NewSimulatedTransactionFactory(dbChanger DocumentApplier, transactionRecFactory TransactionRecordFactory, log logs.Logger) (TransactionFactory, custom_error.CustomError) {
	return func(changeID string, hashValue string) (Transaction, custom_error.CustomError) {
		return &SimulatedTransaction{
			changeID:                changeID,
			hashValue:               hashValue,
			log:                     log,
			dbChanger:               dbChanger,
			rollbacks:               []Migration{},
			createTransactionRecord: transactionRecFactory,
		}, nil
	}, nil
}

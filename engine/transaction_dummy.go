package engine

import (
	"bitbucket.org/4fit/mongol/common/logs"
	"bitbucket.org/4fit/mongol/primitives/custom_error"
)

type transactionDummy struct {
	log logs.Logger
}

func (t *transactionDummy) Commit() custom_error.CustomError {
	t.log.Infof("Transaction commit")
	return nil
}

func (t *transactionDummy) Apply(change *Change) custom_error.CustomError {
	t.log.Infof("Applying change.")
	return nil
}

func (t *transactionDummy) Rollback() custom_error.CustomError {
	t.log.Infof("Transaction rollback")
	return nil
}

func NewDummyTransactionFactory(log logs.Logger) TransactionFactory {
	return func(changeID string, hashValue string) (Transaction, custom_error.CustomError) {
		return &transactionDummy{
			log: log,
		}, nil
	}
}

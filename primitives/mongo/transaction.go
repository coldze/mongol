package mongo

import (
	"bitbucket.org/4fit/mongol/common/logs"
	"bitbucket.org/4fit/mongol/primitives/custom_error"
	mgo "github.com/coldze/mongo-go-driver/mongo"
)

type Transaction interface {
	Commit()
	Rollback()
}

type TransactionFactory func() (Transaction, custom_error.CustomError)

type transactionDummy struct {
	log logs.Logger
}

func (t *transactionDummy) Commit() {
	t.log.Infof("Transaction commit")
}

func (t *transactionDummy) Rollback() {
	t.log.Infof("Transaction rollback")
}

func NewTransactionFactory(db *mgo.Database, log logs.Logger) (TransactionFactory, custom_error.CustomError) {
	return func() (Transaction, custom_error.CustomError) {
		return &transactionDummy{
			log: log,
		}, nil
	}, nil
}

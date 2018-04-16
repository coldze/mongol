package mongo

import (
	"bitbucket.org/4fit/mongol/common/logs"
	"bitbucket.org/4fit/mongol/migrations"
	"bitbucket.org/4fit/mongol/primitives/custom_error"
	mgo "github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/bson"
	"context"
)

type Transaction interface {
	Commit()
	Apply(changeSet *migrations.Change) custom_error.CustomError
	Rollback() custom_error.CustomError
}

type TransactionFactory func() (Transaction, custom_error.CustomError)

type transactionDummy struct {
	log logs.Logger
}

func (t *transactionDummy) Commit() {
	t.log.Infof("Transaction commit")
}

func (t *transactionDummy) Apply(change *migrations.Change) custom_error.CustomError {
	t.log.Infof("Applying change.")
	return nil
}

func (t *transactionDummy) Rollback() custom_error.CustomError {
	t.log.Infof("Transaction rollback")
	return nil
}

type DbChanger struct {
	db *mgo.Database
	context context.Context
}

func (c *DbChanger) Apply(value *bson.Value) custom_error.CustomError {
	_, err := c.db.RunCommand(c.context, value.MutableDocument())
	if err != nil {
		return custom_error.MakeErrorf("Failed to apply change. Error: %v", err)
	}
	return nil
}

type SimulatedTransaction struct {
	log logs.Logger
	dbChanger migrations.DocumentApplier
	rollbacks []migrations.Migration
}

func (t *SimulatedTransaction) Commit() {
	t.log.Infof("Transaction commit")
}

func (t *SimulatedTransaction) Apply(change *migrations.Change) custom_error.CustomError {
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


func NewTransactionFactory(db *mgo.Database, context context.Context, log logs.Logger) (TransactionFactory, custom_error.CustomError) {
	dbChanger := &DbChanger{
		context: context,
		db: db,
	}
	return func() (Transaction, custom_error.CustomError) {
		return &SimulatedTransaction{
			log: log,
			dbChanger: dbChanger,
			rollbacks: []migrations.Migration{},
		}, nil
	}, nil
}

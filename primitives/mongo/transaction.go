package mongo

import (
	"bitbucket.org/4fit/mongol/common/logs"
	"bitbucket.org/4fit/mongol/migrations"
	"bitbucket.org/4fit/mongol/primitives/custom_error"
	mgo "github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/bson"
	"context"
	"log"
)

type Transaction interface {
	Commit()
	Apply(changeSet *migrations.Change) custom_error.CustomError
	Rollback() custom_error.CustomError
}

type TransactionFactory func(changeID string, hashValue string) (Transaction, custom_error.CustomError)

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

type WriteOperationsError struct {
	Errors interface{} `bson:"writeErrors,omitempty"`
}

func hasError(keys bson.Keys) bool {
	for i := range keys {
		if keys[i].Name == "writeErrors" {
			return true
		}
	}
	return false
}

func (c *DbChanger) Apply(value *bson.Value) custom_error.CustomError {
	log.Printf("Applying: %+v", value.MutableDocument().String())
	reader, err := c.db.RunCommand(c.context, value.MutableDocument())
	if err != nil {
		return custom_error.MakeErrorf("Failed to apply change. Error: %v", err)
	}
	log.Print("looking")
	keys, err := reader.Keys(true)
	if err != nil {
		return custom_error.MakeErrorf("Failed to get keys from response. Error: %v", err)
	}
	if !hasError(keys) {
		return nil
	}

	el, err := reader.Lookup("writeErrors")
	log.Printf("looking result: '%+v', %+v", el, err)
	if err != nil {
		return custom_error.MakeErrorf("Failed to get response. Error: %v", err)
	}
	if el != nil {
		return custom_error.MakeErrorf("Failed to apply. Error: %+v", el.String())
	}
	return nil
	/*iter, err := reader.Iterator()
	if err != nil {
		return custom_error.MakeErrorf("Failed to read response after applying a change. Error: %v", err)
	}
	for iter.Next() {
		err = iter.Err()
		if err != nil {
			return custom_error.MakeErrorf("Failed to read applied change result. Error: %v", err)
		}
		log.Printf("%+v. Value: %+v", iter.Element(), iter.Element().Value())
		if iter.Element().Value() == nil {
			continue
		}

		dataX, err := iter.Element().MarshalBSON()
		log.Printf("BSON-ed (pre): '%+v', %+v", string(dataX), dataX)

		data, err := bson.Marshal(iter.Element())
		if err != nil {
			return custom_error.MakeErrorf("Failed to read applied change result. Error: %v", err)
		}
		log.Printf("BSON-ed: %+v", string(data))
		errors := WriteOperationsError{}
		err = bson.Unmarshal(data, &errors)
		if err != nil {
			return custom_error.MakeErrorf("Failed to read applied change result. Error: %v", err)
		}
		log.Printf("BSON-ed: %+v, %+v", string(data), errors)
		if errors.Errors == nil {
			continue
		}
		log.Printf("Iter: %+v", iter.Element().String())
		return custom_error.MakeErrorf("Failed to apply changes. Error: %+v", errors.Errors)
	}*/
	return nil
}

type SimulatedTransaction struct {
	log logs.Logger
	dbChanger migrations.DocumentApplier
	rollbacks []migrations.Migration
	changeID string
	hashValue string
	migrations *mgo.Collection
}

func (t *SimulatedTransaction) Commit() {
	//t.migrations.InsertOne(t.context, document)
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
	migrationCollection := db.Collection(collection_name_migrations_log)
	if migrationCollection == nil {
		return nil, custom_error.MakeErrorf("Internal error. Nulled collection returned.")
	}
	dbChanger := &DbChanger{
		context: context,
		db: db,
	}
	return func(changeID string, hashValue string) (Transaction, custom_error.CustomError) {
		return &SimulatedTransaction{
			changeID: changeID,
			hashValue: hashValue,
			migrations: migrationCollection,
			log: log,
			dbChanger: dbChanger,
			rollbacks: []migrations.Migration{},
		}, nil
	}, nil
}

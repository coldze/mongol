package mongo

import (
	"context"

	"github.com/coldze/mongol/engine"
	"github.com/coldze/primitives/custom_error"
	mgo "github.com/mongodb/mongo-go-driver/mongo"
)

type DbChanger struct {
	db      *mgo.Database
	context context.Context
}

type WriteOperationsError struct {
	Errors interface{} `bson:"writeErrors,omitempty"`
}

func (c *DbChanger) Apply(value interface{}) custom_error.CustomError {
	res := c.db.RunCommand(c.context, value)
	if res.Err() != nil {
		return custom_error.MakeErrorf("Failed to apply change. Error: %v", res.Err())
	}
	return nil
	return nil
}

func NewDbChanger(db *mgo.Database, context context.Context) engine.DocumentApplier {
	dbChanger := &DbChanger{
		context: context,
		db:      db,
	}
	return dbChanger
}

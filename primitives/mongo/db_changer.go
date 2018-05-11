package mongo

import (
	"context"
	"github.com/coldze/mongol/engine"
	"github.com/coldze/primitives/custom_error"
	"github.com/mongodb/mongo-go-driver/bson"
	mgo "github.com/mongodb/mongo-go-driver/mongo"
)

type DbChanger struct {
	db      *mgo.Database
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
	reader, err := c.db.RunCommand(c.context, value.MutableDocument())
	if err != nil {
		return custom_error.MakeErrorf("Failed to apply change. Error: %v", err)
	}
	keys, err := reader.Keys(true)
	if err != nil {
		return custom_error.MakeErrorf("Failed to get keys from response. Error: %v", err)
	}
	if !hasError(keys) {
		return nil
	}

	el, err := reader.Lookup("writeErrors")
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

func NewDbChanger(db *mgo.Database, context context.Context) engine.DocumentApplier {
	/*migrationCollection := db.Collection(collection_name_migrations_log)
	if migrationCollection == nil {
		return nil, custom_error.MakeErrorf("Internal error. Nulled collection returned.")
	}*/
	dbChanger := &DbChanger{
		context: context,
		db:      db,
	}
	return dbChanger
}

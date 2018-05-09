package mongo

import (
	"bitbucket.org/4fit/mongol/engine"
	"bitbucket.org/4fit/mongol/primitives/custom_error"
	"context"
	"github.com/mongodb/mongo-go-driver/bson"
	mgo "github.com/mongodb/mongo-go-driver/mongo"
)

type changeSetValidator struct {
	migrations *mgo.Collection
}

type ChangeSetRecord struct {
	ID        string `bson:"change_set_id"`
	Hash      string `bson:"hash"`
	AppliedAt string `bson:"applied_at"`
}

func (c *changeSetValidator) Process(changeSet *engine.ChangeSet) custom_error.CustomError {
	if changeSet == nil {
		return custom_error.MakeErrorf("Failed to validate changeset. Nil pointer provided.")
	}
	res, err := c.migrations.Find(context.Background(), bson.NewDocument(bson.EC.String("change_set_id", changeSet.ID)))
	if err != nil {
		return custom_error.MakeErrorf("Failed to get changeset from DB. Error: %v", err)
	}
	if res == nil {
		return custom_error.MakeErrorf("Failed to get changeset from DB. Empty response cursor.")
	}
	for res.Next(context.Background()) {
		changeSetRecord := ChangeSetRecord{}
		err := res.Decode(&changeSetRecord)
		if err != nil {
			return custom_error.MakeErrorf("Failed to get changeset from DB. Error: %v", err)
		}
		if changeSetRecord.Hash != changeSet.Hash {
			return custom_error.MakeErrorf("Checksum failed for changeset '%v'. Was: %v Now: %v", changeSet.ID, changeSetRecord.Hash, changeSet.Hash)
		}
	}

	return nil
}

func NewMongoChangeSetValidator(db *mgo.Database, collectionName string) (engine.ChangeSetProcessor, custom_error.CustomError) {
	migrationCollection := db.Collection(collectionName)
	if migrationCollection == nil {
		return nil, custom_error.MakeErrorf("Internal error. Nulled collection returned.")
	}
	return &changeSetValidator{
		migrations: migrationCollection,
	}, nil
}

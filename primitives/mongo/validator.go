package mongo

import (
	"context"

	"github.com/coldze/mongol/engine"
	"github.com/coldze/primitives/custom_error"
	"github.com/mongodb/mongo-go-driver/bson"
	mgo "github.com/mongodb/mongo-go-driver/mongo"
)

type changeSetValidator struct {
	migrations *mgo.Collection
	applied    ChangeSetConsumer
	notApplied ChangeSetConsumer
}

type ChangeSetRecord struct {
	ID        string `bson:"change_set_id"`
	Hash      string `bson:"hash"`
	AppliedAt string `bson:"applied_at"`
}

type ChangeSetConsumer func(changeSet *engine.ChangeSet) custom_error.CustomError

func (c *changeSetValidator) Process(changeSet *engine.ChangeSet) custom_error.CustomError {
	if changeSet == nil {
		return custom_error.MakeErrorf("Failed to validate changeset. Nil pointer provided.")
	}
	res, err := c.migrations.Find(context.Background(), bson.NewDocument(bson.EC.String("change_id", changeSet.ID)))
	if err != nil {
		return custom_error.MakeErrorf("Failed to get changeset from DB. Error: %v", err)
	}
	if res == nil {
		return custom_error.MakeErrorf("Failed to get changeset from DB. Empty response cursor.")
	}
	found := false
	for res.Next(context.Background()) {
		changeSetRecord := ChangeSetRecord{}
		err := res.Decode(&changeSetRecord)
		if err != nil {
			return custom_error.MakeErrorf("Failed to get changeset from DB. Error: %v", err)
		}
		if changeSetRecord.Hash != changeSet.Hash {
			return custom_error.MakeErrorf("Checksum failed for changeset '%v'. Was: %v Now: %v", changeSet.ID, changeSetRecord.Hash, changeSet.Hash)
		}
		found = true
	}
	if found {
		customErr := c.applied(changeSet)
		if customErr != nil {
			return custom_error.NewErrorf(customErr, "Failed to send into applied change-set with ID %v", changeSet.ID)
		}
	} else {
		customErr := c.notApplied(changeSet)
		if customErr != nil {
			return custom_error.NewErrorf(customErr, "Failed to send into not-applied change-set with ID %v", changeSet.ID)
		}
	}
	return nil
}

func NewMongoChangeSetValidator(db *mgo.Database, collectionName string, appliedConsumer ChangeSetConsumer, notAppliedConsumer ChangeSetConsumer) (engine.ChangeSetProcessor, custom_error.CustomError) {
	migrationCollection := db.Collection(collectionName)
	if migrationCollection == nil {
		return nil, custom_error.MakeErrorf("Internal error. Nulled collection returned.")
	}
	return &changeSetValidator{
		migrations: migrationCollection,
		applied:    appliedConsumer,
		notApplied: notAppliedConsumer,
	}, nil
}

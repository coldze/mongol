package mongo

import (
	"context"

	"github.com/coldze/mongol/engine"
	"github.com/coldze/primitives/custom_error"
	mgo "go.mongodb.org/mongo-driver/mongo"
)

type changeSetValidator struct {
	migrations *mgo.Collection
	applied    ChangeSetConsumer
	notApplied ChangeSetConsumer
}

type ChangeRecord struct {
	ID        string `bson:"change_id"`
	Hash      string `bson:"hash"`
	AppliedAt string `bson:"applied_at"`
}

type ChangeSetConsumer func(changeID string) custom_error.CustomError

func (c *changeSetValidator) Process(changeSet *engine.ChangeSet) custom_error.CustomError {
	if changeSet == nil {
		return custom_error.MakeErrorf("Failed to validate changeset. Nil pointer provided.")
	}
	/*res, err := c.migrations.Find(context.Background(), map[string]interface{}{"change_id": changeSet.ID})
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
	}*/
	for _, change := range changeSet.Changes {

		applied, err := c.processChange(change)
		if err != nil {
			return custom_error.NewErrorf(err, "Failed to check change-set with ID %v", changeSet.ID)
		}
		if applied {
			customErr := c.applied(change.ID)
			if customErr != nil {
				return custom_error.NewErrorf(customErr, "Failed to send into applied change with ID %v", change.ID)
			}
		} else {
			customErr := c.notApplied(change.ID)
			if customErr != nil {
				return custom_error.NewErrorf(customErr, "Failed to send into not-applied change with ID %v", change.ID)
			}
		}
	}
	return nil
}

func (c *changeSetValidator) processChange(change *engine.Change) (bool, custom_error.CustomError) {
	if change == nil {
		return false, custom_error.MakeErrorf("Failed to validate change. Nil pointer provided.")
	}
	res, err := c.migrations.Find(context.Background(), map[string]interface{}{"change_id": change.ID})
	if err != nil {
		return false, custom_error.MakeErrorf("Failed to get change from DB. Error: %v", err)
	}
	if res == nil {
		return false, custom_error.MakeErrorf("Failed to get change from DB. Empty response cursor.")
	}
	found := false
	for res.Next(context.Background()) {
		changeRecord := ChangeRecord{}
		err := res.Decode(&changeRecord)
		if err != nil {
			return false, custom_error.MakeErrorf("Failed to get change from DB. Error: %v", err)
		}
		if changeRecord.Hash != change.Hash {
			return false, custom_error.MakeErrorf("Checksum failed for change '%v'. Was: %v Now: %v", change.ID, changeRecord.Hash, change.Hash)
		}
		found = true
	}
	/*if found {
		customErr := c.applied(change)
		if customErr != nil {
			return false, custom_error.NewErrorf(customErr, "Failed to send into applied change with ID %v", change.ID)
		}
	} else {
		customErr := c.notApplied(change)
		if customErr != nil {
			return false, custom_error.NewErrorf(customErr, "Failed to send into not-applied change with ID %v", change.ID)
		}
	}*/
	return found, nil
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

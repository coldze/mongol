package engine

import (
	"bitbucket.org/4fit/mongol/engine/decoding_2"
	"bitbucket.org/4fit/mongol/primitives/custom_error"
	"fmt"
	"github.com/mongodb/mongo-go-driver/bson"
	"time"
)

const (
	COLLECTION_NAME_MIGRATIONS_LOG = "mongol_migrations_3710611845fe4161b74d2ec5eafe9124"
)

type TransactionRecordFactory func(changeID string, hashValue string) (*bson.Value, custom_error.CustomError)

func NewTransactionRecordFactory(collectionName string) TransactionRecordFactory {
	format := fmt.Sprintf("{\"insert\": \"%s\", \"documents\":[{\"change_id\": \"%%s\", \"hash\":\"%%s\", \"applied_at_utc\": %%d }]}", collectionName)
	return func(changeID string, hashValue string) (*bson.Value, custom_error.CustomError) {
		data := fmt.Sprintf(format, changeID, hashValue, time.Now().UnixNano())
		v, err := decoding_2.Decode([]byte(data))
		if err != nil {
			return nil, custom_error.NewErrorf(err, "Failed to create transaction record. ChangeID: %v. Hash: %v", changeID, hashValue)
		}
		return v, nil
	}
}

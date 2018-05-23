package engine

import (
	"fmt"
	"time"

	"github.com/coldze/mongol/engine/decoding"
	"github.com/coldze/primitives/custom_error"
	"github.com/mongodb/mongo-go-driver/bson"
)

const (
	COLLECTION_NAME_MIGRATIONS_LOG = "mongol_migrations_3710611845fe4161b74d2ec5eafe9124"
	transaction_record_format      = "{\"insert\": \"%s\", \"documents\":[{\"change_id\": \"%%s\", \"hash\":\"%%s\", \"applied_at_utc\": %%d }]}"
)

type TransactionRecordFactory func(changeID string, hashValue string) (*bson.Value, custom_error.CustomError)

func NewTransactionRecordFactory(collectionName string) TransactionRecordFactory {
	format := fmt.Sprintf(transaction_record_format, collectionName)
	return func(changeID string, hashValue string) (*bson.Value, custom_error.CustomError) {
		data := fmt.Sprintf(format, changeID, hashValue, time.Now().UnixNano())
		v, err := decoding.Decode([]byte(data))
		if err != nil {
			return nil, custom_error.NewErrorf(err, "Failed to create transaction record. ChangeID: %v. Hash: %v", changeID, hashValue)
		}
		return v, nil
	}
}

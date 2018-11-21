package engine

import (
	"fmt"
	"time"

	"github.com/coldze/mongol/engine/decoding"
	"github.com/coldze/primitives/custom_error"
)

const (
	COLLECTION_NAME_MIGRATIONS_LOG   = "mongol_migrations_3710611845fe4161b74d2ec5eafe9124"
	transaction_add_record_format    = "{\"insert\": \"%s\", \"documents\":[{\"change_id\": \"%%s\", \"hash\":\"%%s\", \"applied_at_utc\": %%d }]}"
	transaction_remove_record_format = "{\"delete\": \"%s\", \"deletes\": [{\"q\": {\"change_id\": \"%%s\"}, \"limit\": 1}]}"
)

type TransactionRecordFactory func(changeID string, hashValue string) (interface{}, custom_error.CustomError)

func NewTransactionRecordFactory(collectionName string) TransactionRecordFactory {
	format := fmt.Sprintf(transaction_add_record_format, collectionName)
	return func(changeID string, hashValue string) (interface{}, custom_error.CustomError) {
		data := fmt.Sprintf(format, changeID, hashValue, time.Now().UnixNano())
		v, err := decoding.DecodeExt([]byte(data))
		if err != nil {
			return nil, custom_error.NewErrorf(err, "Failed to create transaction record. ChangeID: %v. Hash: %v", changeID, hashValue)
		}
		return v, nil
	}
}

func NewRollbackTransactionRecordFactory(collectionName string) TransactionRecordFactory {
	format := fmt.Sprintf(transaction_remove_record_format, collectionName)
	return func(changeID string, hashValue string) (interface{}, custom_error.CustomError) {
		data := fmt.Sprintf(format, changeID)
		v, err := decoding.DecodeExt([]byte(data))
		if err != nil {
			return nil, custom_error.NewErrorf(err, "Failed to create transaction record. ChangeID: %v. Hash: %v", changeID, hashValue)
		}
		return v, nil
	}
}

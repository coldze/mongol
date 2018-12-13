package engine

import (
	"fmt"
	"time"

	"github.com/coldze/mongol/engine/decoding"
	"github.com/coldze/primitives/custom_error"
)

const (
	COLLECTION_NAME_MIGRATIONS_LOG   = "mongol_migrations_3710611845fe4161b74d2ec5eafe9124"
	transaction_remove_record_format = "{\"delete\": \"%s\", \"deletes\": [{\"q\": {\"change_id\": \"%%s\"}, \"limit\": 1}]}"
)

type TransactionRecordFactory func(changeID string, hashValue string, rollback interface{}) (interface{}, custom_error.CustomError)

func NewTransactionRecordFactory(collectionName string) TransactionRecordFactory {
	return func(changeID string, hashValue string, rollback interface{}) (interface{}, custom_error.CustomError) {
		addTransaction := map[string]interface{}{}
		addTransaction["insert"] = collectionName
		docs := []interface{}{
			map[string]interface{}{
				"change_id":      changeID,
				"hash":           hashValue,
				"rollback":       rollback,
				"applied_at_utc": time.Now().UnixNano(),
			},
		}
		addTransaction["documents"] = docs
		return addTransaction, nil
	}
}

func NewRollbackTransactionRecordFactory(collectionName string) TransactionRecordFactory {
	format := fmt.Sprintf(transaction_remove_record_format, collectionName)
	return func(changeID string, hashValue string, rollback interface{}) (interface{}, custom_error.CustomError) {
		data := fmt.Sprintf(format, changeID)
		v, err := decoding.DecodeExt([]byte(data))
		if err != nil {
			return nil, custom_error.NewErrorf(err, "Failed to create transaction record. ChangeID: %v. Hash: %v", changeID, hashValue)
		}
		return v, nil
	}
}

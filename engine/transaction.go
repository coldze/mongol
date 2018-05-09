package engine

import (
	"bitbucket.org/4fit/mongol/primitives/custom_error"
)

type Transaction interface {
	Commit() custom_error.CustomError
	Apply(changeSet *Change) custom_error.CustomError
	Rollback() custom_error.CustomError
}

type TransactionFactory func(changeID string, hashValue string) (Transaction, custom_error.CustomError)

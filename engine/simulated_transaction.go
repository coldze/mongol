package engine

import (
	"github.com/coldze/primitives/custom_error"
	"github.com/coldze/primitives/logs"
)

type MigrationExtractor func(change *Change) Migration

type LimitCounter interface {
	Get() int64
	Dec()
}

type unsafeLimitCounter struct {
	value int64
}

func (u *unsafeLimitCounter) Get() int64 {
	return u.value
}

func (u *unsafeLimitCounter) Dec() {
	u.value--
}

type fakeLimitCounter struct {
}

func (u *fakeLimitCounter) Get() int64 {
	return 1
}

func (u *fakeLimitCounter) Dec() {
}

type limitedTransaction struct {
	log             logs.Logger
	migrationsLimit LimitCounter
	subTransaction  Transaction
}

func (t *limitedTransaction) Commit() custom_error.CustomError {
	err := t.subTransaction.Commit()
	if err != nil {
		return custom_error.NewErrorf(err, "Limited transaction failed to commit.")
	}
	return nil
}

func (t *limitedTransaction) Apply(change *Change) custom_error.CustomError {
	if t.migrationsLimit.Get() <= 0 {
		t.log.Infof("Not applying change, limit reached: %v.", change.ID)
		return nil
	}

	err := t.subTransaction.Apply(change)
	if err != nil {
		return custom_error.NewErrorf(err, "Limited transaction failed to apply.")
	}

	t.migrationsLimit.Dec()
	return nil
}

func (t *limitedTransaction) Rollback() custom_error.CustomError {
	err := t.subTransaction.Rollback()
	if err != nil {
		return custom_error.NewErrorf(err, "Limited transaction failed to rollback.")
	}
	return nil
}

type appliedCheckingTransaction struct {
	log            logs.Logger
	appliedChanges map[string]struct{}
	subTransaction Transaction
}

func (t *appliedCheckingTransaction) Commit() custom_error.CustomError {
	err := t.subTransaction.Commit()
	if err != nil {
		return custom_error.NewErrorf(err, "Applied-checking transaction failed to commit.")
	}
	return nil
}

func (t *appliedCheckingTransaction) Apply(change *Change) custom_error.CustomError {
	_, applied := t.appliedChanges[change.ID]
	if applied {
		t.log.Infof("Skipping already applied change with id: %v", change.ID)
		return nil
	}

	err := t.subTransaction.Apply(change)
	if err != nil {
		return custom_error.NewErrorf(err, "Applied-checking transaction failed to apply.")
	}

	return nil
}

func (t *appliedCheckingTransaction) Rollback() custom_error.CustomError {
	err := t.subTransaction.Rollback()
	if err != nil {
		return custom_error.NewErrorf(err, "Applied-checking transaction failed to rollback.")
	}
	return nil
}

type SimulatedTransaction struct {
	log                     logs.Logger
	dbChanger               DocumentApplier
	rollbacks               []Migration
	changeID                string
	createTransactionRecord TransactionRecordFactory
	newEncoder              EncoderFactory
	getMigrationToApply     MigrationExtractor
	getRollbackMigration    MigrationExtractor
}

func (t *SimulatedTransaction) Commit() custom_error.CustomError {
	/*transactionRecord, err := t.createTransactionRecord(t.changeID, t.hashValue)
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to create transaction record. Change ID: %v. Hash: %v", t.changeID, t.hashValue)
	}
	err = t.dbChanger.Apply(transactionRecord)
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to save transaction record. Change ID: %v. Hash: %v", t.changeID, t.hashValue)
	}*/
	t.log.Infof("Transaction commit")
	return nil
}

func (t *SimulatedTransaction) Apply(change *Change) custom_error.CustomError {
	t.log.Infof("Applying change: %v.", change.ID)

	err := t.getMigrationToApply(change).Apply(t.dbChanger)
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to apply migration.")
	}
	rollbackMigration := t.getRollbackMigration(change)
	encoder := t.newEncoder()
	err = rollbackMigration.Apply(encoder)
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to prepare rollback migration")
	}
	binRollback, err := encoder.Encode()
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to encode rollback migration")
	}

	/*** Into Migration.Apply ***/
	appliedMigrationRecord, err := t.createTransactionRecord(change.ID, change.Hash, binRollback)
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to create transaction record. Change ID: %v. Hash: %v. Change ID: %v", t.changeID, change.Hash, change.ID)
	}
	t.dbChanger.Apply(appliedMigrationRecord)
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to save migration record. Change ID: %v. Hash: %v. Change ID: %v", t.changeID, change.Hash, change.ID)
	}
	/*** Into Migration.Apply ^^^^^ ***/

	t.rollbacks = append(t.rollbacks, rollbackMigration)
	return nil
}

func (t *SimulatedTransaction) Rollback() custom_error.CustomError {
	t.log.Infof("Transaction rollback")
	for i := len(t.rollbacks) - 1; i >= 0; i-- {
		err := t.rollbacks[i].Apply(t.dbChanger)
		if err != nil {
			return custom_error.MakeErrorf("Failed to rollback. Error: %v", err)
		}
	}
	return nil
}

func getForwardMigration(change *Change) Migration {
	return change.Forward
}

func getBackwardMigration(change *Change) Migration {
	return change.Backward
}

func newLimitCounter(maxChanges int64) LimitCounter {
	if maxChanges > 0 {
		return &unsafeLimitCounter{
			maxChanges,
		}
	}
	return &fakeLimitCounter{}
}

func newWrappedTransaction(appliedChanges map[string]struct{}, maxChanges int64, log logs.Logger) (func(Transaction) Transaction, custom_error.CustomError) {
	limitCounter := newLimitCounter(maxChanges)
	return func(transaction Transaction) Transaction {
		nonAppliedTransaction := &appliedCheckingTransaction{
			log:            log,
			subTransaction: transaction,
			appliedChanges: appliedChanges,
		}
		return &limitedTransaction{
			log:             log,
			subTransaction:  nonAppliedTransaction,
			migrationsLimit: limitCounter,
		}
	}, nil
}

func NewSimulatedTransactionFactory(dbChanger DocumentApplier, transactionRecFactory TransactionRecordFactory, appliedChanges map[string]struct{}, maxChanges int64, log logs.Logger) (TransactionFactory, custom_error.CustomError) {
	transactionWrapper, err := newWrappedTransaction(appliedChanges, maxChanges, log)
	if err != nil {
		return nil, custom_error.NewErrorf(err, "Failed to create transaction wrap")
	}
	return func(changeID string) (Transaction, custom_error.CustomError) {
		simTransaction := &SimulatedTransaction{
			changeID:                changeID,
			log:                     log,
			dbChanger:               dbChanger,
			rollbacks:               []Migration{},
			createTransactionRecord: transactionRecFactory,
			newEncoder:              NewBinEncoder,
			getMigrationToApply:     getForwardMigration,
			getRollbackMigration:    getBackwardMigration,
		}
		return transactionWrapper(simTransaction), nil
	}, nil
}

func NewRollbackSimulatedTransactionFactory(dbChanger DocumentApplier, transactionRecFactory TransactionRecordFactory, appliedChanges map[string]struct{}, maxChanges int64, log logs.Logger) (TransactionFactory, custom_error.CustomError) {
	transactionWrapper, err := newWrappedTransaction(appliedChanges, maxChanges, log)
	if err != nil {
		return nil, custom_error.NewErrorf(err, "Failed to create transaction wrap")
	}
	return func(changeID string) (Transaction, custom_error.CustomError) {
		simTransaction := &SimulatedTransaction{
			changeID:                changeID,
			log:                     log,
			dbChanger:               dbChanger,
			rollbacks:               []Migration{},
			createTransactionRecord: transactionRecFactory,
			newEncoder:              NewBinEncoder,

			getMigrationToApply:  getBackwardMigration,
			getRollbackMigration: getForwardMigration,
		}
		return transactionWrapper(simTransaction), nil
	}, nil
}

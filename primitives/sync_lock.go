package primitives

type SyncLock interface {
	Unlock() error
}

package engine

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/coldze/mongol/engine/decoding"
	"github.com/coldze/primitives/custom_error"
)

type ChangeSetProcessor interface {
	Process(changeSet *ChangeSet) custom_error.CustomError
}

type ChangeSetSource interface {
	Apply(processor ChangeSetProcessor) custom_error.CustomError
}

type ChangeLog interface {
	GetConnectionString() string
	GetDBName() string
	GetChangeSets() []*ChangeSet
	GetChangeSetSource() ChangeSetSource
	Apply(processor ChangeSetProcessor) custom_error.CustomError
}

type MigrationFile struct {
	Path         string `json:"include,omitempty"`
	RelativePath bool   `json:"relativeToChangelogFile"`
}

func (m *MigrationFile) validate() custom_error.CustomError {
	if len(m.Path) <= 0 {
		return custom_error.MakeErrorf("MigrationFile format error: path is empty. Expected non-empty field 'path'")
	}
	return nil
}

func getFullPath(relativePath bool, workingDir string, changelogPath string, filePath string) string {
	if relativePath {
		return filepath.Join(changelogPath, filePath)
	}
	return filepath.Join(workingDir, filePath)
}

func NewMigration(m *MigrationFile, workingDir string, changelogPath string, hash hash.Hash) (Migration, custom_error.CustomError) {
	if m == nil {
		return &DummyMigration{}, nil
	}
	err := m.validate()
	if err != nil {
		return nil, custom_error.NewErrorf(err, "Failed to generate migration")
	}
	fullPath := getFullPath(m.RelativePath, workingDir, changelogPath, m.Path)
	migrationRawContent, ioErr := ioutil.ReadFile(fullPath)
	if ioErr != nil {
		return nil, custom_error.MakeErrorf("Failed to read file '%v'. Error: %v", m.Path, ioErr)
	}
	migrationContent, err := decoding.DecodeMigration(migrationRawContent)

	if err != nil {
		return nil, custom_error.MakeErrorf("Failed to generate migration from file '%v'. Error: %v", m.Path, err)
	}
	hash.Write(migrationRawContent)
	return &SimpleMigration{
		source:   m,
		commands: migrationContent,
	}, nil
}

func NewMigrationFromDB() (Migration, custom_error.CustomError) {
	return nil, custom_error.MakeErrorf("Not implemented")
}

type DocumentApplier interface {
	Apply(value interface{}) custom_error.CustomError
}

type Migration interface {
	Apply(visitor DocumentApplier) custom_error.CustomError
}

type DummyMigration struct {
}

func (d *DummyMigration) Apply(visitor DocumentApplier) custom_error.CustomError {
	return nil
}

type SimpleMigration struct {
	source   *MigrationFile
	commands []interface{}
}

func (s *SimpleMigration) Apply(visitor DocumentApplier) custom_error.CustomError {
	for i := range s.commands {
		err := visitor.Apply(s.commands[i])
		log.Printf("%+v, %T", s.commands[i], s.commands[i])
		if err != nil {
			return custom_error.NewErrorf(err, "Failed to apply command: %+v. Type: %T", s.commands[i], s.commands[i])
		}
	}
	return nil
}

type ChangeFile struct {
	Forward  MigrationFile  `json:"migration"`
	Backward *MigrationFile `json:"rollback,omitempty"`
}

func (c *ChangeFile) validate() custom_error.CustomError {
	err := c.Forward.validate()
	if err != nil {
		return err
	}
	if c.Backward == nil {
		return nil
	}
	return c.Backward.validate()
}

func NewChange(c *ChangeFile, workingDir string, changelogPath string, id string) (*Change, custom_error.CustomError) {
	changeHash := md5.New()
	forward, err := NewMigration(&c.Forward, workingDir, changelogPath, changeHash)
	if err != nil {
		return nil, custom_error.NewErrorf(err, "Failed to generate change. Forward migration generate process failed.")
	}
	backward, err := NewMigration(c.Backward, workingDir, changelogPath, changeHash)
	if err != nil {
		return nil, custom_error.NewErrorf(err, "Failed to generate change. Backward migration generate process failed.")
	}
	hashValue := hex.EncodeToString(changeHash.Sum(nil))
	return &Change{
		Backward: backward,
		Forward:  forward,
		Hash:     hashValue,
		ID:       id,
	}, nil
}

type ChangeSetFile struct {
	ID      string       `json:"id"`
	Changes []ChangeFile `json:"changes,omitempty"`
}
type Change struct {
	Forward  Migration
	Backward Migration
	Hash     string
	ID       string
}

type ChangeSet struct {
	ID      string
	Changes []*Change
}

type ChangeSetApplyStrategy func(sets []*ChangeSet, processor ChangeSetProcessor) custom_error.CustomError

type mainChangeLog struct {
	workingDir     string                 `json:"-"`
	Connection     string                 `json:"connection,omitempty"`
	DbName         string                 `json:"dbname,omitempty"`
	MigrationFiles []MigrationFile        `json:"migrations,omitempty"`
	changeSets     []*ChangeSet           `json:"-"`
	strategy       ChangeSetApplyStrategy `json:"-"`
}

func loadChangeSet(path string, workingDir string) (*ChangeSet, custom_error.CustomError) {
	changeSetData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, custom_error.MakeErrorf("Failed to load changeset from '%v'. Error: %v", path, err)
	}
	changeSetFile := ChangeSetFile{}
	err = json.Unmarshal(changeSetData, &changeSetFile)
	if err != nil {
		return nil, custom_error.MakeErrorf("Failed to unmarshal changeset. Error: %v", err)
	}
	changes := make([]*Change, 0, len(changeSetFile.Changes))
	changeIDFormat := changeSetFile.ID + "_transaction_entry_%v"
	for i := range changeSetFile.Changes {
		change, errValue := NewChange(&changeSetFile.Changes[i], workingDir, filepath.Dir(path), fmt.Sprintf(changeIDFormat, i))
		if errValue != nil {
			return nil, custom_error.NewErrorf(errValue, "Failed to validate changeset at path '%v'", path)
		}
		changes = append(changes, change)
	}
	return &ChangeSet{
		ID:      changeSetFile.ID,
		Changes: changes,
	}, nil
}

func (c *mainChangeLog) validate() custom_error.CustomError {
	if len(c.Connection) <= 0 {
		return custom_error.MakeErrorf("MainChangeLog format error: no connection string provided. Expected non-empty field 'connection'")
	}
	if len(c.DbName) <= 0 {
		return custom_error.MakeErrorf("MainChangeLog format error: no db-name provided. Expected non-empty field 'dbname'")
	}
	if (c.MigrationFiles == nil) || len(c.MigrationFiles) <= 0 {
		return custom_error.MakeErrorf("MainChangeLog format error: no migrations specified. Expected non-empty field 'migrations'")
	}
	changeSets := make([]*ChangeSet, 0, len(c.MigrationFiles))
	for i := range c.MigrationFiles {
		err := c.MigrationFiles[i].validate()
		if err != nil {
			return custom_error.NewErrorf(err, "MainChangeLog format error: migrationFile #%v", i+1)
		}
		fullPath := getFullPath(c.MigrationFiles[i].RelativePath, c.workingDir, c.workingDir, c.MigrationFiles[i].Path)
		changeSet, err := loadChangeSet(fullPath, c.workingDir)
		if err != nil {
			return custom_error.NewErrorf(err, "Failed to load migration '%v'", c.MigrationFiles[i].Path)
		}
		changeSets = append(changeSets, changeSet)
	}
	c.changeSets = changeSets
	return nil
}

func (c *mainChangeLog) GetConnectionString() string {
	return c.Connection
}

func (c *mainChangeLog) GetDBName() string {
	return c.DbName
}

func (c *mainChangeLog) GetChangeSets() []*ChangeSet {
	return c.changeSets
}

func (c *mainChangeLog) GetChangeSetSource() ChangeSetSource {
	return c
}

func (c *mainChangeLog) Apply(processor ChangeSetProcessor) custom_error.CustomError {
	err := c.strategy(c.changeSets, processor)
	if err != nil {
		return custom_error.NewErrorf(err, "Failed to apply changes.")
	}
	return nil
}

func forwardStrategy(sets []*ChangeSet, processor ChangeSetProcessor) custom_error.CustomError {
	for i := range sets {
		err := processor.Process(sets[i])
		if err != nil {
			return custom_error.NewErrorf(err, "Failed to process changeset '%v'", sets[i].ID)
		}
	}
	return nil
}

func backwardStrategy(sets []*ChangeSet, processor ChangeSetProcessor) custom_error.CustomError {
	for i := len(sets) - 1; i >= 0; i-- {
		err := processor.Process(sets[i])
		if err != nil {
			return custom_error.NewErrorf(err, "Failed to process changeset '%v'", sets[i].ID)
		}
	}
	return nil
}

func NewChangeLog(path string) (ChangeLog, custom_error.CustomError) {
	if len(path) <= 0 {
		return nil, custom_error.MakeErrorf("Input changelog path is empty. Internal error.")
	}

	changeLogData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, custom_error.MakeErrorf("Failed to open changelog file. Error: %v", err)
	}
	changeLog := mainChangeLog{
		workingDir: filepath.Dir(path),
		strategy:   forwardStrategy,
	}
	err = json.Unmarshal(changeLogData, &changeLog)
	if err != nil {
		return nil, custom_error.MakeErrorf("Failed to unmarshal changelog. Error: %v", err)
	}
	errValue := changeLog.validate()
	if errValue != nil {
		return nil, custom_error.NewErrorf(errValue, "Changelog validation failed")
	}
	return &changeLog, nil
}

func NewRollbackChangeLog(path string) (ChangeLog, custom_error.CustomError) {
	if len(path) <= 0 {
		return nil, custom_error.MakeErrorf("Input changelog path is empty. Internal error.")
	}

	changeLogData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, custom_error.MakeErrorf("Failed to open changelog file. Error: %v", err)
	}
	changeLog := mainChangeLog{
		workingDir: filepath.Dir(path),
		strategy:   backwardStrategy,
	}
	err = json.Unmarshal(changeLogData, &changeLog)
	if err != nil {
		return nil, custom_error.MakeErrorf("Failed to unmarshal changelog. Error: %v", err)
	}
	errValue := changeLog.validate()
	if errValue != nil {
		return nil, custom_error.NewErrorf(errValue, "Changelog validation failed")
	}
	return &changeLog, nil
}

package engine

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"hash"
	"io/ioutil"
	"path/filepath"

	"github.com/coldze/mongol/engine/decoding"
	"github.com/coldze/primitives/custom_error"
	"github.com/mongodb/mongo-go-driver/bson"
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
	migrationContent, ioErr := ioutil.ReadFile(fullPath)
	if ioErr != nil {
		return nil, custom_error.MakeErrorf("Failed to read file '%v'. Error: %v", m.Path, ioErr)
	}
	migrationDocument, err := decoding.Decode(migrationContent)
	if err != nil {
		return nil, custom_error.MakeErrorf("Failed to generate migration from file '%v'. Error: %v", m.Path, err)
	}
	hash.Write(migrationContent)
	return &SimpleMigration{
		source:            m,
		migrationDocument: migrationDocument,
	}, nil
}

type DocumentApplier interface {
	Apply(value *bson.Value) custom_error.CustomError
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
	source            *MigrationFile
	migrationDocument *bson.Value
}

func (s *SimpleMigration) Apply(visitor DocumentApplier) custom_error.CustomError {
	return visitor.Apply(s.migrationDocument)
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

func NewChange(c *ChangeFile, workingDir string, changelogPath string, hash hash.Hash) (*Change, custom_error.CustomError) {
	forward, err := NewMigration(&c.Forward, workingDir, changelogPath, hash)
	if err != nil {
		return nil, custom_error.NewErrorf(err, "Failed to generate change. Forward migration generate process failed.")
	}
	backward, err := NewMigration(c.Backward, workingDir, changelogPath, hash)
	if err != nil {
		return nil, custom_error.NewErrorf(err, "Failed to generate change. Backward migration generate process failed.")
	}
	return &Change{
		Backward: backward,
		Forward:  forward,
	}, nil
}

type ChangeSetFile struct {
	ID      string       `json:"id"`
	Changes []ChangeFile `json:"changes,omitempty"`
}
type Change struct {
	Forward  Migration
	Backward Migration
}

type ChangeSet struct {
	ID      string
	Hash    string
	Changes []*Change
}

type mainChangeLog struct {
	workingDir     string          `json:"-"`
	Connection     string          `json:"connection,omitempty"`
	DbName         string          `json:"dbname,omitempty"`
	MigrationFiles []MigrationFile `json:"migrations,omitempty"`
	changeSets     []*ChangeSet    `json:"-"`
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
	changesetHash := md5.New()
	changes := make([]*Change, 0, len(changeSetFile.Changes))
	for i := range changeSetFile.Changes {
		change, errValue := NewChange(&changeSetFile.Changes[i], workingDir, filepath.Dir(path), changesetHash)
		if errValue != nil {
			return nil, custom_error.NewErrorf(errValue, "Failed to validate changeset at path '%v'", path)
		}
		changes = append(changes, change)
	}
	return &ChangeSet{
		ID:      changeSetFile.ID,
		Hash:    hex.EncodeToString(changesetHash.Sum(nil)),
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
	for i := range c.changeSets {
		err := processor.Process(c.changeSets[i])
		if err != nil {
			return custom_error.NewErrorf(err, "Failed to process changeset '%v'", c.changeSets[i].ID)
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

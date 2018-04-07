package migrations

import (
	"bitbucket.org/4fit/mongol/primitives/custom_error"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
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
}

type MigrationFile struct {
	fullPath     string `json:"-"`
	Path         string `json:"include,omitempty"`
	RelativePath bool   `json:"relativeToChangelogFile"`
	NewMigration MigrationFactory
}

func (m *MigrationFile) validate() custom_error.CustomError {
	if len(m.Path) <= 0 {
		return custom_error.MakeErrorf("MigrationFile format error: path is empty. Expected non-empty field 'path'")
	}
	return nil
}

func (m *MigrationFile) updatePath(workingDir string, changelogPath string) {
	if m.RelativePath {
		m.fullPath = filepath.Join(workingDir, m.Path)
		return
	}
	m.fullPath = filepath.Join(changelogPath, m.Path)
}

func (m *MigrationFile) generate(workingDir string, changelogPath string, hash hash.Hash) (Migration, custom_error.CustomError) {
	err := m.validate()
	if err != nil {
		return nil, custom_error.NewErrorf(err, "Failed to generate migration")
	}
	m.updatePath(workingDir, changelogPath)
	migrationContent, ioErr := ioutil.ReadFile(m.fullPath)
	if ioErr != nil {
		return nil, custom_error.MakeErrorf("Failed to read file '%v'. Error: %v", m.Path, ioErr)
	}
	hash.Write(migrationContent)
	migration, err := m.NewMigration(migrationContent)
	if err != nil {
		return nil, custom_error.NewErrorf(err, "Failed to create migration")
	}
	return migration, custom_error.MakeErrorf("Not implemented")
}

type Migration interface {
	Apply() custom_error.CustomError
}

type MigrationFactory func(data []byte) (Migration, custom_error.CustomError)

type DummyMigration struct {
}

func (d *DummyMigration) Apply() custom_error.CustomError {
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

func (c *ChangeFile) updatePath(workingDir string, changelogPath string) {
	c.Forward.updatePath(workingDir, changelogPath)
	if c.Backward == nil {
		return
	}
	c.Backward.updatePath(workingDir, changelogPath)
}

func (c *ChangeFile) generate(workingDir string, changelogPath string, hash hash.Hash) (*Change, custom_error.CustomError) {
	forward, err := c.Forward.generate(workingDir, changelogPath, hash)
	if err != nil {
		return nil, custom_error.NewErrorf(err, "Failed to generate change. Forward migration generate process failed.")
	}
	var backward Migration
	if c.Backward != nil {
		backward, err = c.Backward.generate(workingDir, changelogPath, hash)
		if err != nil {
			return nil, custom_error.NewErrorf(err, "Failed to generate change. Backward migration generate process failed.")
		}
	} else {
		backward = &DummyMigration{}
	}
	return &Change{
		Backward: backward,
		Forward: forward,
	}, custom_error.MakeErrorf("Not implemented")
}

type ChangeSetFile struct {
	ID      string   `json:"id"`
	Changes []ChangeFile `json:"changes,omitempty"`
}
type Change struct {
	Forward Migration
	Backward Migration
}

type ChangeSet struct {
	ID      string
	Hash    string
	Changes []*Change
}

func (c *ChangeSet) Apply() (errResult custom_error.CustomError) {
	rollback := []Migration{}
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		errValue, ok := r.(custom_error.CustomError)
		if ok {
			errResult = errValue
		}
		for i := len(rollback) - 1; i >=0; i-- {
			rollbackErr := rollback[i].Apply()
			if rollbackErr != nil {
				errResult = custom_error.NewErrorf(errResult, "Migration failed.")
			}
		}
	}()
	for i := range c.Changes {
		err := c.Changes[i].Forward.Apply()
		if err != nil {
			panic(custom_error.NewErrorf(err, "Migration failed. Change-set: %v. Migration: %v", c.ID, i+1))
		}
		rollback = append(rollback, c.Changes[i].Backward)
	}
	return custom_error.MakeErrorf("Not implemented")
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
		change, errValue := changeSetFile.Changes[i].generate(workingDir, filepath.Dir(path), changeSetFile)
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
		c.MigrationFiles[i].updatePath(c.workingDir, c.workingDir)
		changeSet, err := loadChangeSet(c.MigrationFiles[i].fullPath, c.workingDir)
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

func NewChangeLog(path *string) (ChangeLog, custom_error.CustomError) {
	if (path == nil) || len(*path) <= 0 {
		return nil, custom_error.MakeErrorf("Input changelog path is empty. Internal error.")
	}

	changeLogData, err := ioutil.ReadFile(*path)
	if err != nil {
		return nil, custom_error.MakeErrorf("Failed to open changelog file. Error: %v", err)
	}
	changeLog := mainChangeLog{
		workingDir: filepath.Dir(*path),
	}
	err = json.Unmarshal(changeLogData, &changeLog)
	if err != nil {
		return nil, custom_error.MakeErrorf("Failed to unmarshal changelog. Error: %v", err)
	}
	errValue := changeLog.validate()
	if err != nil {
		return nil, custom_error.NewErrorf(errValue, "Changelog validation failed")
	}
	return &changeLog, nil
}

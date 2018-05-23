package engine

import (
	"github.com/coldze/primitives/custom_error"
)

type arrayChangeLog struct {
	changeSets []*ChangeSet
}

func (c *arrayChangeLog) Apply(processor ChangeSetProcessor) custom_error.CustomError {
	for i := range c.changeSets {
		err := processor.Process(c.changeSets[i])
		if err != nil {
			return custom_error.NewErrorf(err, "Failed to process changeset '%v'", c.changeSets[i].ID)
		}
	}
	return nil
}

func NewArrayChangeLog(changeSets []*ChangeSet) (ChangeSetSource, custom_error.CustomError) {
	return &arrayChangeLog{
		changeSets: changeSets,
	}, nil
}

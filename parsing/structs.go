package parsing

import (
	"context"
	"github.com/mongodb/mongo-go-driver/bson"
)

type AdminCommand map[string]interface{}

type RunCommand interface {
	GetContext() context.Context
	GetCommand() *bson.Document
}

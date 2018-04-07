package parsing

import (
	"bitbucket.org/4fit/mongol/primitives/custom_error"
	"github.com/mongodb/mongo-go-driver/bson"
)

func ComposeDocument(args map[string]interface{}) (*bson.Element, custom_error.CustomError) {
	return nil, custom_error.MakeErrorf("Not implemented")
}

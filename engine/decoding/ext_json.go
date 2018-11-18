package decoding

import (
	"github.com/coldze/primitives/custom_error"
	"github.com/mongodb/mongo-go-driver/bson"
)

func DecodeExt(data []byte) (interface{}, custom_error.CustomError) {
	doc := bson.D{}
	err := bson.UnmarshalExtJSON(data, false, &doc)
	if err != nil {
		return nil, custom_error.MakeErrorf("Failed to decode ext-json. Error: %v", err)
	}
	return doc, nil
}

package decoding

import (
	"github.com/coldze/primitives/custom_error"
	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/x/bsonx"
	"log"
)

func DecodeExt(data []byte) (interface{}, custom_error.CustomError) {
	doc := bson.D{}
	err := bson.UnmarshalExtJSON(data, false, &doc)
	if err != nil {
		return nil, custom_error.MakeErrorf("Failed to decode ext-json. Error: %v", err)
	}
	return doc, nil
}

func DecodeMigration(data []byte) ([]interface{}, custom_error.CustomError) {
	doc := bson.D{}
	err := bson.UnmarshalExtJSON(data, false, &doc)
	if err != nil {
		return nil, custom_error.MakeErrorf("Failed to decode ext-json. Error: %v", err)
	}
	commands, ok := doc.Map()["cmds"]
	if !ok {
		return []interface{}{doc}, nil
	}
	arr, ok := commands.(bsonx.Arr)
	if !ok {
		log.Printf("Not an array")
		return []interface{}{doc}, nil
	}
	res := []interface{}{}
	for i := range arr {
		res = append(res, arr[i])
	}
	return res, nil
}

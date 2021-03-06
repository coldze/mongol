package decoding

import (
	"github.com/coldze/primitives/custom_error"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	mapped := doc.Map()
	if len(mapped) <= 0 {
		return nil, nil
	}
	commands, ok := mapped["cmds"]
	if !ok {
		return []interface{}{doc}, nil
	}
	arr, ok := commands.(primitive.A)
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

package commands

import (
	"context"

	"github.com/coldze/primitives/custom_error"
	mgo "github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/options"
)

func newMgoClient(ctx context.Context, uri string) (*mgo.Client, custom_error.CustomError) {
	clientOpts := options.Client()
	if clientOpts == nil {
		return nil, custom_error.MakeErrorf("Something went completely wrong. ClientOpts are nil")
	}
	clientOpts = clientOpts.ApplyURI(uri)
	if clientOpts == nil {
		return nil, custom_error.MakeErrorf("Something went completely wrong. ClientOpts are nil")
	}
	err := clientOpts.Validate()
	if err != nil {
		return nil, custom_error.MakeErrorf("Failed to validate ClientOpts. Error: %v", err)
	}
	mongoClient, err := mgo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, custom_error.MakeErrorf("Failed to connect to mongo. Error: %v", err)
	}
	return mongoClient, nil
}

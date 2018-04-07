package parsing

import (
	"bitbucket.org/4fit/mongol/primitives/custom_error"
	"github.com/mongodb/mongo-go-driver/bson"
	"reflect"
)

const (
	cmd_create_collection = "createCollection"
)

const (
	cmd_args = "options"
)

type commandValidator func(interface{}, map[string]interface{}) (*bson.Document, custom_error.CustomError)
type supportedCommand struct {
	cmdName     string
	cmdValidate commandValidator
}

var supportedCommands = make([]*supportedCommand, 0, 1)

func addCommand(cmd string, cmdValidator commandValidator) {
	supportedCommands = append(supportedCommands, &supportedCommand{
		cmdName:     cmd,
		cmdValidate: cmdValidator,
	})
}

func GenerateCommand(args map[string]interface{}) (*bson.Document, custom_error.CustomError) {
	for _, checkingCmd := range supportedCommands {
		cmd, ok := args[checkingCmd.cmdName]
		if !ok {
			continue
		}
		delete(args, checkingCmd.cmdName)
		cmdDoc, err := checkingCmd.cmdValidate(cmd, args)
		if err != nil {
			return nil, err
		}
		return cmdDoc, nil
	}
	return nil, custom_error.MakeErrorf("No supported command found")
}

func parseValue(value interface{}) (*bson.Value, custom_error.CustomError) {
	if value == nil {
		return bson.VC.Null(), nil
	}
	typeValue := reflect.TypeOf(value).String()
	switch typeValue {
	case "map[string] interface{}":
		{
			subDoc, err := composeParameters(value.(map[string]interface{}))
			if err != nil {
				return nil, custom_error.NewErrorf(err, "Failed to parse value")
			}
			return bson.VC.Document(subDoc), nil
		}
	case "[]interface {}":
		{
			arrValue := value.([]interface{})
			arrElements := make([]*bson.Value, 0, len(arrValue))
			for i := range arrValue {
				arrElement, err := parseValue(arrValue[i])
				if err != nil {
					return nil, custom_error.NewErrorf(err, "Failed to parse array element with index %v", i)
				}
				arrElements = append(arrElements, arrElement)
			}
			return bson.VC.ArrayFromValues(arrElements...), nil
		}
	case "string":
		{
			return bson.VC.String(value.(string)), nil
		}
	case "float64":
		{
			return bson.VC.Double(value.(float64)), nil
		}
	default:
		{
			return nil, custom_error.MakeErrorf("Can't parse field. Type: '%v'", typeValue)
		}
	}
}

func parseElement(key string, value interface{}) (*bson.Element, custom_error.CustomError) {
	if value == nil {
		return bson.EC.Null(key), nil
	}
	typeValue := reflect.TypeOf(value).String()
	switch typeValue {
	case "map[string]interface {}":
		{
			subDoc, err := composeParameters(value.(map[string]interface{}))
			if err != nil {
				return nil, custom_error.NewErrorf(err, "Failed to parse parameter '%v'", key)
			}
			return bson.EC.SubDocument(key, subDoc), nil
		}
	case "[]interface {}":
		{
			arrValue := value.([]interface{})
			arrElements := make([]*bson.Value, 0, len(arrValue))
			for i := range arrValue {
				arrElement, err := parseValue(arrValue[i])
				if err != nil {
					return nil, custom_error.NewErrorf(err, "Failed to parse array element with index %v", i)
				}
				arrElements = append(arrElements, arrElement)
			}
			return bson.EC.ArrayFromElements(key, arrElements...), nil
		}
	case "string":
		{
			return bson.EC.String(key, value.(string)), nil
		}
	case "float64":
		{
			return bson.EC.Double(key, value.(float64)), nil
		}
	default:
		{
			return nil, custom_error.MakeErrorf("Can't parse field '%v'. Type: '%v'", key, typeValue)
		}
	}
}

func composeParameters(args map[string]interface{}) (*bson.Document, custom_error.CustomError) {
	elements := []*bson.Element{}
	for k, v := range args {
		element, err := parseElement(k, v)
		if err != nil {
			return nil, custom_error.NewErrorf(err, "Failed to parse value of %v", k)
		}
		elements = append(elements, element)
	}
	return bson.NewDocument(elements...), nil
}

func validateCreateCollection(arg interface{}, args map[string]interface{}) (*bson.Document, custom_error.CustomError) {
	argType := reflect.TypeOf(arg)
	if argType.String() != "string" {
		return nil, custom_error.MakeErrorf("Invalid arg type for createCollection. Expected string. Got: %v", argType.String())
	}
	elements := []*bson.Element{bson.EC.String("create", arg.(string))}
	for k, v := range args {
		element, err := parseElement(k, v)
		if err != nil {
			return nil, custom_error.NewErrorf(err, "Failed to parse value of %v", k)
		}
		elements = append(elements, element)
	}
	return bson.NewDocument(elements...), nil
}

func init() {
	addCommand(cmd_create_collection, validateCreateCollection)
}

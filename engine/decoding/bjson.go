package decoding

import (
	"github.com/coldze/primitives/custom_error"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"strconv"
	"strings"
	"time"
)

const (
	TYPE_UNKNOWN = iota
	TYPE_NULL
	TYPE_BOOLEAN
	TYPE_INT32
	TYPE_INT64
	TYPE_FLOAT64
	TYPE_DECIMAL128
	TYPE_STRING
	TYPE_ARRAY
	TYPE_OBJECT
)

const (
	SYMBOL_CURL_BRACER_OPEN  = '{'
	SYMBOL_CURL_BRACER_CLOSE = '}'
	SYMBOL_BRACER_OPEN       = '['
	SYMBOL_BRACER_CLOSE      = ']'
	SYMBOL_COMMA             = ','
	SYMBOL_DOUBLE_DOT        = ':'
	SYMBOL_QUOTE             = '"'
	SYMBOL_SLASH             = '/'
	SYMBOL_BACKSLASH         = '\\'
	SYMBOL_SPACE             = ' '
	SYMBOL_TAB               = '\t'
	SYMBOL_NEW_LINE          = '\n'
	SYMBOL_CARRIAGE_RETURN   = '\r'
)

type Object struct {
	TypedValue interface{}
	RawValue   []byte
	Type       int64
}

func Decode(data []byte) (interface{}, custom_error.CustomError) {
	res, _, err := parseAny(data)
	if err != nil {
		return nil, custom_error.NewErrorf(err, "Failed to decode data")
	}
	return res, nil
}

func collectEscapedString(data []byte) (int, custom_error.CustomError) {
	if len(data) <= 0 {
		return 0, custom_error.MakeErrorf("Escaped sequence wrong fromat")
	}
	c := data[0]
	switch c {
	case 'u':
		{
			if len(data) < 4 {
				return 0, custom_error.MakeErrorf("Escaped sequence wrong fromat")
			}
			for i := 1; i < 4; i++ {
				uniC := data[i]
				if '0' <= uniC && uniC <= '9' || 'a' <= uniC && uniC <= 'f' || 'A' <= uniC && uniC <= 'F' {
					continue
				}
				return 0, custom_error.MakeErrorf("Escaped sequence wrong fromat")
			}
			return 4, nil
		}
	case 'b', 'f', 'n', 'r', 't', SYMBOL_BACKSLASH, SYMBOL_SLASH, SYMBOL_QUOTE:
		{
			return 1, nil
		}
	}
	return 0, custom_error.MakeErrorf("Escaped sequence wrong fromat")
}

func extractString(in []byte, ending byte) (*string, []byte, custom_error.CustomError) {
	data := in
	max := len(data)
	if max <= 0 {
		return nil, nil, custom_error.MakeErrorf("Invalid format")
	}
	i := 0
	for {
		c := data[i]
		switch {
		case c == ending:
			{
				v := string(data[0:i])
				return &v, data[i+1:], nil
			}
		case c == SYMBOL_BACKSLASH:
			{
				inc, err := collectEscapedString(data[i+1:])
				if err != nil {
					return nil, nil, custom_error.NewErrorf(err, "Escaped sequence parsing failed.")
				}
				i += inc
			}
		case c < 0x20:
			{

			}
		}
		i++
		if i >= max {
			return nil, nil, custom_error.MakeErrorf("Invalid format")
		}
	}
	return nil, nil, custom_error.MakeErrorf("Invalid format")
}

func parseString(in []byte) (interface{}, []byte, custom_error.CustomError) {
	v, data, err := extractString(in, SYMBOL_QUOTE)
	if err != nil {
		return nil, nil, custom_error.NewErrorf(err, "Failed to parse string")
	}
	if v == nil {
		return nil, nil, custom_error.MakeErrorf("Failed to parse string: empty data")
	}
	return *v, data, err
}

func extractNumber(in []byte) (*string, []byte, custom_error.CustomError) {
	data := skipEmpty(in)
	max := len(data)
	if max <= 0 {
		return nil, nil, custom_error.MakeErrorf("Invalid format")
	}
	for i, v := range data {
		if (v >= '0') && (v <= '9') {
			continue
		}
		if (v == 'E') || (v == 'e') || (v == '.') || (v == '+') || (v == '-') {
			continue
		}
		s := string(data[:i])
		return &s, data[i:], nil
	}
	return nil, nil, custom_error.MakeErrorf("Invalid format")
}

func parseNumber(in []byte) (interface{}, []byte, custom_error.CustomError) {
	s, data, errValue := extractNumber(in)
	if errValue != nil {
		return nil, nil, custom_error.NewErrorf(errValue, "Failed to parse number")
	}
	floatValue, err := strconv.ParseFloat(*s, 10)
	if err != nil {
		return nil, nil, custom_error.NewErrorf(custom_error.MakeError(err), "Failed to parse number")
	}
	return floatValue, data, nil
}

func parseDatetime(in []byte) (interface{}, []byte, custom_error.CustomError) {
	dateStr, data, err := extractString(in, ')')
	if err != nil {
		return nil, nil, custom_error.NewErrorf(err, "Failed to parse date-time")
	}
	if dateStr == nil {
		return nil, nil, custom_error.MakeErrorf("Failed to parse date-time. Empty data")
	}
	parts := strings.Split(*dateStr, ",")
	var format string
	switch len(parts) {
	case 1:
		{
			format = time.RFC3339
		}
	case 2:
		{
			format = strings.Trim(parts[1], " \t")
		}
	default:
		{
			return nil, nil, custom_error.MakeErrorf("Invalid format of date-time")
		}
	}
	timeValue, errValue := time.Parse(format, strings.Trim(parts[0], " \t"))
	if errValue != nil {
		return nil, nil, custom_error.MakeError(err)
	}
	return primitive.DateTime(timeValue.Unix() / 1000000), data, nil
}

func extractTypedInt(in []byte) (int64, []byte, custom_error.CustomError) {
	strValue, data, err := extractNumber(in)
	if err != nil {
		return 0, nil, custom_error.NewErrorf(err, "Failed to parse int")
	}
	if strValue == nil || len(*strValue) <= 0 {
		return 0, nil, custom_error.MakeErrorf("Failed to parse int. No data")
	}
	intValue, errValue := strconv.ParseInt(*strValue, 10, 64)
	if errValue != nil {
		return 0, nil, custom_error.NewErrorf(custom_error.MakeError(errValue), "Failed to parse int.")
	}
	return intValue, data, nil
}

func parseInt64(in []byte) (interface{}, []byte, custom_error.CustomError) {
	res, data, err := extractTypedInt(in)
	if err != nil {
		return nil, nil, custom_error.NewErrorf(err, "Failed to parse int64")
	}
	return res, data, nil
}

func parseInt32(in []byte) (interface{}, []byte, custom_error.CustomError) {
	res, data, err := extractTypedInt(in)
	if err != nil {
		return nil, nil, custom_error.NewErrorf(err, "Failed to parse int64")
	}
	return int32(res), data, nil
}

func parseDecimal128(in []byte) (interface{}, []byte, custom_error.CustomError) {
	strValue, data, err := extractString(in, ')')
	if err != nil {
		return nil, nil, custom_error.NewErrorf(err, "Failed to parse decimal128")
	}
	if strValue == nil || len(*strValue) <= 0 {
		return nil, nil, custom_error.MakeErrorf("Failed to parse decimal128. No data")
	}
	decimalValue, errValue := primitive.ParseDecimal128(*strValue)
	if errValue != nil {
		return nil, nil, custom_error.NewErrorf(custom_error.MakeError(errValue), "Failed to parse decimal128")
	}
	return decimalValue, data, nil
}

func parseTimestamp(in []byte) (interface{}, []byte, custom_error.CustomError) {
	p1, data, err := extractNumber(in)
	if err != nil {
		return nil, nil, custom_error.NewErrorf(err, "Failed to parse timestamp")
	}
	if p1 == nil {
		return nil, nil, custom_error.MakeErrorf("Invalid format")
	}
	p1value, errValue := strconv.ParseUint(*p1, 10, 32)
	if errValue != nil {
		return nil, nil, custom_error.MakeError(errValue)
	}
	data = skipEmpty(data)
	if len(data) < 1 {
		return nil, nil, custom_error.MakeErrorf("Invalid format")
	}
	if data[0] != ',' {
		return nil, nil, custom_error.MakeErrorf("Invalid format")
	}
	p2, data, err := extractNumber(data[1:])
	if err != nil {
		return nil, nil, custom_error.NewErrorf(err, "Failed to parse timestamp")
	}
	if p2 == nil {
		return nil, nil, custom_error.MakeErrorf("Invalid format")
	}
	p2value, errValue := strconv.ParseUint(*p2, 10, 32)
	if errValue != nil {
		return nil, nil, custom_error.MakeError(errValue)
	}
	return primitive.Timestamp{T: uint32(p1value), I: uint32(p2value)}, data, nil
}

type parsingFunc func(in []byte) (interface{}, []byte, custom_error.CustomError)

func parseSpecial(in []byte, expected string, parse parsingFunc) (interface{}, []byte, custom_error.CustomError) {
	data := in
	expectedLength := len(expected)
	if len(data) < expectedLength+3 {
		return nil, nil, custom_error.MakeErrorf("Invalid format")
	}
	if string(data[:expectedLength]) != expected && (data[expectedLength] != '(') {
		return nil, nil, custom_error.MakeErrorf("Invalid format")
	}
	var v interface{}
	var err custom_error.CustomError
	v, data, err = parse(data[expectedLength+1:])
	if err != nil {
		return nil, nil, custom_error.NewErrorf(err, "Failed to parse double()")
	}
	data = skipEmpty(data)
	if len(data) < 1 && data[0] != ')' {
		return nil, nil, custom_error.MakeErrorf("Invalid format")
	}
	return v, data[1:], nil
}

func parseObjectID(in []byte) (interface{}, []byte, custom_error.CustomError) {
	data := skipEmpty(in)
	if len(data) < 2 {
		return nil, nil, custom_error.MakeErrorf("Invalid format")
	}
	hexValue, data, err := extractString(data[1:], ')')
	if err != nil {
		return nil, nil, custom_error.NewErrorf(err, "Failed to parse ObjectID")
	}
	if hexValue == nil {
		return nil, nil, custom_error.MakeErrorf("Failed to parse ObjectID. Empty data.")
	}
	v, errValue := primitive.ObjectIDFromHex(*hexValue)
	if errValue != nil {
		return nil, nil, custom_error.NewErrorf(custom_error.MakeError(err), "Failed to parse ObjectID")
	}
	return v, data, nil
}

func parseAny(in []byte) (interface{}, []byte, custom_error.CustomError) {
	data := skipEmpty(in)
	if len(data) <= 0 {
		return nil, nil, custom_error.MakeErrorf("Invalid format")
	}
	c := data[0]
	var err custom_error.CustomError
	switch c {
	case SYMBOL_QUOTE:
		{
			return parseString(data[1:])
		}
	case 'i':
		{
			//int32
			if len(data) < 5 {
				return nil, nil, custom_error.MakeErrorf("Ivalid format")
			}
			if data[3] == '3' {
				return parseSpecial(data, "int32", parseInt32)
			}
			if data[3] == '6' {
				return parseSpecial(data, "int64", parseInt64)
			}
			return nil, nil, custom_error.MakeErrorf("Invalid format")
			/*
				bson.VC.Int32()
				bson.VC.Int64()
			*/
		}
	case 'd':
		{
			switch data[1] {
			case 'a':
				{
					return parseSpecial(data, "datetime", parseDatetime)
				}
			case 'e':
				{
					return parseSpecial(data, "decimal128", parseDecimal128)
				}
			}

			//return nil, nil, custom_error.MakeErrorf("Not implemented")
			/*
				bson.VC.DateTime()
				bson.VC.Decimal128()
			*/
		}
	case 'o':
		{
			if len(data) < 13 {
				return nil, nil, custom_error.MakeErrorf("Invalid format")
			}
			return parseSpecial(data, "objectID", parseObjectID)
			/*
				bson.VC.ObjectID()
			*/
		}
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		{
			var v interface{}
			v, data, err = parseNumber(data[:])
			if err != nil {
				return nil, nil, custom_error.NewErrorf(err, "Invalid format.")
			}
			return v, data, nil
		}
	case 'f':
		{
			if len(data) < 5 {
				return nil, nil, custom_error.MakeErrorf("Invalid format")
			}
			if string(data[:5]) == "false" {
				return false, data[5:], nil
			}
			return nil, nil, custom_error.MakeErrorf("Invalid format")
		}
	case 't':
		{
			//return nil, nil, custom_error.MakeErrorf("Not implemented")
			/*
				bson.VC.Timestamp()
			*/
			if len(data) < 4 {
				return nil, nil, custom_error.MakeErrorf("Invalid format")
			}
			if data[1] == 'i' {
				return parseSpecial(data, "timestamp", parseTimestamp)
			}
			if string(data[:4]) == "true" {
				return true, data[4:], nil
			}
			return nil, nil, custom_error.MakeErrorf("Invalid format")
		}
	case 'n':
		{
			if len(data) < 4 {
				return nil, nil, custom_error.MakeErrorf("Invalid format")
			}
			if string(data[:4]) == "null" {
				return primitive.Null{}, data[4:], nil
			}
			return nil, nil, custom_error.MakeErrorf("Invalid format")
		}
	case SYMBOL_BRACER_OPEN:
		{
			return parseArray(data[1:])
		}
	case SYMBOL_CURL_BRACER_OPEN:
		{
			return parseObject(data[1:])
		}
	default:
		return nil, nil, custom_error.MakeErrorf("Invalid format")
	}
	return nil, in, custom_error.MakeErrorf("Not implemented")
}

func parseObject(in []byte) (interface{}, []byte, custom_error.CustomError) {
	data := skipEmpty(in)
	if len(data) <= 0 {
		return nil, nil, custom_error.MakeErrorf("Invalid format")
	}
	if data[0] == SYMBOL_CURL_BRACER_CLOSE {
		return map[string]interface{}{}, data[1:], nil
	}
	res := map[string]interface{}{}
	for {
		if data[0] != SYMBOL_QUOTE {
			return nil, nil, custom_error.MakeErrorf("Invalid format")
		}
		var key *string
		var err custom_error.CustomError
		key, data, err = extractString(data[1:], '"')
		if err != nil {
			return nil, nil, custom_error.NewErrorf(err, "Failed to parse object key")
		}
		if key == nil {
			return nil, nil, custom_error.MakeErrorf("Failed to get object key")
		}
		data = skipEmpty(data)
		if len(data) <= 0 {
			return nil, nil, custom_error.MakeErrorf("Invalid format")
		}
		if data[0] != SYMBOL_DOUBLE_DOT {
			return nil, nil, custom_error.MakeErrorf("Invalid format")
		}
		var value interface{}
		value, data, err = parseAny(data[1:])
		if err != nil {
			return nil, nil, custom_error.NewErrorf(err, "Failed to parse value for key in map")
		}
		data = skipEmpty(data)
		if len(data) <= 0 {
			return nil, nil, custom_error.MakeErrorf("Invalid format")
		}
		_, ok := res[*key]
		if ok {
			return nil, nil, custom_error.MakeErrorf("Dublicated key: %v", *key)
		}
		res[*key] = value
		switch data[0] {
		case SYMBOL_CURL_BRACER_CLOSE:
			{
				return res, data[1:], nil
			}
		case SYMBOL_COMMA:
			{
				data = skipEmpty(data[1:])
				if len(data) <= 0 {
					return nil, nil, custom_error.MakeErrorf("Invalid format")
				}
			}
		default:
			return nil, nil, custom_error.MakeErrorf("Invalid format")
		}
	}
	return nil, nil, custom_error.MakeErrorf("Invalid format")
	//key, data, err := parseString(data[1:])
}

func parseArray(in []byte) (interface{}, []byte, custom_error.CustomError) {
	data := skipEmpty(in)
	if len(data) <= 0 {
		return nil, nil, custom_error.MakeErrorf("Invalid format")
	}
	if data[0] == SYMBOL_BRACER_CLOSE {
		return []interface{}{}, data[1:], nil
	}
	res := []interface{}{}

	for {
		var value interface{}
		var err custom_error.CustomError
		value, data, err = parseAny(data)
		if err != nil {
			return nil, nil, custom_error.NewErrorf(err, "Invalid format")
		}
		data = skipEmpty(data)
		if len(data) <= 0 {
			return nil, nil, custom_error.MakeErrorf("Invalid format")
		}
		res = append(res, value)
		switch data[0] {
		case SYMBOL_BRACER_CLOSE:
			{
				return res, data[1:], nil
			}
		case SYMBOL_COMMA:
			{
				data = skipEmpty(data[1:])
				if len(data) <= 0 {
					return nil, nil, custom_error.MakeErrorf("Invalid format")
				}
				continue
			}
		}
		return nil, nil, custom_error.MakeErrorf("Invalid format")
	}
}

func skipEmpty(data []byte) []byte {
	for i, v := range data {
		if (v == SYMBOL_SPACE) || (v == SYMBOL_TAB) || (v == SYMBOL_NEW_LINE) || (v == SYMBOL_CARRIAGE_RETURN) {
			continue
		}
		return data[i:]
	}
	return []byte{}
}

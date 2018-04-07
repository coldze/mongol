package custom_error

import (
	"fmt"
	"runtime"
)

type ErrorDetails struct {
	Line int
	File string
}

type CustomError interface {
	fmt.Stringer
	error
	GetError() error
	GetSubError() CustomError
	GetDetails() ErrorDetails
}

type customErrorImpl struct {
	subError CustomError
	details  ErrorDetails
	err      error
}

func (c *customErrorImpl) GetError() error {
	return c.err
}

func (c *customErrorImpl) GetSubError() CustomError {
	return c.subError
}

func (c *customErrorImpl) GetDetails() ErrorDetails {
	return c.details
}

func (c *customErrorImpl) String() string {
	thisErr := fmt.Sprintf("%v:%v\n\tError: %v\n\n", c.details.File, c.details.Line, c.err)
	if c.subError != nil {
		thisErr += c.subError.String()
	}
	return thisErr
}

func (c *customErrorImpl) Error() string {
	return c.String()
}

func newError(subError CustomError, err error, skip int) CustomError {
	details := ErrorDetails{}
	_, details.File, details.Line, _ = runtime.Caller(skip)
	return &customErrorImpl{
		subError: subError,
		details:  details,
		err:      err,
	}
}

func NewError(subError CustomError, err error) CustomError {
	return newError(subError, err, 2)
}

func MakeError(err error) CustomError {
	return newError(nil, err, 2)
}

func MakeErrorf(format string, args ...interface{}) CustomError {
	return newError(nil, fmt.Errorf(format, args...), 2)
}

func NewErrorf(subError CustomError, format string, args ...interface{}) CustomError {
	return newError(subError, fmt.Errorf(format, args...), 2)
}

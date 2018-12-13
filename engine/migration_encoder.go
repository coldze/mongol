package engine

import "github.com/coldze/primitives/custom_error"

type EncoderFactory func() DocumentEncoder

type DocumentEncoder interface {
	DocumentApplier
	Encode() (interface{}, custom_error.CustomError)
}

type binEncoder struct {
	applied []interface{}
}

func (b *binEncoder) Apply(value interface{}) custom_error.CustomError {
	b.applied = append(b.applied, value)
	return nil
}

func (b *binEncoder) Encode() (interface{}, custom_error.CustomError) {
	return b.applied, nil
}

func NewBinEncoder() DocumentEncoder {
	return &binEncoder{
		applied: []interface{}{},
	}
}

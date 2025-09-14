package types

import (
	"fmt"
)

type Case[I, R any] struct {
	id                   string
	description          string
	inputGenerator       func() I
	inputEncoder         func(i I) ([]byte, error)
	responseUnmarshaller func(res []byte) (R, error)
	validator            func(input I, response R) (*ValidationResult, error)
}

func NewCase[I, R any](
	id, description string,
	inputGenerator func() I,
	inputEncoder func(i I) ([]byte, error),
	responseUnmarshaller func(res []byte) (R, error),
	validator func(input I, response R) (*ValidationResult, error),
) *Case[I, R] {
	return &Case[I, R]{
		id:                   id,
		description:          description,
		inputGenerator:       inputGenerator,
		inputEncoder:         inputEncoder,
		responseUnmarshaller: responseUnmarshaller,
		validator:            validator,
	}
}

func (c *Case[I, R]) ID() string {
	return c.id
}

func (c *Case[I, R]) Description() string {
	return c.description
}

func (c *Case[I, R]) GetInput() any {
	return c.inputGenerator()
}

func (c *Case[I, R]) EncodeInput(input any) ([]byte, error) {
	ti, ok := input.(I)
	if !ok {
		return nil, ErrIncorrectType
	}

	return c.inputEncoder(ti)
}

func (c *Case[I, R]) ValidateResponse(input any, response []byte) (*ValidationResult, error) {
	// Type assert the stored input back to the original type
	typedInput, ok := input.(I)
	if !ok {
		return nil, fmt.Errorf("stored input type assertion failed: expected %T, got %T", *new(I), input)
	}

	// Decode the response
	respValue, err := c.responseUnmarshaller(response)
	if err != nil {
		return nil, err
	}

	// Validate using the original generated input and the decoded response
	return c.validator(typedInput, respValue)
}

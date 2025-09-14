package types

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

type Registry struct {
	testCases map[string]TestCase
}

func NewRegistry() *Registry {
	return &Registry{
		testCases: make(map[string]TestCase),
	}
}

func (r *Registry) Register(testCase TestCase) error {
	if testCase.ID() == "" {
		return errors.New("testcase must have an id")
	}
	r.testCases[testCase.ID()] = testCase

	return nil
}

func (r *Registry) Get(id string) (TestCase, error) {
	tc, ok := r.testCases[id]
	if !ok {
		return nil, fmt.Errorf("testcase with id %s does not exist", id)
	}

	return tc, nil
}

func (r *Registry) List() []TestCase {
	return slices.Collect(maps.Values(r.testCases))
}

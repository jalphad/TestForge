package types

// TestCase represents a single test scenario
type TestCase interface {
	ID() string
	Description() string
	GetInput() any
	EncodeInput(input any) ([]byte, error)
	ValidateResponse(input any, response []byte) (*ValidationResult, error)
}

// TestRegistry manages available test cases
type TestRegistry interface {
	Register(testCase TestCase) error
	Get(id string) (TestCase, error)
	List() []TestCase
}

// ValidationResult contains the outcome of response validation
type ValidationResult struct {
	Valid   bool
	Score   float64
	Message string
	Details map[string]interface{}
}

// TestingServer is the main server interface
type TestingServer interface {
	Start() error
	Stop() error
	RegisterTestCase(testCase TestCase) error
	GetRegistry() TestRegistry
}

package types

import "time"

// TestCaseRequest tracks a specific test case request with its generated input
type TestCaseRequest struct {
	TestCaseID     string
	RequestID      string
	GeneratedInput interface{}
	Timestamp      time.Time
}

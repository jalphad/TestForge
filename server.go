package testforge

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	v1 "github.com/jalphad/testforge/proto"
	"github.com/jalphad/testforge/types"
	"google.golang.org/grpc"
)

func NewServer(cfg *Config) *Server {
	return &Server{
		config:         cfg,
		registry:       types.NewRegistry(),
		activeRequests: make(map[string]*types.TestCaseRequest),
		requestsMutex:  &sync.RWMutex{},
	}
}

// Server implements the gRPC testing service
type Server struct {
	v1.UnimplementedTestingServiceServer
	registry       types.TestRegistry
	grpcServer     *grpc.Server
	httpGateway    *http.Server // Optional REST gateway
	middleware     []Middleware
	config         *Config
	activeRequests map[string]*types.TestCaseRequest
	requestsMutex  *sync.RWMutex
}

func (s *Server) ListTestCases(ctx context.Context, request *v1.ListTestCasesRequest) (*v1.ListTestCasesResponse, error) {
	testCases := s.registry.List()

	// Convert to proto messages
	var protoTestCases []*v1.TestCaseInfo
	for _, tc := range testCases {
		protoTestCases = append(protoTestCases, &v1.TestCaseInfo{
			Id:          tc.ID(),
			Description: tc.Description(),
		})
	}

	return &v1.ListTestCasesResponse{
		TestCases: protoTestCases,
	}, nil
}

func (s *Server) GetTestCase(ctx context.Context, testcase *v1.GetTestCaseRequest) (*v1.GetTestCaseResponse, error) {
	testCase, err := s.registry.Get(testcase.Id)
	if err != nil {
		return nil, err
	}

	// Generate a unique request ID
	requestID, err := s.generateRequestID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate request ID: %w", err)
	}

	// Generate input for this specific request with the original value
	generatedInput := testCase.GetInput()

	// Store the request with its generated input
	s.requestsMutex.Lock()
	s.activeRequests[requestID] = &types.TestCaseRequest{
		TestCaseID:     testcase.Id,
		RequestID:      requestID,
		GeneratedInput: generatedInput,
		Timestamp:      time.Now(),
	}
	s.requestsMutex.Unlock()

	encodedInput, err := testCase.EncodeInput(generatedInput)
	if err != nil {
		return nil, err
	}

	return &v1.GetTestCaseResponse{
		Info: &v1.TestCaseInfo{
			Id:          testCase.ID(),
			Description: testCase.Description(),
		},
		Data: &v1.TestCaseData{
			Id:    requestID, // Use the unique request ID
			Input: encodedInput,
		},
	}, nil
}

func (s *Server) SubmitSolution(ctx context.Context, request *v1.SubmitSolutionRequest) (*v1.SubmitSolutionResponse, error) {
	// The TestCaseId in the request is actually the requestID
	s.requestsMutex.RLock()
	testCaseRequest, exists := s.activeRequests[request.TestCaseId]
	s.requestsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("request ID %s not found", request.TestCaseId)
	}

	// Get the test case
	testCase, err := s.registry.Get(testCaseRequest.TestCaseID)
	if err != nil {
		return nil, err
	}

	// Validate the response using the stored input
	result, err := testCase.ValidateResponse(testCaseRequest.GeneratedInput, request.Response)
	if err != nil {
		return nil, err
	}

	// Clean up the request after validation
	s.requestsMutex.Lock()
	delete(s.activeRequests, request.TestCaseId)
	s.requestsMutex.Unlock()

	return &v1.SubmitSolutionResponse{
		Valid:   result.Valid,
		Score:   result.Score,
		Message: result.Message,
	}, nil
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return err
	}

	s.grpcServer = grpc.NewServer()
	v1.RegisterTestingServiceServer(s.grpcServer, s)
	if err = s.grpcServer.Serve(lis); err != nil {
		return err
	}

	return nil
}

func (s *Server) Stop() error {
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}
	return nil
}

func (s *Server) RegisterTestCase(testCase types.TestCase) error {
	return s.registry.Register(testCase)
}

func (s *Server) GetRegistry() types.TestRegistry {
	return s.registry
}

// generateRequestID creates a unique request ID
func (s *Server) generateRequestID() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// Config holds server configuration
type Config struct {
	Address            string
	TLSEnabled         bool
	CertFile           string
	KeyFile            string
	MaxConcurrentTests int
	RequestTimeout     time.Duration
	EnableMetrics      bool
	EnableHTTPGateway  bool
}

// Middleware for extensibility
type Middleware interface {
	Process(ctx context.Context, req interface{}) (context.Context, error)
}

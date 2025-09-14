package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"

	"github.com/jalphad/testforge"
	v1 "github.com/jalphad/testforge/proto"
	"github.com/jalphad/testforge/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Create server configuration
	config := &testforge.Config{
		Address:    ":50051",
		TLSEnabled: false,
	}

	// Create and configure server
	server := testforge.NewServer(config)

	// Create a sorting test case with random input generation
	sortingTestCase := types.NewCase(
		"sort-integers",
		"Sort a list of integers in ascending order",
		func() []int {
			// Generate random integers each time
			size := 5 + rand.Intn(6) // 5-10 elements
			nums := make([]int, size)
			for i := range nums {
				nums[i] = rand.Intn(100) // 0-99
			}
			return nums
		},
		func(input []int) ([]byte, error) {
			return json.Marshal(input)
		},
		func(response []byte) ([]int, error) {
			var result []int
			err := json.Unmarshal(response, &result)
			return result, err
		},
		func(input []int, response []int) (*types.ValidationResult, error) {
			// Create expected result from the original input
			expected := make([]int, len(input))
			copy(expected, input)
			sort.Ints(expected)

			if len(response) != len(expected) {
				return &types.ValidationResult{
					Valid:   false,
					Score:   0.0,
					Message: fmt.Sprintf("Expected %d elements, got %d", len(expected), len(response)),
				}, nil
			}

			for i, v := range expected {
				if i >= len(response) || response[i] != v {
					return &types.ValidationResult{
						Valid:   false,
						Score:   0.0,
						Message: fmt.Sprintf("Expected %v, got %v", expected, response),
					}, nil
				}
			}

			return &types.ValidationResult{
				Valid:   true,
				Score:   100.0,
				Message: "Correctly sorted!",
			}, nil
		},
	)

	// Register the test case
	err := server.RegisterTestCase(sortingTestCase)
	if err != nil {
		log.Fatalf("Failed to register test case: %v", err)
	}

	// Start server in a goroutine
	serverDone := make(chan error, 1)
	go func() {
		log.Println("Starting server on :50051...")
		serverDone <- server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create client connection
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	client := v1.NewTestingServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// List available test cases
	listResp, err := client.ListTestCases(ctx, &v1.ListTestCasesRequest{})
	if err != nil {
		log.Fatalf("Failed to list test cases: %v", err)
	}

	fmt.Printf("Available test cases: %d\n", len(listResp.TestCases))
	for _, tc := range listResp.TestCases {
		fmt.Printf("- ID: %s, Description: %s\n", tc.Id, tc.Description)
	}

	// Get the specific test case
	getResp, err := client.GetTestCase(ctx, &v1.GetTestCaseRequest{
		Id: "sort-integers",
	})
	if err != nil {
		log.Fatalf("Failed to get test case: %v", err)
	}

	fmt.Printf("\nTest case: %s\n", getResp.Info.Description)

	// Parse the input
	var inputInts []int
	err = json.Unmarshal(getResp.Data.Input, &inputInts)
	if err != nil {
		log.Fatalf("Failed to parse input: %v", err)
	}
	fmt.Printf("Input: %v\n", inputInts)

	// Solve the problem (sort the integers)
	solution := make([]int, len(inputInts))
	copy(solution, inputInts)
	sort.Ints(solution)
	fmt.Printf("Solution: %v\n", solution)

	// Submit the solution
	solutionBytes, err := json.Marshal(solution)
	if err != nil {
		log.Fatalf("Failed to marshal solution: %v", err)
	}

	submitResp, err := client.SubmitSolution(ctx, &v1.SubmitSolutionRequest{
		TestCaseId: getResp.Data.Id, // Use the unique request ID from the GetTestCase response
		Response:   solutionBytes,
		ClientId:   "demo-client",
	})
	if err != nil {
		log.Fatalf("Failed to submit solution: %v", err)
	}

	// Print the result
	fmt.Printf("\nSolution Result:\n")
	fmt.Printf("Valid: %t\n", submitResp.Valid)
	fmt.Printf("Score: %.1f\n", submitResp.Score)
	fmt.Printf("Message: %s\n", submitResp.Message)

	// Stop the server
	fmt.Println("\nStopping server...")
	err = server.Stop()
	if err != nil {
		log.Fatalf("Failed to stop server: %v", err)
	}

	fmt.Println("Program completed successfully!")
}

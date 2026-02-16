package agent

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/tinywasm/mcpserve"
)

var testMemory MemoryStore
var testHandler *mcpserve.Handler

type testToolProvider struct{}

func (p testToolProvider) GetMCPToolsMetadata() []mcpserve.ToolMetadata {
	return []mcpserve.ToolMetadata{
		{
			Name:        "calculator",
			Description: "Calculates sum",
			Parameters: []mcpserve.ParameterMetadata{
				{
					Name:        "a",
					Description: "First number",
					Required:    true,
					Type:        "number",
				},
				{
					Name:        "b",
					Description: "Second number",
					Required:    true,
					Type:        "number",
				},
			},
			Execute: func(args map[string]any) {
				var a, b int
				// JSON unmarshals numbers as float64
				if v, ok := args["a"].(float64); ok {
					a = int(v)
				} else if v, ok := args["a"].(int); ok {
					a = v
				}
				if v, ok := args["b"].(float64); ok {
					b = int(v)
				} else if v, ok := args["b"].(int); ok {
					b = v
				}
				// Output result to stdout
				// mcpserve captures stdout and returns it as tool result content
				// or just use fmt.Print?
				// Let's assume just printing the value works.
				fmt.Printf("%d", a+b)
			},
		},
	}
}

func getFreePort() string {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return "8080"
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return "8080"
	}
	defer l.Close()
	return fmt.Sprintf("%d", l.Addr().(*net.TCPAddr).Port)
}

func TestMain(m *testing.M) {
	var err error
	testMemory, err = NewSQLiteMemory(":memory:")
	if err != nil {
		panic(err)
	}

	port := getFreePort()
	testHandler = mcpserve.NewHandler(
		mcpserve.Config{Port: port},
		[]mcpserve.ToolProvider{testToolProvider{}},
		nil,
		nil,
	)

	go testHandler.Serve()

	// Wait for server to start and URL be available
	// URL() should probably use config port if not updated
	// We might need to override URL logic or just wait

	for i := 0; i < 50; i++ {
		if testHandler.URL() != "" && testHandler.URL() != "http://localhost:0/mcp" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if testHandler.URL() == "" {
		fmt.Println("Warning: MCP Server URL not ready after timeout")
	} else {
		fmt.Printf("MCP Server listening on %s\n", testHandler.URL())
	}

	code := m.Run()

	testHandler.Stop()
	os.Exit(code)
}

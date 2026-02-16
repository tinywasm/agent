package agent

import (
	"os"
	"testing"
)

var testMemory MemoryStore

func TestMain(m *testing.M) {
	var err error
	testMemory, err = NewSQLiteMemory(":memory:")
	if err != nil {
		panic(err)
	}

	code := m.Run()
	os.Exit(code)
}

package mock_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/s00d/onlinesim-go-api/v2/mock"
)

func TestPersistState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	m, err := mock.NewBuilder().StatePath(path).Start()
	if err != nil {
		t.Fatal(err)
	}
	m.SetBalance(77, 2, 1)
	m.Close()

	m2, err := mock.NewBuilder().StatePath(path).Start()
	if err != nil {
		t.Fatal(err)
	}
	defer m2.Close()

	bal, err := m2.Client().User().Balance(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if bal.Balance != 77 {
		t.Fatalf("got %v", bal.Balance)
	}
}

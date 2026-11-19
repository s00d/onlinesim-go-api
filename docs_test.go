package onlinesim_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestDocsExamplesCompile builds the Go examples of README.md and docs/content/*.md against this module,
// so the docs cannot drift from the API. A ```go block that starts with `package` is a full program;
// any other block that uses `client` is a fragment, compiled inside
// func(ctx context.Context, client *onlinesim.Client, tzid int64) error. The rest are signatures.
func TestDocsExamplesCompile(t *testing.T) {
	if testing.Short() {
		t.Skip("runs the go toolchain")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not in PATH")
	}
	root, err := filepath.Abs(".")
	must(t, err)
	sources, err := filepath.Glob("docs/content/*.md")
	must(t, err)
	sources = append([]string{"README.md"}, sources...)

	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		path := filepath.Join(dir, name)
		must(t, os.MkdirAll(filepath.Dir(path), 0o755))
		must(t, os.WriteFile(path, []byte(content), 0o600))
	}
	write("go.mod", "module docsexamples\n\ngo 1.22\n\nrequire github.com/s00d/onlinesim-go-api/v3 v3.0.0\n\nreplace github.com/s00d/onlinesim-go-api/v3 => "+root+"\n")

	block := regexp.MustCompile("(?s)```go[^\n]*\n(.*?)```")
	usesClient := regexp.MustCompile(`\bclient\b`)
	count := 0
	for _, source := range sources {
		md, err := os.ReadFile(source)
		must(t, err)
		for i, m := range block.FindAllStringSubmatch(string(md), -1) {
			code := m[1]
			name := fmt.Sprintf("%s_%d", strings.NewReplacer("/", "_", ".", "_").Replace(source), i)
			switch {
			case strings.HasPrefix(code, "package "):
				write(filepath.Join("programs", name, "main.go"), code)
			case usesClient.MatchString(code):
				write(filepath.Join("fragments", name+".go"), fragment(name, code))
			default:
				continue
			}
			count++
		}
	}
	if count < 30 {
		t.Fatalf("found only %d examples", count)
	}

	cmd := exec.Command(goBin, "build", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOWORK=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("docs examples do not compile (%v):\n%s", err, out)
	}
}

func fragment(name, code string) string {
	return `package fragments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	onlinesim "github.com/s00d/onlinesim-go-api/v3"
	"github.com/s00d/onlinesim-go-api/v3/mock"
)

var (
	_ = context.Background
	_ = json.Marshal
	_ = errors.New
	_ = fmt.Sprint
	_ = io.EOF
	_ = log.Print
	_ = http.StatusOK
	_ = os.Getenv
	_ = time.Second
	_ = onlinesim.New
	_ = mock.Start
)

func ` + strings.ReplaceAll(name, "-", "_") + `(ctx context.Context, client *onlinesim.Client, tzid int64) error {
	{
` + code + `
	}
	return nil
}
`
}

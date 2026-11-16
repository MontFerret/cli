package debug

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MontFerret/cli/v2/cmd/internal/execution"
	"github.com/MontFerret/cli/v2/cmd/internal/testutil"
	"github.com/MontFerret/cli/v2/internal/testutil/wirehost"
	"github.com/MontFerret/ferret/v2/pkg/source"

	"github.com/MontFerret/cli/v2/pkg/browser"
	"github.com/MontFerret/cli/v2/pkg/build"
	"github.com/MontFerret/cli/v2/pkg/config"
	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

func TestDebugCommandRequiresExactlyOneScript(t *testing.T) {
	command := New(new(config.Store))

	if err := command.Args(command, nil); err == nil {
		t.Fatal("expected missing argument error")
	}
	if err := command.Args(command, []string{"one.fql", "two.fql"}); err == nil {
		t.Fatal("expected too many arguments error")
	}
	if err := command.Args(command, []string{"one.fql"}); err != nil {
		t.Fatalf("unexpected argument error: %v", err)
	}
}

func TestDebugUsesWireHostAndParameters(t *testing.T) {
	host := wirehost.New(t)
	path := filepath.Join(t.TempDir(), "script.fql")
	testutil.WriteQuery(t, path, "RETURN DEMO::IDENTITY(@limit)")
	store := testutil.NewStore(t)
	cmd := New(store)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	var out bytes.Buffer
	cmd.SetIn(strings.NewReader("c\nq\n"))
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--runtime=wire", "--runtime-endpoint=" + host.Endpoint, "--param=limit=10", path})
	err := cmd.ExecuteContext(config.With(t.Context(), store))
	if err != nil || host.Connections() != 1 || !strings.Contains(out.String(), "Result:\n10\n") {
		t.Fatalf("error=%v connections=%d stdout=%q", err, host.Connections(), out.String())
	}

	waitForDisconnect(t, host)
}

func TestExecuteDebugRejectsRemoteRuntimeBeforeStarting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "query.fql")
	testutil.WriteQuery(t, path, "RETURN 1")

	err := execute(
		testutil.NewCommand(),
		cliruntime.Options{Type: "https://worker.example"},
		browser.Options{},
		nil,
		[]string{path},
	)
	if !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("expected unsupported Worker error, got %v", err)
	}
	if got := err.Error(); !strings.Contains(got, "use builtin or Wire") {
		t.Fatalf("unexpected unsupported Worker error: %q", got)
	}
}

func TestExecuteDebugRejectsArtifact(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "query.fql")
	artifactPath := filepath.Join(dir, "query.fqlc")
	testutil.WriteQuery(t, sourcePath, "RETURN 1")
	if err := build.WriteArtifact(t.Context(), testutil.NewCompiler(t), source.New(sourcePath, "RETURN 1"), artifactPath); err != nil {
		t.Fatal(err)
	}

	host := wirehost.New(t)
	for _, opts := range []cliruntime.Options{cliruntime.NewDefaultOptions(), {Type: "wire", Endpoint: host.Endpoint}} {
		err := execute(testutil.NewCommand(), opts, browser.Options{}, nil, []string{artifactPath})
		if err == nil || err.Error() != "debugging compiled artifacts is not supported yet; run debug with the original .fql source file" {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if host.Connections() != 0 {
		t.Fatal("compiled artifact opened a Wire connection")
	}
}

func TestDebugCommandUsesSharedParamFlags(t *testing.T) {
	command := New(new(config.Store))
	command.SetContext(config.With(context.Background(), new(config.Store)))

	if err := command.Flags().Set(execution.ParamFlag, "limit=2"); err != nil {
		t.Fatal(err)
	}
	values, err := command.Flags().GetStringArray(execution.ParamFlag)
	if err != nil {
		t.Fatal(err)
	}
	params, err := execution.ParseParams(values)
	if err != nil {
		t.Fatal(err)
	}
	if params["limit"] != float64(2) {
		t.Fatalf("unexpected params: %#v", params)
	}
}

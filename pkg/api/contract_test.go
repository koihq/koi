package api

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/stretchr/testify/require"
)

var contractRouter = sync.OnceValues(func() (routers.Router, error) {
	_, file, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(file), "..", "..", "docs", "openapi.yaml")

	doc, err := openapi3.NewLoader().LoadFromFile(path)
	if err != nil {
		return nil, err
	}

	if err := doc.Validate(context.Background()); err != nil {
		return nil, err
	}

	return gorillamux.NewRouter(doc)
})

// contractTransport validates every documented response against docs/openapi.yaml. Requests to
// routes the spec does not declare pass through: tests also probe removed and test-only routes.
type contractTransport struct {
	t    testing.TB
	base http.RoundTripper
}

func newContractTransport(t testing.TB) http.RoundTripper {
	t.Helper()

	return contractTransport{
		t:    t,
		base: http.DefaultTransport,
	}
}

func (c contractTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := c.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	router, err := contractRouter()
	if err != nil {
		c.t.Errorf("load OpenAPI contract: %v", err)
		return resp, nil
	}

	route, pathParams, err := router.FindRoute(req)
	if err != nil {
		return resp, nil
	}

	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}

	resp.Body = io.NopCloser(bytes.NewReader(body))

	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request:    req,
			PathParams: pathParams,
			Route:      route,
		},
		Status: resp.StatusCode,
		Header: resp.Header,
		Options: &openapi3filter.Options{
			IncludeResponseStatus: true,
			MultiError:            true,
		},
	}
	input.SetBodyBytes(body)

	if err := openapi3filter.ValidateResponse(req.Context(), input); err != nil {
		c.t.Errorf("%s %s → %d violates docs/openapi.yaml: %v", req.Method, req.URL.Path, resp.StatusCode, err)
	}

	return resp, nil
}

type contractRecorder struct {
	testing.TB
	violations []string
}

func (r *contractRecorder) Errorf(format string, args ...any) {
	r.violations = append(r.violations, fmt.Sprintf(format, args...))
}

// TestContractErrorCodesMatchDomain keeps the documented error and field codes equal to the
// constants the domain can return, so a new code cannot reach clients undocumented.
func TestContractErrorCodesMatchDomain(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	source := filepath.Join(filepath.Dir(file), "..", "domain", "error.go")

	parsed, err := parser.ParseFile(token.NewFileSet(), source, nil, 0)
	require.NoError(t, err)

	constants := map[string][]string{}
	ast.Inspect(parsed, func(node ast.Node) bool {
		spec, ok := node.(*ast.ValueSpec)
		if !ok || len(spec.Values) != 1 {
			return true
		}

		literal, ok := spec.Values[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}

		value, err := strconv.Unquote(literal.Value)
		require.NoError(t, err)

		for _, prefix := range []string{"Code", "Field"} {
			if strings.HasPrefix(spec.Names[0].Name, prefix) {
				constants[prefix] = append(constants[prefix], value)
			}
		}

		return true
	})

	doc, err := openapi3.NewLoader().LoadFromFile(filepath.Join(filepath.Dir(file), "..", "..", "docs", "openapi.yaml"))
	require.NoError(t, err)

	for prefix, schema := range map[string]string{"Code": "ErrorCode", "Field": "FieldCode"} {
		var documented []string
		for _, value := range doc.Components.Schemas[schema].Value.Enum {
			documented = append(documented, value.(string))
		}

		require.ElementsMatch(t, constants[prefix], documented, "domain %s* constants vs %s", prefix, schema)
	}
}

func TestContractTransportRejectsUndocumentedResponses(t *testing.T) {
	_, err := contractRouter()
	require.NoError(t, err)

	for _, tc := range []struct {
		name       string
		path       string
		body       string
		violations int
	}{
		{name: "documented", path: "/health", body: `{"status":"ok","db":"ok"}`},
		{name: "wrong shape", path: "/health", body: `{"status":1}`, violations: 1},
		{name: "undeclared route", path: "/test-only", body: `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(server.Close)

			recorder := &contractRecorder{TB: t}
			client := &http.Client{Transport: newContractTransport(recorder)}

			resp, err := client.Get(server.URL + tc.path)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())

			require.Len(t, recorder.violations, tc.violations, recorder.violations)
		})
	}
}

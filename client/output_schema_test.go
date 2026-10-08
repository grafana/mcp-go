package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListToolsPreservesNullableArrayOutputSchema(t *testing.T) {
	const schema = `{"type":["null","array"],"items":{"type":"object","properties":{"name":{"type":"string"}},"required":["name"]}}`
	const listing = `[{"name":"list_branches","inputSchema":{"type":"object"},"outputSchema":` + schema + `},{"name":"other","inputSchema":{"type":"object"}}]`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request)) {
			return
		}
		var result string
		switch request.Method {
		case "server/discover":
			result = `{"supportedVersions":["2026-07-28"],"capabilities":{"tools":{}},"serverInfo":{"name":"schema-test","version":"1"}}`
		case "tools/list":
			result = `{"tools":` + listing + `}`
		default:
			t.Errorf("unexpected method %s", request.Method)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err := fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, request.ID, result)
		assert.NoError(t, err)
	}))
	t.Cleanup(server.Close)
	client, err := NewStreamableHttpClient(server.URL)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, client.Close()) })
	require.NoError(t, client.Start(t.Context()))
	initialized, err := client.Initialize(t.Context(), mcp.InitializeRequest{})
	require.NoError(t, err)
	require.Equal(t, mcp.ProtocolVersion20260728, initialized.ProtocolVersion)
	result, err := client.ListTools(t.Context(), mcp.ListToolsRequest{})
	require.NoError(t, err)
	require.Len(t, result.Tools, 2)
	assert.Equal(t, "list_branches", result.Tools[0].Name)
	assert.JSONEq(t, schema, string(result.Tools[0].RawOutputSchema))
	assert.Equal(t, "other", result.Tools[1].Name)
	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	var cached mcp.ListToolsResult
	require.NoError(t, json.Unmarshal(encoded, &cached))
	assert.JSONEq(t, schema, string(cached.Tools[0].RawOutputSchema))
}

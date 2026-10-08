package mcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolUnmarshalRawOutputSchema(t *testing.T) {
	tests := []struct{ name, schema string }{
		{"nullable array", `{"type":["null","array"],"items":{"type":"object","properties":{"name":{"type":"string"}}}}`},
		{"nullable object", `{"type":["object","null"]}`},
		{"array", `{"type":"array","items":{"type":"string"}}`},
		{"reference", `{"$defs":{"result":{"type":"string"}},"$ref":"#/$defs/result"}`},
		{"composition", `{"oneOf":[{"type":"string"},{"type":"number"}]}`},
		{"object with extra keywords", `{"type":"object","minProperties":1,"$schema":"https://json-schema.org/draft/2020-12/schema"}`},
		{"empty schema", `{}`},
		{"true schema", `true`},
		{"false schema", `false`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := `{"tools":[{"name":"example","title":"Example","description":"A tool","_meta":{"custom":"value"},"defer_loading":true,"inputSchema":{"type":"object","$defs":{"arg":{"type":"string"}},"properties":{"value":{"$ref":"#/$defs/arg"}},"required":["value"]},"outputSchema":` + tt.schema + `}]}`
			var listing ListToolsResult
			require.NoError(t, json.Unmarshal([]byte(payload), &listing))
			require.Len(t, listing.Tools, 1)
			tool := listing.Tools[0]
			assert.Equal(t, "example", tool.Name)
			assert.Equal(t, "Example", tool.Title)
			assert.Equal(t, "A tool", tool.Description)
			assert.True(t, tool.DeferLoading)
			require.NotNil(t, tool.Meta)
			assert.JSONEq(t, tt.schema, string(tool.RawOutputSchema))
			assert.Empty(t, tool.OutputSchema.Type)
			assert.Equal(t, "object", tool.InputSchema.Type)
			assert.Contains(t, tool.InputSchema.Defs, "arg")
			assert.Equal(t, []string{"value"}, tool.InputSchema.PropertyOrder)
			encoded, err := json.Marshal(listing)
			require.NoError(t, err)
			var roundTrip ListToolsResult
			require.NoError(t, json.Unmarshal(encoded, &roundTrip))
			assert.Equal(t, tool, roundTrip.Tools[0])
			var wire struct {
				Tools []map[string]json.RawMessage `json:"tools"`
			}
			require.NoError(t, json.Unmarshal(encoded, &wire))
			assert.JSONEq(t, tt.schema, string(wire.Tools[0]["outputSchema"]))
		})
	}
}

func TestToolUnmarshalTypedOutputSchema(t *testing.T) {
	var tool Tool
	require.NoError(t, json.Unmarshal([]byte(`{"name":"example","inputSchema":{"type":"object"},"outputSchema":{"type":"object","properties":{"value":{"type":["string","null"]}},"required":["value"],"additionalProperties":false}}`), &tool))
	assert.Nil(t, tool.RawOutputSchema)
	assert.Equal(t, "object", tool.OutputSchema.Type)
	assert.Equal(t, []string{"value"}, tool.OutputSchema.Required)
	assert.Equal(t, false, tool.OutputSchema.AdditionalProperties)
	assert.Contains(t, tool.OutputSchema.Properties, "value")
	tool.OutputSchema.Required = nil
	encoded, err := json.Marshal(tool)
	require.NoError(t, err)
	var roundTrip Tool
	require.NoError(t, json.Unmarshal(encoded, &roundTrip))
	assert.Empty(t, roundTrip.OutputSchema.Required)
}

func TestToolUnmarshalClearsPreviousOutputSchema(t *testing.T) {
	var tool Tool
	for _, schema := range []string{`{"type":["null","array"]}`, `{"type":"object"}`, `null`} {
		require.NoError(t, json.Unmarshal([]byte(`{"name":"example","outputSchema":`+schema+`}`), &tool))
	}
	assert.Nil(t, tool.RawOutputSchema)
	assert.Empty(t, tool.OutputSchema.Type)
	encoded, err := json.Marshal(tool)
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(encoded, &fields))
	assert.NotContains(t, fields, "outputSchema")
}

func TestToolUnmarshalRejectsInvalidSchemas(t *testing.T) {
	for _, payload := range []string{
		`{"inputSchema":{"type":["object","null"]}}`,
		`{"outputSchema":42}`,
		`{"outputSchema":"object"}`,
		`{"outputSchema":[]}`,
	} {
		var tool Tool
		assert.Error(t, json.Unmarshal([]byte(payload), &tool))
	}
}

func TestToolUnmarshalTypedOutputDefinitions(t *testing.T) {
	var tool Tool
	require.NoError(t, json.Unmarshal([]byte(`{"name":"example","outputSchema":{"type":"object","definitions":{"result":{"type":"string"}},"properties":{"value":{"$ref":"#/definitions/result"}}}}`), &tool))
	assert.Nil(t, tool.RawOutputSchema)
	assert.Contains(t, tool.OutputSchema.Defs, "result")
	assert.Equal(t, map[string]any{"$ref": "#/$defs/result"}, tool.OutputSchema.Properties["value"])
}

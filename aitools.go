package ayinbox

import (
	"context"
	"encoding/json"
)

// Tool is one agent tool. With format "ayinbox" Name, Description and
// InputSchema are set; with the other formats Type and Function are set.
type Tool struct {
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"input_schema,omitempty"`
	Type        string         `json:"type,omitempty"`
	Function    *ToolFunction  `json:"function,omitempty"`
}

// ToolFunction is a tool in the function-calling formats.
type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ToolListParams filters GET /ai/tools.
type ToolListParams struct {
	// Format is ayinbox (default), openai, hermes or functions.
	Format string `url:"format"`
	// Toolsets limits the list, for example inbox, campaigns, contacts.
	Toolsets []string `url:"toolsets"`
}

// ToolResult is the answer of a tool call.
type ToolResult struct {
	Name string `json:"name"`
	// Result is the tool's JSON output, or a JSON string when the tool
	// returned text.
	Result json.RawMessage `json:"result"`
	// ResultText is the output as text.
	ResultText string `json:"result_text"`
	// Untrusted is true when Result holds text written by people outside the
	// workspace. Fence it as data before giving it to a model.
	Untrusted bool `json:"untrusted"`
}

// AIToolsService lists and calls the agent tools exposed over HTTP.
type AIToolsService struct{ c *Client }

// List returns the tools the credential's scopes allow (GET /ai/tools).
// Send-class tools are never listed. params may be nil.
func (s *AIToolsService) List(ctx context.Context, params *ToolListParams, opts ...RequestOption) ([]Tool, error) {
	var out struct {
		Data []Tool `json:"data"`
	}
	if _, err := s.c.Request(ctx, "GET", "/ai/tools", encodeQuery(params), nil, &out, opts...); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// Call runs one tool with its JSON arguments (POST /ai/tools/{name}/call).
// args may be nil for {}. Paid tools require an "idempotency_key" argument.
// The scope is the tool's own.
func (s *AIToolsService) Call(ctx context.Context, name string, args any, opts ...RequestOption) (*ToolResult, error) {
	if isNilValue(args) {
		args = map[string]any{}
	}
	var out struct {
		Data ToolResult `json:"data"`
	}
	if _, err := s.c.Request(ctx, "POST", pathf("/ai/tools/%s/call", name), nil, args, &out, opts...); err != nil {
		return nil, err
	}
	return &out.Data, nil
}

package mcpapi

import (
	"encoding/json"

	"github.com/seaskyland/openclaw4j-sandbox/internal/bashapi"
	"github.com/seaskyland/openclaw4j-sandbox/internal/fileapi"
)

type Service struct {
	fileService fileapi.Service
	bashService *bashapi.Service
}

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type ToolResult struct {
	Content []TextContent `json:"content"`
	IsError bool          `json:"isError"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id,omitempty"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func NewService(fileService fileapi.Service, bashService *bashapi.Service) Service {
	return Service{fileService: fileService, bashService: bashService}
}

func ListTools() []Tool {
	return []Tool{
		tool("file_read", "Read a text file from the sandbox workspace", []string{"file"}),
		tool("file_write", "Write a text file in the sandbox workspace", []string{"file", "content"}),
		tool("file_list", "List a sandbox workspace directory", []string{"path"}),
		tool("file_replace", "Replace text in a sandbox workspace file", []string{"file", "old_str", "new_str"}),
		tool("file_search", "Search a file with a regular expression", []string{"file", "regex"}),
		tool("sandbox_execute_bash", "Execute a shell command in the sandbox workspace", []string{"command"}),
	}
}

func (s Service) CallTool(name string, arguments map[string]any) ToolResult {
	var response any
	success := false

	switch name {
	case "file_read":
		req, err := decodeArgs[fileapi.ReadRequest](arguments)
		if err != nil {
			return toolError(err.Error())
		}
		res := s.fileService.Read(req)
		response = res
		success = res.Success
	case "file_write":
		req, err := decodeArgs[fileapi.WriteRequest](arguments)
		if err != nil {
			return toolError(err.Error())
		}
		res := s.fileService.Write(req)
		response = res
		success = res.Success
	case "file_list":
		req, err := decodeArgs[fileapi.ListRequest](arguments)
		if err != nil {
			return toolError(err.Error())
		}
		res := s.fileService.List(req)
		response = res
		success = res.Success
	case "file_replace":
		req, err := decodeArgs[fileapi.ReplaceRequest](arguments)
		if err != nil {
			return toolError(err.Error())
		}
		res := s.fileService.Replace(req)
		response = res
		success = res.Success
	case "file_search":
		req, err := decodeArgs[fileapi.SearchRequest](arguments)
		if err != nil {
			return toolError(err.Error())
		}
		res := s.fileService.Search(req)
		response = res
		success = res.Success
	case "sandbox_execute_bash":
		req, err := decodeArgs[bashapi.ExecRequest](arguments)
		if err != nil {
			return toolError(err.Error())
		}
		res := s.bashService.Exec(req)
		response = res
		success = res.Success
	default:
		return toolError("unknown MCP tool")
	}

	text, err := json.Marshal(response)
	if err != nil {
		return toolError(err.Error())
	}
	return ToolResult{
		Content: []TextContent{{Type: "text", Text: string(text)}},
		IsError: !success,
	}
}

func (s Service) HandleJSONRPC(req JSONRPCRequest) JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return rpcResult(req.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "openclaw4j-sandbox",
				"version": "0.1.0",
			},
		})
	case "tools/list":
		return rpcResult(req.ID, map[string]any{"tools": ListTools()})
	case "tools/call":
		var params toolsCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return rpcError(req.ID, -32602, err.Error())
		}
		return rpcResult(req.ID, s.CallTool(params.Name, params.Arguments))
	default:
		return rpcError(req.ID, -32601, "method not found")
	}
}

func tool(name string, description string, required []string) Tool {
	properties := map[string]any{}
	for _, field := range required {
		properties[field] = map[string]any{"type": "string"}
	}
	return Tool{
		Name:        name,
		Description: description,
		InputSchema: map[string]any{
			"type":       "object",
			"required":   required,
			"properties": properties,
		},
	}
}

func decodeArgs[T any](arguments map[string]any) (T, error) {
	var req T
	bytes, err := json.Marshal(arguments)
	if err != nil {
		return req, err
	}
	err = json.Unmarshal(bytes, &req)
	return req, err
}

func toolError(message string) ToolResult {
	text, _ := json.Marshal(map[string]any{
		"success": false,
		"message": message,
		"data": map[string]any{
			"error_type": "tool_error",
			"message":    message,
		},
	})
	return ToolResult{
		Content: []TextContent{{Type: "text", Text: string(text)}},
		IsError: true,
	}
}

func rpcResult(id any, result any) JSONRPCResponse {
	return JSONRPCResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func rpcError(id any, code int, message string) JSONRPCResponse {
	return JSONRPCResponse{JSONRPC: "2.0", ID: id, Error: &JSONRPCError{Code: code, Message: message}}
}

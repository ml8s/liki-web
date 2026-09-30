// Package llm provides the chat message model shared by protocol surfaces.
//
// Messages
//
//	Role                     — system / user / assistant / tool
//	RoleSystem, RoleUser, RoleAssistant, RoleTool
//	Message                  — chat message with optional tool calls
//	ToolCall                 — function call requested by LLM
//	FunctionCall             — function name + JSON arguments
//
// The DeepSeek streaming client was removed together with the retired
// standalone products; active chat runs through liki-agents.
package llm

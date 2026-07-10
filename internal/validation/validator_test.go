package validation

import (
	"testing"

	relayprovider "micro-one-api/domain/upstream/provider"
)

func TestValidateMessagesAllowsAssistantToolCallWithEmptyContent(t *testing.T) {
	err := ValidateMessages([]relayprovider.Message{
		{Role: "user", Content: "北京天气怎么样"},
		{
			Role:             "assistant",
			Content:          "",
			ReasoningContent: "Need to call the weather tool for Beijing.",
			ToolCalls: []relayprovider.ToolCall{{
				ID:   "call_weather_beijing",
				Type: "function",
				Function: relayprovider.ToolCallFunction{
					Name:      "get_current_weather",
					Arguments: `{"location":"Beijing"}`,
				},
			}},
		},
		{Role: "tool", Content: "Sunny 25C", ToolCallID: "call_weather_beijing"},
	})
	if err != nil {
		t.Fatalf("ValidateMessages() error = %v", err)
	}
}

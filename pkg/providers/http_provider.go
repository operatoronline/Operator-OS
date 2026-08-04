// Operator - Ultra-lightweight personal AI agent
// Operator OS — github.com/operatoronline/Operator-OS
// License: MIT
//
// Copyright (c) 2026 Operator contributors

package providers

import (
	"context"
	"time"

	"github.com/operatoronline/Operator-OS/pkg/providers/openai_compat"
)

type HTTPProvider struct {
	delegate *openai_compat.Provider
}

func NewHTTPProviderWithOptions(apiKey, apiBase, proxy string, opts ...openai_compat.Option) *HTTPProvider {
	return &HTTPProvider{
		delegate: openai_compat.NewProvider(apiKey, apiBase, proxy, opts...),
	}
}

func NewHTTPProvider(apiKey, apiBase, proxy string) *HTTPProvider {
	return NewHTTPProviderWithOptions(apiKey, apiBase, proxy)
}

func NewHTTPProviderWithMaxTokensField(apiKey, apiBase, proxy, maxTokensField string) *HTTPProvider {
	return NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(apiKey, apiBase, proxy, maxTokensField, 0)
}

func NewHTTPProviderWithMaxTokensFieldAndRequestTimeout(
	apiKey, apiBase, proxy, maxTokensField string,
	requestTimeoutSeconds int,
) *HTTPProvider {
	return NewHTTPProviderWithOptions(
		apiKey,
		apiBase,
		proxy,
		openai_compat.WithMaxTokensField(maxTokensField),
		openai_compat.WithRequestTimeout(time.Duration(requestTimeoutSeconds)*time.Second),
	)
}

func (p *HTTPProvider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	return p.delegate.Chat(ctx, messages, tools, model, options)
}

func (p *HTTPProvider) GetDefaultModel() string {
	return ""
}

package service

import (
	"sensecraft-cloud-terminal/internal/app"
	"sensecraft-cloud-terminal/internal/client"
)

// newAIHTTPClient builds the AI HTTP client: Access Key Basic → {ai-host}/openapi.
func newAIHTTPClient(ctx app.AICmdContext) *client.AIApiClient {
	return client.NewAIApiClientAccessKey(ctx.Access, ctx.SandBox)
}

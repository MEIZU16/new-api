package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestShouldRetrySkipsSafetyPolicyRejections(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "Your request was rejected by the safety system.",
		Code:    "moderation_blocked",
	}, http.StatusBadGateway)

	require.False(t, shouldRetry(ctx, err, 2))
}

func TestShouldRetryKeepsTransientUpstreamFailures(t *testing.T) {
	ctx, _ := gin.CreateTestContext(nil)
	err := types.WithOpenAIError(types.OpenAIError{
		Message: "upstream temporarily unavailable",
		Code:    "server_error",
	}, http.StatusServiceUnavailable)

	require.True(t, shouldRetry(ctx, err, 2))
}

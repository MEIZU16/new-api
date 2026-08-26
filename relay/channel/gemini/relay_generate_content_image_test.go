package gemini

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	hosttypes "github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGeminiGenerateContentImageHandlerReturnsOpenAIImages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	responseBody := `{
		"candidates": [
			{"content":{"parts":[{"text":"ignored"},{"inlineData":{"mimeType":"image/png","data":"aW1hZ2Ux"}}]}},
			{"content":{"parts":[{"inlineData":{"mimeType":"image/jpeg","data":"aW1hZ2Uy"}}]}}
		],
		"usageMetadata":{"promptTokenCount":12,"candidatesTokenCount":34,"totalTokenCount":46}
	}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(responseBody)),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gemini-3-pro-image"},
		PriceData:   hosttypes.PriceData{UsePrice: true},
	}
	info.PriceData.AddOtherRatio("n", 4)

	usage, apiError := GeminiGenerateContentImageHandler(c, info, resp)
	require.Nil(t, apiError)
	require.Equal(t, 12, usage.PromptTokens)
	require.Equal(t, 34, usage.CompletionTokens)
	require.Equal(t, float64(2), info.PriceData.OtherRatios()["n"])
	require.Equal(t, 2, common.GetContextKeyInt(c, constant.ContextKeyImageOutputCount))
	require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var response dto.ImageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.NotZero(t, response.Created)
	require.Equal(t, []dto.ImageData{
		{B64Json: "aW1hZ2Ux"},
		{B64Json: "aW1hZ2Uy"},
	}, response.Data)
	require.NotContains(t, recorder.Body.String(), "candidates")
	require.NotContains(t, recorder.Body.String(), "inlineData")
	require.NotContains(t, recorder.Body.String(), "revised_prompt")
	require.NotContains(t, recorder.Body.String(), `"url"`)
}

func TestGeminiGenerateContentImageHandlerCapsOutputAtPublicMaximum(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(
			`{"candidates":[{"content":{"parts":[` +
				`{"inlineData":{"mimeType":"image/png","data":"MQ=="}},` +
				`{"inlineData":{"mimeType":"image/png","data":"Mg=="}},` +
				`{"inlineData":{"mimeType":"image/png","data":"Mw=="}},` +
				`{"inlineData":{"mimeType":"image/png","data":"NA=="}},` +
				`{"inlineData":{"mimeType":"image/png","data":"NQ=="}}` +
				`]}}]}`,
		)),
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "gemini-3-pro-image"},
		PriceData:   hosttypes.PriceData{UsePrice: true},
	}

	_, apiError := GeminiGenerateContentImageHandler(c, info, resp)
	require.Nil(t, apiError)
	var response dto.ImageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Len(t, response.Data, dto.MaxImageN)
	require.Equal(t, float64(dto.MaxImageN), info.PriceData.OtherRatios()["n"])
}

func TestGeminiGenerateContentImageHandlerRejectsSafetyBlockWithoutRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(
			`{"candidates":[],"promptFeedback":{"blockReason":"SAFETY"}}`,
		)),
	}

	_, apiError := GeminiGenerateContentImageHandler(c, &relaycommon.RelayInfo{}, resp)
	require.NotNil(t, apiError)
	require.Equal(t, types.ErrorCodePromptBlocked, apiError.GetErrorCode())
	require.Equal(t, http.StatusBadRequest, apiError.StatusCode)
	require.True(t, types.IsSkipRetryError(apiError))
}

func TestGeminiGenerateContentImageHandlerTreatsPolicyFinishReasonsAsNonRetryable(t *testing.T) {
	for _, finishReason := range []string{"RECITATION", "SPII", "IMAGE_PROHIBITED_CONTENT"} {
		t.Run(finishReason, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
			resp := &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(strings.NewReader(
					`{"candidates":[{"content":{"parts":[]},"finishReason":"` + finishReason + `"}]}`,
				)),
			}

			_, apiError := GeminiGenerateContentImageHandler(c, &relaycommon.RelayInfo{}, resp)
			require.NotNil(t, apiError)
			require.Equal(t, types.ErrorCodePromptBlocked, apiError.GetErrorCode())
			require.True(t, types.IsSkipRetryError(apiError))
		})
	}
}

func TestGeminiGenerateContentImageHandlerReturnsRetryableErrorWhenNoImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body: io.NopCloser(strings.NewReader(
			`{"candidates":[{"content":{"parts":[{"text":"no image"}]},"finishReason":"STOP"}]}`,
		)),
	}

	_, apiError := GeminiGenerateContentImageHandler(c, &relaycommon.RelayInfo{}, resp)
	require.NotNil(t, apiError)
	require.Equal(t, http.StatusBadGateway, apiError.StatusCode)
	require.False(t, types.IsSkipRetryError(apiError))
}

package gemini

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type openAIImageB64Response struct {
	Created int64                  `json:"created"`
	Data    []openAIImageB64Result `json:"data"`
}

type openAIImageB64Result struct {
	B64JSON string `json:"b64_json"`
}

// GeminiGenerateContentImageHandler converts a Gemini generateContent image
// response into the stable OpenAI Images b64_json response shape.
func GeminiGenerateContentImageHandler(c *gin.Context, info *relaycommon.RelayInfo, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	defer service.CloseResponseBodyGracefully(resp)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	var geminiResponse dto.GeminiChatResponse
	if err := common.Unmarshal(responseBody, &geminiResponse); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusBadGateway)
	}

	blockReason := ""
	if geminiResponse.PromptFeedback != nil && geminiResponse.PromptFeedback.BlockReason != nil {
		blockReason = strings.TrimSpace(*geminiResponse.PromptFeedback.BlockReason)
	}

	openAIResponse := openAIImageB64Response{
		Created: common.GetTimestamp(),
		Data:    make([]openAIImageB64Result, 0, len(geminiResponse.Candidates)),
	}
	for _, candidate := range geminiResponse.Candidates {
		if candidate.FinishReason != nil && isGeminiImageSafetyReason(*candidate.FinishReason) {
			blockReason = strings.TrimSpace(*candidate.FinishReason)
		}
		for _, part := range candidate.Content.Parts {
			if len(openAIResponse.Data) >= dto.MaxImageN {
				break
			}
			if part.InlineData == nil || strings.TrimSpace(part.InlineData.Data) == "" {
				continue
			}
			if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(part.InlineData.MimeType)), "image/") {
				continue
			}
			openAIResponse.Data = append(openAIResponse.Data, openAIImageB64Result{
				B64JSON: part.InlineData.Data,
			})
		}
		if len(openAIResponse.Data) >= dto.MaxImageN {
			break
		}
	}

	if len(openAIResponse.Data) == 0 {
		if blockReason != "" {
			common.SetContextKey(c, constant.ContextKeyAdminRejectReason, "gemini_block_reason="+blockReason)
			return nil, types.NewOpenAIError(
				fmt.Errorf("prompt was blocked by the safety system: %s", blockReason),
				types.ErrorCodePromptBlocked,
				http.StatusBadRequest,
				types.ErrOptionWithSkipRetry(),
			)
		}
		return nil, types.NewOpenAIError(
			errors.New("upstream Gemini response contained no generated image"),
			types.ErrorCodeBadResponseBody,
			http.StatusBadGateway,
		)
	}

	if info != nil && info.PriceData.UsePrice && len(openAIResponse.Data) <= dto.MaxImageN {
		info.PriceData.AddOtherRatio("n", float64(len(openAIResponse.Data)))
	}
	usage := buildUsageFromGeminiResponse(c, info, &geminiResponse)

	jsonResponse, err := common.Marshal(openAIResponse)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if resp.Header == nil {
		resp.Header = make(http.Header)
	}
	resp.Header.Set("Content-Type", "application/json")
	service.IOCopyBytesGracefully(c, resp, jsonResponse)
	return &usage, nil
}

func isGeminiImageSafetyReason(reason string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(reason))
	return strings.Contains(normalized, "SAFETY") ||
		strings.Contains(normalized, "BLOCK") ||
		strings.Contains(normalized, "PROHIBITED") ||
		strings.Contains(normalized, "RECITATION") ||
		strings.Contains(normalized, "SPII")
}

package relayconvert

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/convmeta"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/QuantumNous/new-api/relaykit/types"
)

func init() {
	registerBuiltinRequestConverter(RequestConverterSpec{
		ID:      ConverterOpenAIImagesToGeminiContent,
		From:    types.RelayFormatOpenAIImage,
		To:      types.RelayFormatGemini,
		Quality: RequestConverterQualityGood,
		Convert: convertOpenAIImagesRequestToGemini,
	})
}

func convertOpenAIImagesRequestToGemini(_ context.Context, info convmeta.Meta, request any) (any, error) {
	imageRequest, ok := request.(*dto.ImageRequest)
	if !ok {
		if value, valueOK := request.(dto.ImageRequest); valueOK {
			imageRequest = &value
		}
	}
	if imageRequest == nil {
		return nil, fmt.Errorf("expected OpenAI Images request, got %T", request)
	}
	if strings.TrimSpace(imageRequest.Prompt) == "" {
		return nil, invalidOpenAIImageConversionError("prompt is required")
	}
	if imageRequest.Stream != nil && *imageRequest.Stream {
		return nil, invalidOpenAIImageConversionError("stream is not supported for Gemini image generation")
	}
	responseFormat := strings.ToLower(strings.TrimSpace(imageRequest.ResponseFormat))
	if responseFormat != "" && responseFormat != "b64_json" {
		return nil, invalidOpenAIImageConversionError("response_format must be b64_json when provided")
	}

	candidateCount := uint(1)
	if imageRequest.N != nil && *imageRequest.N > 0 {
		candidateCount = *imageRequest.N
	}
	if candidateCount > dto.MaxImageN {
		return nil, invalidOpenAIImageConversionError(
			fmt.Sprintf("n must be an integer between 1 and %d", dto.MaxImageN),
		)
	}

	parts := make([]dto.GeminiPart, 0, len(imageRequest.ReferenceImages)+1)
	for index, reference := range imageRequest.ReferenceImages {
		mimeType := strings.TrimSpace(reference.MimeType)
		if !strings.HasPrefix(strings.ToLower(mimeType), "image/") {
			return nil, invalidOpenAIImageConversionError(
				fmt.Sprintf("reference image %d has invalid MIME type", index+1),
			)
		}
		if strings.TrimSpace(reference.Data) == "" {
			return nil, invalidOpenAIImageConversionError(
				fmt.Sprintf("reference image %d is empty", index+1),
			)
		}
		parts = append(parts, dto.GeminiPart{
			InlineData: &dto.GeminiInlineData{
				MimeType: mimeType,
				Data:     reference.Data,
			},
		})
	}
	parts = append(parts, dto.GeminiPart{Text: imageRequest.Prompt})

	aspectRatio, err := openAIImageAspectRatio(imageRequest)
	if err != nil {
		return nil, invalidOpenAIImageConversionError(err.Error())
	}
	originalModel := imageRequest.Model
	if info != nil && strings.TrimSpace(info.GetOriginModelName()) != "" {
		originalModel = info.GetOriginModelName()
	}
	imageConfig := map[string]string{
		"imageSize": geminiImageResolutionForModel(originalModel),
	}
	if aspectRatio != "" {
		imageConfig["aspectRatio"] = aspectRatio
	}
	imageConfigJSON, err := kitutil.Marshal(imageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Gemini image config: %w", err)
	}

	count := int(candidateCount)
	return &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{
				Role:  "user",
				Parts: parts,
			},
		},
		GenerationConfig: dto.GeminiChatGenerationConfig{
			CandidateCount:     &count,
			ResponseModalities: []string{"IMAGE"},
			ImageConfig:        imageConfigJSON,
		},
	}, nil
}

func invalidOpenAIImageConversionError(message string) error {
	return types.NewErrorWithStatusCode(
		errors.New(message),
		types.ErrorCodeInvalidRequest,
		http.StatusBadRequest,
		types.ErrOptionWithSkipRetry(),
	)
}

func geminiImageResolutionForModel(model string) string {
	normalized := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasSuffix(normalized, "-4k"):
		return "4K"
	case strings.HasSuffix(normalized, "-2k"):
		return "2K"
	default:
		return "1K"
	}
}

func openAIImageAspectRatio(request *dto.ImageRequest) (string, error) {
	if len(request.ExtraFields) > 0 && string(request.ExtraFields) != "null" {
		var extraFields struct {
			AspectRatio string `json:"aspect_ratio"`
		}
		if err := kitutil.Unmarshal(request.ExtraFields, &extraFields); err != nil {
			return "", fmt.Errorf("extra_fields must be a JSON object: %w", err)
		}
		if strings.TrimSpace(extraFields.AspectRatio) != "" {
			return normalizeGeminiAspectRatio(extraFields.AspectRatio)
		}
	}

	size := strings.ToLower(strings.TrimSpace(request.Size))
	if size == "" || size == "auto" {
		return "", nil
	}
	switch size {
	case "1024x1792":
		return "9:16", nil
	case "1792x1024":
		return "16:9", nil
	}
	if strings.Contains(size, ":") {
		return normalizeGeminiAspectRatio(size)
	}

	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return "", fmt.Errorf("size must be a width-height pair or aspect ratio")
	}
	return normalizeGeminiAspectRatio(parts[0] + ":" + parts[1])
}

func normalizeGeminiAspectRatio(value string) (string, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("aspect_ratio must use width:height format")
	}
	width, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || width <= 0 {
		return "", fmt.Errorf("aspect_ratio width must be a positive integer")
	}
	height, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || height <= 0 {
		return "", fmt.Errorf("aspect_ratio height must be a positive integer")
	}

	divisor := width
	other := height
	for other != 0 {
		divisor, other = other, divisor%other
	}
	return fmt.Sprintf("%d:%d", width/divisor, height/divisor), nil
}

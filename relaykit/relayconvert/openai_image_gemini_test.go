package relayconvert

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/convmeta"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/stretchr/testify/require"
)

func TestOpenAIImagesToGeminiGenerateContent(t *testing.T) {
	n := uint(3)
	request := &dto.ImageRequest{
		Model:       "gemini-3-pro-image",
		Prompt:      "Use the references to create a cinematic landscape.",
		N:           &n,
		Quality:     "high",
		Size:        "4096x4096",
		ExtraFields: json.RawMessage(`{"aspect_ratio":"16:9"}`),
		ReferenceImages: []dto.ImageReference{
			{MimeType: "image/png", Data: "cG5n"},
			{MimeType: "image/jpeg", Data: "anBlZw=="},
		},
	}
	info := &convmeta.Values{
		OriginModelName:     "gemini-3-pro-image-2k",
		UpstreamModelName:   "gemini-3-pro-image",
		ChannelMetaAttached: true,
	}

	result, err := ConvertRequestByID(
		context.Background(),
		info,
		ConverterOpenAIImagesToGeminiContent,
		request,
	)
	require.NoError(t, err)
	require.Equal(t, types.RelayFormat(types.RelayFormatOpenAIImage), result.From)
	require.Equal(t, types.RelayFormat(types.RelayFormatGemini), result.To)

	converted, ok := result.Value.(*dto.GeminiChatRequest)
	require.True(t, ok)
	require.Len(t, converted.Contents, 1)
	require.Equal(t, "user", converted.Contents[0].Role)
	require.Len(t, converted.Contents[0].Parts, 3)
	require.Equal(t, "cG5n", converted.Contents[0].Parts[0].InlineData.Data)
	require.Equal(t, "image/png", converted.Contents[0].Parts[0].InlineData.MimeType)
	require.Equal(t, "anBlZw==", converted.Contents[0].Parts[1].InlineData.Data)
	require.Equal(t, request.Prompt, converted.Contents[0].Parts[2].Text)
	require.Equal(t, []string{"IMAGE"}, converted.GenerationConfig.ResponseModalities)
	require.NotNil(t, converted.GenerationConfig.CandidateCount)
	require.Equal(t, 3, *converted.GenerationConfig.CandidateCount)

	var imageConfig struct {
		AspectRatio string `json:"aspectRatio"`
		ImageSize   string `json:"imageSize"`
	}
	require.NoError(t, kitutil.Unmarshal(converted.GenerationConfig.ImageConfig, &imageConfig))
	require.Equal(t, "16:9", imageConfig.AspectRatio)
	require.Equal(t, "2K", imageConfig.ImageSize)
}

func TestOpenAIImagesToGeminiLocksResolutionToOriginalModel(t *testing.T) {
	testCases := []struct {
		name          string
		originalModel string
		size          string
		quality       string
		wantSize      string
		wantRatio     string
	}{
		{
			name:          "base SKU stays 1K despite large client size and high quality",
			originalModel: "gemini-3.1-flash-image",
			size:          "4096x4096",
			quality:       "high",
			wantSize:      "1K",
			wantRatio:     "1:1",
		},
		{
			name:          "2K SKU stays 2K",
			originalModel: "gemini-3.1-flash-image-2k",
			size:          "1792x1024",
			quality:       "low",
			wantSize:      "2K",
			wantRatio:     "16:9",
		},
		{
			name:          "4K SKU stays 4K",
			originalModel: "gemini-3-pro-image-4k",
			size:          "1024x1792",
			quality:       "standard",
			wantSize:      "4K",
			wantRatio:     "9:16",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			n := uint(1)
			result, err := ConvertRequestByID(
				context.Background(),
				&convmeta.Values{OriginModelName: testCase.originalModel},
				ConverterOpenAIImagesToGeminiContent,
				&dto.ImageRequest{
					Model:   "mapped-upstream-model",
					Prompt:  "draw a lighthouse",
					N:       &n,
					Size:    testCase.size,
					Quality: testCase.quality,
				},
			)
			require.NoError(t, err)
			converted := result.Value.(*dto.GeminiChatRequest)

			var imageConfig struct {
				AspectRatio string `json:"aspectRatio"`
				ImageSize   string `json:"imageSize"`
			}
			require.NoError(t, kitutil.Unmarshal(converted.GenerationConfig.ImageConfig, &imageConfig))
			require.Equal(t, testCase.wantSize, imageConfig.ImageSize)
			require.Equal(t, testCase.wantRatio, imageConfig.AspectRatio)
		})
	}
}

func TestOpenAIImagesToGeminiRejectsInvalidAspectRatio(t *testing.T) {
	_, err := ConvertRequestByID(
		context.Background(),
		&convmeta.Values{OriginModelName: "gemini-3-pro-image"},
		ConverterOpenAIImagesToGeminiContent,
		&dto.ImageRequest{
			Model:       "gemini-3-pro-image",
			Prompt:      "draw a cat",
			ExtraFields: json.RawMessage(`{"aspect_ratio":"16:0"}`),
		},
	)
	require.Error(t, err)

	var apiError *types.NewAPIError
	require.ErrorAs(t, err, &apiError)
	require.Equal(t, http.StatusBadRequest, apiError.StatusCode)
	require.True(t, types.IsSkipRetryError(apiError))
}

func TestOpenAIImagesToGeminiRejectsStreaming(t *testing.T) {
	stream := true
	_, err := ConvertRequestByID(
		context.Background(),
		&convmeta.Values{OriginModelName: "gemini-3-pro-image"},
		ConverterOpenAIImagesToGeminiContent,
		&dto.ImageRequest{
			Model:  "gemini-3-pro-image",
			Prompt: "draw a cat",
			Stream: &stream,
		},
	)
	require.Error(t, err)

	var apiError *types.NewAPIError
	require.ErrorAs(t, err, &apiError)
	require.Equal(t, http.StatusBadRequest, apiError.StatusCode)
	require.True(t, types.IsSkipRetryError(apiError))
}

func TestOpenAIImagesToGeminiAcceptsOnlyBase64ResponseFormat(t *testing.T) {
	for _, responseFormat := range []string{"", "b64_json", " B64_JSON "} {
		t.Run("accepts_"+responseFormat, func(t *testing.T) {
			_, err := ConvertRequestByID(
				context.Background(),
				&convmeta.Values{OriginModelName: "gemini-3-pro-image"},
				ConverterOpenAIImagesToGeminiContent,
				&dto.ImageRequest{
					Model:          "gemini-3-pro-image",
					Prompt:         "draw a cat",
					ResponseFormat: responseFormat,
				},
			)
			require.NoError(t, err)
		})
	}

	_, err := ConvertRequestByID(
		context.Background(),
		&convmeta.Values{OriginModelName: "gemini-3-pro-image"},
		ConverterOpenAIImagesToGeminiContent,
		&dto.ImageRequest{
			Model:          "gemini-3-pro-image",
			Prompt:         "draw a cat",
			ResponseFormat: "url",
		},
	)
	require.Error(t, err)

	var apiError *types.NewAPIError
	require.ErrorAs(t, err, &apiError)
	require.Equal(t, http.StatusBadRequest, apiError.StatusCode)
	require.True(t, types.IsSkipRetryError(apiError))
}

func TestOpenAIImagesToGeminiRejectsExcessiveCandidateCount(t *testing.T) {
	n := uint(dto.MaxImageN + 1)
	_, err := ConvertRequestByID(
		context.Background(),
		&convmeta.Values{OriginModelName: "gemini-3-pro-image"},
		ConverterOpenAIImagesToGeminiContent,
		&dto.ImageRequest{Model: "gemini-3-pro-image", Prompt: "draw a cat", N: &n},
	)
	require.Error(t, err)

	var apiError *types.NewAPIError
	require.ErrorAs(t, err, &apiError)
	require.Equal(t, http.StatusBadRequest, apiError.StatusCode)
	require.True(t, types.IsSkipRetryError(apiError))
}

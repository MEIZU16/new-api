package advancedcustom

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	relayhelper "github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdaptorConvertsOpenAIImageGenerationToGemini(t *testing.T) {
	adaptor := &Adaptor{}
	info := advancedCustomImageRelayInfo("/v1/images/generations")
	c := advancedCustomGinContext("/v1/images/generations")
	n := uint(2)

	converted, err := adaptor.ConvertImageRequest(c, info, dto.ImageRequest{
		Model:       "gemini-3-pro-image",
		Prompt:      "draw a red panda",
		N:           &n,
		ExtraFields: mustAdvancedCustomRawMessage(t, map[string]any{"aspect_ratio": "3:2"}),
	})
	require.NoError(t, err)

	geminiRequest, ok := converted.(*dto.GeminiChatRequest)
	require.True(t, ok)
	require.Equal(t, "draw a red panda", geminiRequest.Contents[0].Parts[0].Text)
	require.Equal(t, []string{"IMAGE"}, geminiRequest.GenerationConfig.ResponseModalities)
	require.Equal(t, 2, *geminiRequest.GenerationConfig.CandidateCount)

	var imageConfig map[string]string
	require.NoError(t, kitutil.Unmarshal(geminiRequest.GenerationConfig.ImageConfig, &imageConfig))
	require.Equal(t, "3:2", imageConfig["aspectRatio"])
	require.Equal(t, "4K", imageConfig["imageSize"])
}

func TestAdaptorConvertsMultipartImageEditToGeminiJSON(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gemini-3-pro-image-4k"))
	require.NoError(t, writer.WriteField("prompt", "turn this into a watercolor"))
	require.NoError(t, writer.WriteField("extra_fields", `{"aspect_ratio":"4:3"}`))

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="image"; filename="reference.jpg"`)
	header.Set("Content-Type", "image/jpeg")
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	imageBytes := mustEncodeAdvancedCustomTestImage(t, "jpeg")
	_, err = part.Write(imageBytes)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	c, _ := newAdvancedCustomTestContext(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	imageRequest, err := relayhelper.GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
	require.NoError(t, err)

	adaptor := &Adaptor{}
	info := advancedCustomImageRelayInfo("/v1/images/edits")
	info.RelayMode = relayconstant.RelayModeImagesEdits
	converted, err := adaptor.ConvertImageRequest(c, info, *imageRequest)
	require.NoError(t, err)

	geminiRequest, ok := converted.(*dto.GeminiChatRequest)
	require.True(t, ok)
	require.Len(t, geminiRequest.Contents, 1)
	require.Len(t, geminiRequest.Contents[0].Parts, 2)
	require.NotNil(t, geminiRequest.Contents[0].Parts[0].InlineData)
	require.Equal(t, "image/jpeg", geminiRequest.Contents[0].Parts[0].InlineData.MimeType)
	require.Equal(t, base64.StdEncoding.EncodeToString(imageBytes), geminiRequest.Contents[0].Parts[0].InlineData.Data)
	require.Equal(t, "turn this into a watercolor", geminiRequest.Contents[0].Parts[1].Text)
	var imageConfig map[string]string
	require.NoError(t, kitutil.Unmarshal(geminiRequest.GenerationConfig.ImageConfig, &imageConfig))
	require.Equal(t, "4:3", imageConfig["aspectRatio"])
	require.Equal(t, "4K", imageConfig["imageSize"])

	outboundHeader := http.Header{}
	require.NoError(t, adaptor.SetupRequestHeader(c, &outboundHeader, info))
	require.Equal(t, "application/json", outboundHeader.Get("Content-Type"))
}

func TestAdaptorRejectsUnsupportedGeminiImageEditInputs(t *testing.T) {
	t.Run("JSON edits require multipart", func(t *testing.T) {
		c, _ := newAdvancedCustomTestContext(
			http.MethodPost,
			"/v1/images/edits",
			strings.NewReader(`{"model":"gemini-3-pro-image","prompt":"edit this"}`),
		)
		c.Request.Header.Set("Content-Type", "application/json")
		info := advancedCustomImageRelayInfo("/v1/images/edits")
		info.RelayMode = relayconstant.RelayModeImagesEdits

		_, err := (&Adaptor{}).ConvertImageRequest(c, info, dto.ImageRequest{
			Model:  "gemini-3-pro-image",
			Prompt: "edit this",
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "require multipart/form-data")
		require.True(t, types.IsSkipRetryError(types.NewError(err, types.ErrorCodeConvertRequestFailed)))
	})

	for _, testCase := range []struct {
		name      string
		withImage bool
		withMask  bool
		wantError string
	}{
		{name: "missing image", wantError: "image is required"},
		{name: "mask is unsupported", withImage: true, withMask: true, wantError: "mask is not supported"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			require.NoError(t, writer.WriteField("model", "gemini-3-pro-image"))
			require.NoError(t, writer.WriteField("prompt", "edit this"))
			if testCase.withImage {
				part, err := writer.CreateFormFile("image", "reference.png")
				require.NoError(t, err)
				_, err = part.Write([]byte("png-bytes"))
				require.NoError(t, err)
			}
			if testCase.withMask {
				part, err := writer.CreateFormFile("mask", "mask.png")
				require.NoError(t, err)
				_, err = part.Write([]byte("mask-bytes"))
				require.NoError(t, err)
			}
			require.NoError(t, writer.Close())

			c, _ := newAdvancedCustomTestContext(http.MethodPost, "/v1/images/edits", &body)
			c.Request.Header.Set("Content-Type", writer.FormDataContentType())
			request, err := relayhelper.GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
			require.NoError(t, err)
			info := advancedCustomImageRelayInfo("/v1/images/edits")
			info.RelayMode = relayconstant.RelayModeImagesEdits

			_, err = (&Adaptor{}).ConvertImageRequest(c, info, *request)
			require.Error(t, err)
			require.Contains(t, err.Error(), testCase.wantError)
		})
	}
}

func TestAdaptorRejectsSpoofedGeminiReferenceImageMimeType(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gemini-3-pro-image"))
	require.NoError(t, writer.WriteField("prompt", "edit this"))
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="image"; filename="not-an-image.jpg"`)
	header.Set("Content-Type", "image/jpeg")
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write([]byte("this is not an image"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	c, _ := newAdvancedCustomTestContext(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	request, err := relayhelper.GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
	require.NoError(t, err)
	info := advancedCustomImageRelayInfo("/v1/images/edits")
	info.RelayMode = relayconstant.RelayModeImagesEdits

	_, err = (&Adaptor{}).ConvertImageRequest(c, info, *request)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be a valid PNG, JPEG, or WebP image")
	apiError, ok := err.(*types.NewAPIError)
	require.True(t, ok)
	require.Equal(t, http.StatusBadRequest, apiError.StatusCode)
	require.True(t, types.IsSkipRetryError(apiError))
}

func TestAdaptorAcceptsMoreThanFourGeminiReferenceImages(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gemini-3-pro-image"))
	require.NoError(t, writer.WriteField("prompt", "edit this"))
	imageBytes := mustEncodeAdvancedCustomTestImage(t, "png")
	for range 5 {
		part, err := writer.CreateFormFile("image[]", "reference.png")
		require.NoError(t, err)
		_, err = part.Write(imageBytes)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	c, _ := newAdvancedCustomTestContext(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	request, err := relayhelper.GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
	require.NoError(t, err)
	info := advancedCustomImageRelayInfo("/v1/images/edits")
	info.RelayMode = relayconstant.RelayModeImagesEdits

	converted, err := (&Adaptor{}).ConvertImageRequest(c, info, *request)
	require.NoError(t, err)
	geminiRequest, ok := converted.(*dto.GeminiChatRequest)
	require.True(t, ok)
	require.Len(t, geminiRequest.Contents[0].Parts, 6)
}

func TestAdaptorRejectsGeminiReferenceImageByteLimits(t *testing.T) {
	t.Run("actual bytes exceed reported size", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.WriteField("model", "gemini-3-pro-image"))
		require.NoError(t, writer.WriteField("prompt", "edit this"))
		part, err := writer.CreateFormFile("image", "reference.png")
		require.NoError(t, err)
		_, err = io.CopyN(part, advancedCustomRepeatingReader{}, maxGeminiReferenceImageBytes+1)
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		c, _ := newAdvancedCustomTestContext(http.MethodPost, "/v1/images/edits", &body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		request, err := relayhelper.GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
		require.NoError(t, err)
		defer common.CleanupBodyStorage(c)
		defer c.Request.MultipartForm.RemoveAll()
		require.Len(t, c.Request.MultipartForm.File["image"], 1)
		c.Request.MultipartForm.File["image"][0].Size = 0
		info := advancedCustomImageRelayInfo("/v1/images/edits")
		info.RelayMode = relayconstant.RelayModeImagesEdits

		_, err = (&Adaptor{}).ConvertImageRequest(c, info, *request)
		require.Error(t, err)
		require.Contains(t, err.Error(), "exceeds the 16 MiB limit")
		apiError, ok := err.(*types.NewAPIError)
		require.True(t, ok)
		require.Equal(t, http.StatusRequestEntityTooLarge, apiError.StatusCode)
		require.True(t, types.IsSkipRetryError(apiError))
	})

	for _, testCase := range []struct {
		name  string
		sizes []int64
		want  string
	}{
		{
			name:  "per image limit",
			sizes: []int64{maxGeminiReferenceImageBytes + 1},
			want:  "exceeds the 16 MiB limit",
		},
		{
			name: "aggregate limit",
			sizes: []int64{
				maxGeminiReferenceImageBytes,
				maxGeminiReferenceImageBytes,
				1,
			},
			want: "exceed the 32 MiB aggregate limit",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			c, _ := newAdvancedCustomTestContext(http.MethodPost, "/v1/images/edits", nil)
			c.Request.Header.Set("Content-Type", "multipart/form-data; boundary=test")
			files := make([]*multipart.FileHeader, 0, len(testCase.sizes))
			for _, size := range testCase.sizes {
				files = append(files, &multipart.FileHeader{Filename: "reference.png", Size: size})
			}
			c.Request.MultipartForm = &multipart.Form{File: map[string][]*multipart.FileHeader{"image": files}}
			info := advancedCustomImageRelayInfo("/v1/images/edits")
			info.RelayMode = relayconstant.RelayModeImagesEdits

			_, err := (&Adaptor{}).ConvertImageRequest(c, info, dto.ImageRequest{
				Model:  "gemini-3-pro-image",
				Prompt: "edit this",
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), testCase.want)
			apiError, ok := err.(*types.NewAPIError)
			require.True(t, ok)
			require.Equal(t, http.StatusRequestEntityTooLarge, apiError.StatusCode)
			require.True(t, types.IsSkipRetryError(apiError))
		})
	}
}

func TestAdaptorSendsConvertedMultipartEditAsJSON(t *testing.T) {
	var receivedContentType string
	var receivedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		receivedContentType = request.Header.Get("Content-Type")
		var err error
		receivedBody, err = io.ReadAll(request.Body)
		require.NoError(t, err)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/png","data":"b3V0"}}]}}]}`))
	}))
	defer server.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gemini-3.1-flash-image"))
	require.NoError(t, writer.WriteField("prompt", "edit this"))
	part, err := writer.CreateFormFile("image", "reference.png")
	require.NoError(t, err)
	_, err = part.Write(mustEncodeAdvancedCustomTestImage(t, "png"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	c, recorder := newAdvancedCustomTestContext(http.MethodPost, "/v1/images/edits", &body)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	imageRequest, err := relayhelper.GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
	require.NoError(t, err)

	adaptor := &Adaptor{}
	info := advancedCustomImageRelayInfo("/v1/images/edits")
	info.ChannelBaseUrl = server.URL
	info.RelayMode = relayconstant.RelayModeImagesEdits
	converted, err := adaptor.ConvertImageRequest(c, info, *imageRequest)
	require.NoError(t, err)
	payload, err := common.Marshal(converted)
	require.NoError(t, err)

	response, err := adaptor.DoRequest(c, info, bytes.NewReader(payload))
	require.NoError(t, err)
	require.Equal(t, "application/json", receivedContentType)

	var upstream dto.GeminiChatRequest
	require.NoError(t, common.Unmarshal(receivedBody, &upstream))
	require.Len(t, upstream.Contents[0].Parts, 2)

	usage, apiError := adaptor.DoResponse(c, response.(*http.Response), info)
	require.Nil(t, apiError)
	require.IsType(t, &dto.Usage{}, usage)
	var openAIResponse dto.ImageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &openAIResponse))
	require.Equal(t, "b3V0", openAIResponse.Data[0].B64Json)
	require.NotContains(t, recorder.Body.String(), "inlineData")
}

func advancedCustomImageRelayInfo(incomingPath string) *relaycommon.RelayInfo {
	info := advancedCustomRelayInfo(&dto.AdvancedCustomConfig{
		Routes: []dto.AdvancedCustomRoute{
			{
				IncomingPath: incomingPath,
				UpstreamPath: "/v1beta/models/{model}:generateContent",
				Converter:    relayconvert.ConverterOpenAIImagesToGeminiContent,
			},
		},
	})
	info.RelayFormat = types.RelayFormatOpenAIImage
	info.RelayMode = relayconstant.RelayModeImagesGenerations
	info.RequestURLPath = incomingPath
	info.OriginModelName = "gemini-3-pro-image-4k"
	info.UpstreamModelName = "gemini-3-pro-image"
	return info
}

func newAdvancedCustomTestContext(method string, path string, body io.Reader) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(method, path, body)
	return c, recorder
}

type advancedCustomRepeatingReader struct{}

func (advancedCustomRepeatingReader) Read(data []byte) (int, error) {
	for index := range data {
		data[index] = 'a'
	}
	return len(data), nil
}

func mustEncodeAdvancedCustomTestImage(t *testing.T, format string) []byte {
	t.Helper()
	var data bytes.Buffer
	imageData := image.NewRGBA(image.Rect(0, 0, 1, 1))
	switch format {
	case "jpeg":
		require.NoError(t, jpeg.Encode(&data, imageData, nil))
	case "png":
		require.NoError(t, png.Encode(&data, imageData))
	default:
		t.Fatalf("unsupported test image format %q", format)
	}
	return data.Bytes()
}

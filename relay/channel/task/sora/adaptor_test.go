package sora

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSoraBuildRequestBodyReturnsReplayablePassThroughBody(t *testing.T) {
	payload := []byte("opaque-sora-request-body")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/octet-stream")
	defer common.CleanupBodyStorage(c)

	info := &relaycommon.RelayInfo{}
	body, err := (&TaskAdaptor{}).BuildRequestBody(c, info)
	require.NoError(t, err)
	replayable, ok := body.(common.ReplayableBody)
	require.True(t, ok)

	sent, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, payload, sent)
	assert.EqualValues(t, len(payload), replayable.Size())

	replayBody, err := replayable.NewReader()
	require.NoError(t, err)
	replay, err := io.ReadAll(replayBody)
	require.NoError(t, err)
	require.NoError(t, replayBody.Close())
	assert.Equal(t, payload, replay)
}

func TestSoraValidationStoresResolvedDefaults(t *testing.T) {
	payload := []byte(`{"model":"omni-flash","prompt":"a cat surfing"}`)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	defer common.CleanupBodyStorage(c)

	info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info)
	require.Nil(t, taskErr)
	request, err := relaycommon.GetTaskRequest(c)
	require.NoError(t, err)
	assert.Equal(t, "720x1280", request.Size)
	assert.Equal(t, "4", request.Seconds)
	assert.Zero(t, request.Duration)
}

func TestSoraBuildRequestBodyWritesResolvedJSONDefaults(t *testing.T) {
	payload := []byte(`{"model":"omni-flash","prompt":"a cat surfing"}`)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("task_request", relaycommon.TaskSubmitReq{Prompt: "a cat surfing", Model: "omni-flash"})
	defer common.CleanupBodyStorage(c)

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "flow/omni"},
	}
	body, err := (&TaskAdaptor{}).BuildRequestBody(c, info)
	require.NoError(t, err)
	encoded, err := io.ReadAll(body)
	require.NoError(t, err)

	var request map[string]any
	require.NoError(t, json.Unmarshal(encoded, &request))
	assert.Equal(t, "flow/omni", request["model"])
	assert.Equal(t, "720x1280", request["size"])
	assert.Equal(t, "4", request["seconds"])
}

func TestSoraBuildRequestBodyWritesResolvedMultipartDefaults(t *testing.T) {
	var input bytes.Buffer
	inputWriter := multipart.NewWriter(&input)
	require.NoError(t, inputWriter.WriteField("model", "omni-flash"))
	require.NoError(t, inputWriter.WriteField("prompt", "a cat surfing"))
	require.NoError(t, inputWriter.Close())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(input.Bytes()))
	c.Request.Header.Set("Content-Type", inputWriter.FormDataContentType())
	c.Set("task_request", relaycommon.TaskSubmitReq{Prompt: "a cat surfing", Model: "omni-flash"})
	defer common.CleanupBodyStorage(c)

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "flow/omni"},
	}
	body, err := (&TaskAdaptor{}).BuildRequestBody(c, info)
	require.NoError(t, err)
	encoded, err := io.ReadAll(body)
	require.NoError(t, err)

	_, params, err := mime.ParseMediaType(c.Request.Header.Get("Content-Type"))
	require.NoError(t, err)
	reader := multipart.NewReader(bytes.NewReader(encoded), params["boundary"])
	form, err := reader.ReadForm(1 << 20)
	require.NoError(t, err)
	defer form.RemoveAll()
	assert.Equal(t, []string{"flow/omni"}, form.Value["model"])
	assert.Equal(t, []string{"720x1280"}, form.Value["size"])
	assert.Equal(t, []string{"4"}, form.Value["seconds"])
}

func multipartVideoRequest(t *testing.T, fields map[string]string, referenceFiles int) *gin.Context {
	t.Helper()
	var input bytes.Buffer
	writer := multipart.NewWriter(&input)
	for key, value := range fields {
		require.NoError(t, writer.WriteField(key, value))
	}
	for index := 0; index < referenceFiles; index++ {
		part, err := writer.CreateFormFile("input_reference", "ref.png")
		require.NoError(t, err)
		_, err = part.Write([]byte("\x89PNG\r\n\x1a\n"))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(input.Bytes()))
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	return c
}

func TestSoraActionReflectsTheRequestedGenerationMode(t *testing.T) {
	for _, tc := range []struct {
		name           string
		fields         map[string]string
		referenceFiles int
		want           string
	}{
		{
			name:   "prompt only is text generation",
			fields: map[string]string{"model": "omni-flash", "prompt": "a cat surfing"},
			want:   constant.TaskActionTextGenerate,
		},
		{
			name:           "a single upload is image to video",
			fields:         map[string]string{"model": "omni-flash", "prompt": "make it move"},
			referenceFiles: 1,
			want:           constant.TaskActionGenerate,
		},
		{
			name:           "two uploads are a start and end pair",
			fields:         map[string]string{"model": "omni-flash", "prompt": "morph"},
			referenceFiles: 2,
			want:           constant.TaskActionFirstTailGenerate,
		},
		{
			name:           "three uploads are a reference set",
			fields:         map[string]string{"model": "omni-flash", "prompt": "combine"},
			referenceFiles: 3,
			want:           constant.TaskActionReferenceGenerate,
		},
		{
			name: "a metadata frame pair is a start and end pair",
			fields: map[string]string{
				"model":    "omni-flash",
				"prompt":   "morph",
				"metadata": `{"first_frame_url":"https://example.com/a.png","last_frame_url":"https://example.com/b.png"}`,
			},
			want: constant.TaskActionFirstTailGenerate,
		},
		{
			name: "metadata img_url is image to video",
			fields: map[string]string{
				"model":    "omni-flash",
				"prompt":   "make it move",
				"metadata": `{"img_url":"https://example.com/a.png"}`,
			},
			want: constant.TaskActionGenerate,
		},
		{
			name: "an explicit mode outranks the reference count",
			fields: map[string]string{
				"model":  "omni-flash",
				"prompt": "combine",
				"mode":   "r2v",
			},
			referenceFiles: 2,
			want:           constant.TaskActionReferenceGenerate,
		},
		{
			name: "an explicit operation outranks the reference count",
			fields: map[string]string{
				"model":     "omni-flash",
				"prompt":    "combine",
				"operation": "reference_to_video",
			},
			referenceFiles: 2,
			want:           constant.TaskActionReferenceGenerate,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := multipartVideoRequest(t, tc.fields, tc.referenceFiles)
			defer common.CleanupBodyStorage(c)

			info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
			taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info)

			require.Nil(t, taskErr)
			assert.Equal(t, tc.want, info.Action)
		})
	}
}

func TestSoraRejectsOperationAndModeTogether(t *testing.T) {
	// Upstream refuses the pair, so accepting it here would spend a channel
	// call to learn what the request already says.
	c := multipartVideoRequest(t, map[string]string{
		"model":     "omni-flash",
		"prompt":    "combine",
		"operation": "reference_to_video",
		"mode":      "r2v",
	}, 0)
	defer common.CleanupBodyStorage(c)

	info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info)

	require.NotNil(t, taskErr)
	assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
}

func TestSoraAcceptsTheUpstreamReferenceJSONDialect(t *testing.T) {
	// a2a spells a reference as {"image_url": ...} and a reference set as an
	// array of them. Failing to decode either shape rejected the request here
	// before the provider that understands it ever saw it.
	payload := []byte(`{"model":"omni-flash","prompt":"combine","input_reference":{"image_url":"https://example.com/a.png"}}`)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	defer common.CleanupBodyStorage(c)

	info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info)

	require.Nil(t, taskErr)
	assert.Equal(t, constant.TaskActionGenerate, info.Action)
	request, err := relaycommon.GetTaskRequest(c)
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/a.png", request.InputReference)
}

func TestSoraCountsAReferenceImageArrayAsAReferenceSet(t *testing.T) {
	payload := []byte(`{"model":"omni-flash","prompt":"combine","reference_images":[{"image_url":"https://example.com/a.png"},{"image_url":"https://example.com/b.png"},"https://example.com/c.png"]}`)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/videos", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	defer common.CleanupBodyStorage(c)

	info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
	taskErr := (&TaskAdaptor{}).ValidateRequestAndSetAction(c, info)

	require.Nil(t, taskErr)
	assert.Equal(t, constant.TaskActionReferenceGenerate, info.Action)
	request, err := relaycommon.GetTaskRequest(c)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"https://example.com/a.png",
		"https://example.com/b.png",
		"https://example.com/c.png",
	}, request.ReferenceImages)
}

func TestSoraForwardsEveryReferenceUploadUpstream(t *testing.T) {
	// The upstream operation is selected by how many references arrive, so a
	// dropped part would silently downgrade a reference set to image-to-video.
	c := multipartVideoRequest(t, map[string]string{
		"model":  "omni-flash",
		"prompt": "combine",
	}, 3)
	c.Set("task_request", relaycommon.TaskSubmitReq{Prompt: "combine", Model: "omni-flash"})
	defer common.CleanupBodyStorage(c)

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{UpstreamModelName: "flow/omni"},
	}
	body, err := (&TaskAdaptor{}).BuildRequestBody(c, info)
	require.NoError(t, err)
	encoded, err := io.ReadAll(body)
	require.NoError(t, err)

	_, params, err := mime.ParseMediaType(c.Request.Header.Get("Content-Type"))
	require.NoError(t, err)
	form, err := multipart.NewReader(bytes.NewReader(encoded), params["boundary"]).ReadForm(1 << 20)
	require.NoError(t, err)
	defer form.RemoveAll()
	assert.Len(t, form.File["input_reference"], 3)
}

func TestSoraDoResponseRestoresThePublicModelName(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	upstream := `{"id":"video_internal","task_id":"video_internal_legacy","object":"video","model":"flow/omni","status":"processing","remixed_from_video_id":"video_internal_origin","url":"https://storage.example/video.mp4?signature=secret","download_url":"https://storage.example/download"}`
	resp := &http.Response{Body: io.NopCloser(strings.NewReader(upstream))}
	info := &relaycommon.RelayInfo{
		OriginModelName: "omni-flash",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_public",
		},
	}

	taskID, _, taskErr := (&TaskAdaptor{}).DoResponse(c, resp, info)
	require.Nil(t, taskErr)
	assert.Equal(t, "video_internal", taskID)

	var response responseTask
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "task_public", response.ID)
	assert.Equal(t, "task_public", response.TaskID)
	assert.Equal(t, "omni-flash", response.Model)
	assert.NotContains(t, recorder.Body.String(), "storage.example")
	assert.NotContains(t, recorder.Body.String(), "download_url")
	assert.NotContains(t, recorder.Body.String(), "video_internal_origin")
}

func TestSoraDoResponseUsesPublicRemixOrigin(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	upstream := `{"id":"video_internal","object":"video","model":"flow/omni","status":"processing","remixed_from_video_id":"video_internal_origin"}`
	resp := &http.Response{Body: io.NopCloser(strings.NewReader(upstream))}
	info := &relaycommon.RelayInfo{
		OriginModelName: "omni-flash",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action:       constant.TaskActionRemix,
			OriginTaskID: "task_origin_public",
			PublicTaskID: "task_public",
		},
	}

	_, _, taskErr := (&TaskAdaptor{}).DoResponse(c, resp, info)
	require.Nil(t, taskErr)

	var response responseTask
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	assert.Equal(t, "task_origin_public", response.RemixedFromVideoID)
	assert.NotContains(t, recorder.Body.String(), "video_internal_origin")
}

func TestSoraPollingUsesPublicResponseAllowlist(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public",
		Properties: model.Properties{
			OriginModelName: "omni-flash",
		},
		Data: json.RawMessage(`{"id":"video_internal","task_id":"video_internal_legacy","object":"video","model":"flow/omni","status":"completed","remixed_from_video_id":"video_internal_origin","url":"https://storage.example/video.mp4?signature=secret","download_url":"https://storage.example/download","provider_metadata":{"bucket":"private"}}`),
	}

	data, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)

	var response map[string]any
	require.NoError(t, json.Unmarshal(data, &response))
	assert.Equal(t, "task_public", response["id"])
	assert.Equal(t, "task_public", response["task_id"])
	assert.Equal(t, "omni-flash", response["model"])
	assert.Equal(t, "completed", response["status"])
	assert.NotContains(t, response, "url")
	assert.NotContains(t, response, "download_url")
	assert.NotContains(t, response, "provider_metadata")
	assert.NotContains(t, response, "remixed_from_video_id")
	assert.NotContains(t, string(data), "storage.example")
	assert.NotContains(t, string(data), "video_internal_origin")
}

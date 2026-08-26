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

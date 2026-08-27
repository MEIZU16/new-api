package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"

	"github.com/stretchr/testify/assert"
)

// Every endpoint type a channel can advertise must resolve to a default path.
// The pricing response builds its endpoint map from these defaults, and the
// model documentation renders nothing at all for an endpoint type that is
// missing here, so a gap is silently invisible rather than a hard failure.
func TestDefaultEndpointInfoCoversAdvertisedEndpointTypes(t *testing.T) {
	cases := []struct {
		endpointType constant.EndpointType
		path         string
		method       string
	}{
		{constant.EndpointTypeOpenAI, "/v1/chat/completions", "POST"},
		{constant.EndpointTypeOpenAIResponse, "/v1/responses", "POST"},
		{constant.EndpointTypeOpenAIResponseCompact, "/v1/responses/compact", "POST"},
		{constant.EndpointTypeOpenAIAlphaSearch, "/v1/alpha/search", "POST"},
		{constant.EndpointTypeAnthropic, "/v1/messages", "POST"},
		{constant.EndpointTypeGemini, "/v1beta/models/{model}:generateContent", "POST"},
		{constant.EndpointTypeJinaRerank, "/v1/rerank", "POST"},
		{constant.EndpointTypeImageGeneration, "/v1/images/generations", "POST"},
		{constant.EndpointTypeEmbeddings, "/v1/embeddings", "POST"},
		{constant.EndpointTypeOpenAIVideo, "/v1/videos", "POST"},
	}

	for _, tc := range cases {
		t.Run(string(tc.endpointType), func(t *testing.T) {
			info, ok := GetDefaultEndpointInfo(tc.endpointType)

			assert.True(t, ok)
			assert.Equal(t, tc.path, info.Path)
			assert.Equal(t, tc.method, info.Method)
		})
	}
}

// A Sora channel publishes only the video endpoint type, so that type carries
// the whole public API surface of every video SKU routed through it.
func TestSoraChannelVideoEndpointTypeResolves(t *testing.T) {
	endpointTypes := GetEndpointTypesByChannelType(constant.ChannelTypeSora, "omni-flash")

	assert.Equal(t, []constant.EndpointType{constant.EndpointTypeOpenAIVideo}, endpointTypes)
	for _, endpointType := range endpointTypes {
		_, ok := GetDefaultEndpointInfo(endpointType)
		assert.True(t, ok, "endpoint type %q has no default path", endpointType)
	}
}

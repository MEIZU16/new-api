package relay

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/stretchr/testify/require"
)

func TestApplyImageParamOverrideLocksMultipartEditQuality(t *testing.T) {
	request := &dto.ImageRequest{
		Model:   "aistudio/gemini-3-pro-image",
		Prompt:  "edit the reference image",
		Quality: "4k",
	}
	info := &relaycommon.RelayInfo{
		OriginModelName: "gemini-3-pro-image-2k",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "aistudio/gemini-3-pro-image",
			ParamOverride: map[string]interface{}{
				"operations": []interface{}{
					map[string]interface{}{
						"path":  "quality",
						"mode":  "set",
						"value": "2k",
						"conditions": []interface{}{
							map[string]interface{}{
								"path":  "original_model",
								"mode":  "full",
								"value": "gemini-3-pro-image-2k",
							},
						},
					},
				},
			},
		},
	}

	require.Nil(t, applyImageParamOverride(request, info))
	require.Equal(t, "2k", request.Quality)
	require.Equal(t, "aistudio/gemini-3-pro-image", request.Model)
}

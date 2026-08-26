package advancedcustom

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

const (
	maxGeminiReferenceImageBytes     = int64(16 << 20)
	maxGeminiReferenceAggregateBytes = int64(32 << 20)
)

func populateGeminiImageReferences(c *gin.Context, request *dto.ImageRequest) error {
	if c == nil || c.Request == nil {
		return invalidGeminiImageEditRequest("missing image edit request")
	}

	form := c.Request.MultipartForm
	if form == nil {
		parsed, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return invalidGeminiImageEditRequest(
				fmt.Sprintf("failed to parse image edit form request: %v", err),
			)
		}
		form = parsed
		c.Request.MultipartForm = parsed
		c.Request.PostForm = parsed.Value
	}
	if form == nil {
		return invalidGeminiImageEditRequest("image is required")
	}
	if len(form.File["mask"]) > 0 {
		return invalidGeminiImageEditRequest("mask is not supported for this model")
	}

	imageFiles := make([]*multipart.FileHeader, 0)
	imageFiles = append(imageFiles, form.File["image"]...)
	imageFiles = append(imageFiles, form.File["image[]"]...)

	indexedFields := make([]string, 0)
	for field := range form.File {
		if strings.HasPrefix(field, "image[") && field != "image[]" {
			indexedFields = append(indexedFields, field)
		}
	}
	sort.Strings(indexedFields)
	for _, field := range indexedFields {
		imageFiles = append(imageFiles, form.File[field]...)
	}
	if len(imageFiles) == 0 {
		return invalidGeminiImageEditRequest("image is required")
	}
	declaredBytes := int64(0)
	for index, fileHeader := range imageFiles {
		if fileHeader == nil || fileHeader.Size < 0 {
			return invalidGeminiImageEditRequest(
				fmt.Sprintf("reference image %d is invalid", index+1),
			)
		}
		if fileHeader.Size > maxGeminiReferenceImageBytes {
			return geminiImageEditPayloadTooLarge(
				fmt.Sprintf("reference image %d exceeds the 16 MiB limit", index+1),
			)
		}
		if fileHeader.Size > maxGeminiReferenceAggregateBytes-declaredBytes {
			return geminiImageEditPayloadTooLarge("reference images exceed the 32 MiB aggregate limit")
		}
		declaredBytes += fileHeader.Size
	}

	request.ReferenceImages = make([]dto.ImageReference, 0, len(imageFiles))
	actualBytes := int64(0)
	for index, fileHeader := range imageFiles {
		file, err := fileHeader.Open()
		if err != nil {
			return invalidGeminiImageEditRequest(
				fmt.Sprintf("failed to open reference image %d: %v", index+1, err),
			)
		}
		data, readErr := io.ReadAll(io.LimitReader(file, maxGeminiReferenceImageBytes+1))
		closeErr := file.Close()
		if readErr != nil {
			return invalidGeminiImageEditRequest(
				fmt.Sprintf("failed to read reference image %d: %v", index+1, readErr),
			)
		}
		if closeErr != nil {
			return invalidGeminiImageEditRequest(
				fmt.Sprintf("failed to close reference image %d: %v", index+1, closeErr),
			)
		}
		if int64(len(data)) > maxGeminiReferenceImageBytes {
			return geminiImageEditPayloadTooLarge(
				fmt.Sprintf("reference image %d exceeds the 16 MiB limit", index+1),
			)
		}
		if len(data) == 0 {
			return invalidGeminiImageEditRequest(
				fmt.Sprintf("reference image %d is empty", index+1),
			)
		}
		if int64(len(data)) > maxGeminiReferenceAggregateBytes-actualBytes {
			return geminiImageEditPayloadTooLarge("reference images exceed the 32 MiB aggregate limit")
		}
		actualBytes += int64(len(data))

		mimeType := normalizedImageMimeType(data)
		if mimeType == "" {
			return invalidGeminiImageEditRequest(
				fmt.Sprintf("reference image %d must be a valid PNG, JPEG, or WebP image", index+1),
			)
		}
		request.ReferenceImages = append(request.ReferenceImages, dto.ImageReference{
			MimeType: mimeType,
			Data:     base64.StdEncoding.EncodeToString(data),
		})
	}
	return nil
}

func normalizedImageMimeType(data []byte) string {
	detected := strings.ToLower(strings.TrimSpace(http.DetectContentType(data)))
	switch detected {
	case "image/png", "image/jpeg", "image/webp":
		return detected
	default:
		return ""
	}
}

func invalidGeminiImageEditRequest(message string) error {
	return types.NewErrorWithStatusCode(
		errors.New(message),
		types.ErrorCodeInvalidRequest,
		http.StatusBadRequest,
		types.ErrOptionWithSkipRetry(),
	)
}

func geminiImageEditPayloadTooLarge(message string) error {
	return types.NewErrorWithStatusCode(
		errors.New(message),
		types.ErrorCodeInvalidRequest,
		http.StatusRequestEntityTooLarge,
		types.ErrOptionWithSkipRetry(),
	)
}

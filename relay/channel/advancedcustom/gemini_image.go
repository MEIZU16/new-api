package advancedcustom

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
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

	request.ReferenceImages = make([]dto.ImageReference, 0, len(imageFiles))
	for index, fileHeader := range imageFiles {
		file, err := fileHeader.Open()
		if err != nil {
			return invalidGeminiImageEditRequest(
				fmt.Sprintf("failed to open reference image %d: %v", index+1, err),
			)
		}
		data, readErr := io.ReadAll(file)
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
		if len(data) == 0 {
			return invalidGeminiImageEditRequest(
				fmt.Sprintf("reference image %d is empty", index+1),
			)
		}

		mimeType := normalizedImageMimeType(fileHeader, data)
		if mimeType == "" {
			return invalidGeminiImageEditRequest(
				fmt.Sprintf("reference image %d must use an image MIME type", index+1),
			)
		}
		request.ReferenceImages = append(request.ReferenceImages, dto.ImageReference{
			MimeType: mimeType,
			Data:     base64.StdEncoding.EncodeToString(data),
		})
	}
	return nil
}

func normalizedImageMimeType(fileHeader *multipart.FileHeader, data []byte) string {
	if fileHeader != nil {
		if mediaType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type")); err == nil {
			mediaType = strings.ToLower(strings.TrimSpace(mediaType))
			if strings.HasPrefix(mediaType, "image/") {
				return mediaType
			}
		}
	}

	detected := strings.ToLower(strings.TrimSpace(http.DetectContentType(data)))
	if strings.HasPrefix(detected, "image/") {
		return detected
	}
	if fileHeader == nil {
		return ""
	}
	extensionType := mime.TypeByExtension(strings.ToLower(filepath.Ext(fileHeader.Filename)))
	if mediaType, _, err := mime.ParseMediaType(extensionType); err == nil {
		mediaType = strings.ToLower(strings.TrimSpace(mediaType))
		if strings.HasPrefix(mediaType, "image/") {
			return mediaType
		}
	}
	return ""
}

func invalidGeminiImageEditRequest(message string) error {
	return types.NewErrorWithStatusCode(
		errors.New(message),
		types.ErrorCodeInvalidRequest,
		http.StatusBadRequest,
		types.ErrOptionWithSkipRetry(),
	)
}

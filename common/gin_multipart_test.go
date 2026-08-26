package common

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type multipartStorageWithoutBytes struct {
	BodyStorage
}

func (multipartStorageWithoutBytes) Bytes() ([]byte, error) {
	return nil, errors.New("multipart parsing must not materialize body storage")
}

func TestParseMultipartFormReusableStreamsFromBodyStorage(t *testing.T) {
	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	require.NoError(t, writer.WriteField("prompt", "edit this"))
	file, err := writer.CreateFormFile("image", "reference.png")
	require.NoError(t, err)
	_, err = file.Write([]byte("reference"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	storage, err := CreateBodyStorage(payload.Bytes())
	require.NoError(t, err)
	defer storage.Close()

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", nil)
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	c.Set(KeyBodyStorage, multipartStorageWithoutBytes{BodyStorage: storage})

	form, err := ParseMultipartFormReusable(c)
	require.NoError(t, err)
	defer form.RemoveAll()
	require.Equal(t, []string{"edit this"}, form.Value["prompt"])
	require.Len(t, form.File["image"], 1)
	require.Equal(t, "reference.png", form.File["image"][0].Filename)
}

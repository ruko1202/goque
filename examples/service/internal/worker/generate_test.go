package worker

import (
	"bytes"
	"encoding/json"
	"example/internal/models"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	url = "http://localhost:8080/api/tasks"
)

type requestBody struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func TestGenerate(t *testing.T) {
	count := 1000
	for i := range count {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			payload, err := generatePayload(models.TaskTypeEmail)
			require.NoError(t, err)

			buf := new(bytes.Buffer)
			err = json.NewEncoder(buf).Encode(&requestBody{
				Type:    models.TaskTypeEmail,
				Payload: payload,
			})
			require.NoError(t, err)

			resp, err := http.Post(url, "application/json", buf)
			require.NoError(t, err)
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			t.Log(string(body))
			require.Equal(t, http.StatusCreated, resp.StatusCode)
		})
	}
}

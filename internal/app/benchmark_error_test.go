package app

import (
	"bytes"
	"mime/multipart"
	"strings"
	"testing"
)

func TestReadBenchmarkFiles_ErrorMessages(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		content     string
		wantErrMsg  string
	}{
		{
			name:     "unsupported file format",
			filename: "invalid.csv",
			content:  "This is not a valid benchmark file\nsome data\n123,456",
			wantErrMsg: "file 'invalid.csv': unsupported file format",
		},
		{
			name:     "empty file",
			filename: "empty.csv",
			content:  "os,cpu,gpu,ram,kernel,driver,cpuscheduler\nLinux,Intel,NVIDIA,16GB,5.15,nvidia,performance\nfps,frametime\n",
			wantErrMsg: "file 'empty.csv': no valid benchmark data found in file",
		},
		{
			name:     "file with invalid specs",
			filename: "badspecs.csv",
			content:  ", Hardware monitoring log v1.6 ,,\n0\nPower,Framerate\n100,60",
			wantErrMsg: "file 'badspecs.csv': invalid specs line format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a multipart file header
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("files", tt.filename)
			if err != nil {
				t.Fatalf("Failed to create form file: %v", err)
			}
			_, err = part.Write([]byte(tt.content))
			if err != nil {
				t.Fatalf("Failed to write content: %v", err)
			}
			if err = writer.Close(); err != nil {
				t.Fatalf("Failed to close writer: %v", err)
			}

			// Parse the multipart form
			reader := multipart.NewReader(body, writer.Boundary())
			form, err := reader.ReadForm(1024 * 1024)
			if err != nil {
				t.Fatalf("Failed to read form: %v", err)
			}
			defer func() {
				if removeErr := form.RemoveAll(); removeErr != nil {
					t.Logf("Warning: failed to remove form files: %v", removeErr)
				}
			}()

			// Call ReadBenchmarkFiles
			_, err = ReadBenchmarkFiles(form.File["files"])
			
			if err == nil {
				t.Fatalf("Expected error but got nil")
			}

			if !strings.Contains(err.Error(), tt.wantErrMsg) {
				t.Errorf("Error message doesn't contain expected text.\nGot: %q\nWant substring: %q", err.Error(), tt.wantErrMsg)
			}

			// Verify the filename is in the error message
			if !strings.Contains(err.Error(), tt.filename) {
				t.Errorf("Error message doesn't contain filename %q. Got: %q", tt.filename, err.Error())
			}
		})
	}
}

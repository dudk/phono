package controller

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"strconv"

	"github.com/pipelined/phono/input"
	"github.com/pipelined/pipe"
)

// Convert converts form files to the format provided y form.
// To limit maximum input file size, pass map of extensions with limits.
// Process request steps:
//	1. Retrieve input format from URL
//	2. Use http.MaxBytesReader to avoid memory abuse
//	3. Parse output configuration
//	4. Create temp file
//	5. Run conversion
//	6. Send result file
func Convert(form input.ConvertForm, bufferSize int, tempDir string) http.Handler {
	formData := form.Data()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, err := w.Write(formData)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		case http.MethodPost:
			// get max size for the format
			if maxSize, err := form.InputMaxSize(r.URL.Path); err == nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxSize)
				// check max size
				if err := r.ParseMultipartForm(maxSize); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
			} else {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			f, handler, err := r.FormFile(form.FileKey())
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			defer f.Close()

			// parse pump
			pump, err := input.FilePump(handler.Filename, f)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// parse sink and validate parameters
			buildFn, ext, err := form.ParseSink(r.Form)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// create temp file
			tempFile, err := ioutil.TempFile(tempDir, "")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer cleanUp(tempFile)

			// convert file using temp file
			if err = convert(bufferSize, pump, buildFn(tempFile)); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			// reset temp file
			_, err = tempFile.Seek(0, 0)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to reset temp file: %v", err), http.StatusInternalServerError)
				return
			}
			// get temp file stats for headers
			stat, err := tempFile.Stat()
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to get file stats: %v", err), http.StatusInternalServerError)
				return
			}
			fileSize := strconv.FormatInt(stat.Size(), 10)
			//Send the headers
			w.Header().Set("Content-Disposition", "attachment; filename="+outFileName("result", 1, ext))
			w.Header().Set("Content-Type", mime.TypeByExtension(ext))
			w.Header().Set("Content-Length", fileSize)
			_, err = io.Copy(w, tempFile) // send file to a client
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to transfer file: %v", err), http.StatusInternalServerError)
			}
			return
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

// convert using pump as the source and SinkBuilder as destination.
func convert(bufferSize int, pump pipe.Pump, sink pipe.Sink) error {
	// build convert pipe
	convert, err := pipe.New(bufferSize,
		pipe.WithPump(pump),
		pipe.WithSinks(sink),
	)
	if err != nil {
		return fmt.Errorf("Failed to build pipe: %v", err)
	}

	// run conversion
	err = pipe.Wait(convert.Run())
	if err != nil {
		return fmt.Errorf("Failed to execute pipe: %v", err)
	}
	return nil
}

// outFileName return output file name. It replaces input format extension with output.
func outFileName(prefix string, idx int, ext string) string {
	return fmt.Sprintf("%v_%d%v", prefix, idx, ext)
}

// cleanUp removes temporary file and handles all errors on the way.
func cleanUp(f *os.File) {
	err := f.Close()
	if err != nil {
		log.Printf("Failed to close temp file")
	}
	err = os.Remove(f.Name())
	if err != nil {
		log.Printf("Failed to delete temp file")
	}
}

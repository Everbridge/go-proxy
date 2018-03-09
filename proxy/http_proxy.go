package proxy

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/visola/go-proxy/config"
	myhttp "github.com/visola/go-proxy/http"
)

var urlToProxy = fmt.Sprintf("https://localhost:%d", port)

func proxyRequest(req *http.Request, w http.ResponseWriter, mapping config.Mapping) (*proxyResponse, error) {
	oldPath := req.URL.Path
	newPath := mapping.To + "/" + oldPath[len(mapping.From):]

	if strings.HasSuffix(newPath, "/") {
		newPath = newPath[:len(newPath)-1]
	}

	newURL, parseErr := url.Parse(fmt.Sprintf("%s?%s", newPath, req.URL.RawQuery))
	if parseErr != nil {
		return &proxyResponse{
			executedURL:  newPath,
			responseCode: http.StatusInternalServerError,
		}, parseErr
	}

	defer req.Body.Close()
	bodyInBytes, readBodyErr := ioutil.ReadAll(req.Body)

	if readBodyErr != nil {
		return &proxyResponse{
			executedURL:  newPath,
			responseCode: http.StatusInternalServerError,
		}, readBodyErr
	}

	newReq, newReqErr := http.NewRequest(req.Method, newURL.String(), bytes.NewBuffer(bodyInBytes))
	if newReqErr != nil {
		return &proxyResponse{
			executedURL:  newPath,
			responseCode: http.StatusInternalServerError,
		}, newReqErr
	}

	// Copy request headers
	for name, values := range req.Header {
		for _, value := range values {
			newReq.Header.Add(name, value)
		}
	}

	client := &http.Client{
		// Do not auto-follow redirects
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, respErr := client.Do(newReq)

	if respErr != nil {
		return &proxyResponse{
			executedURL:  newPath,
			responseCode: http.StatusInternalServerError,
		}, respErr
	}

	defer resp.Body.Close()

	// Copy response headers
	for name, values := range resp.Header {
		for _, value := range values {

			// Fix location headers to point to proxy
			if strings.ToLower(name) == "location" {
				if strings.HasPrefix(value, mapping.To) {
					value = urlToProxy + value[len(mapping.To):]
				}
			}
			w.Header().Add(name, value)
		}
	}

	// Copy status
	w.WriteHeader(resp.StatusCode)

	responseBytes := make([]byte, 0)
	buffer := make([]byte, 512)
	for {
		bytesRead, readError := resp.Body.Read(buffer)

		if readError != nil && readError != io.EOF {
			return &proxyResponse{
				executedURL:  newPath,
				responseCode: http.StatusInternalServerError,
			}, readError
		}

		if bytesRead == 0 {
			break
		}

		responseBytes = append(responseBytes, buffer[:bytesRead]...)
		w.Write(buffer[:bytesRead])
	}

	bodyString := "Binary"
	if myhttp.IsText(resp.Header["Content-Type"]...) {
		if myhttp.IsGzipped(resp.Header["Content-Encoding"]...) {
			gzippedReader, _ := gzip.NewReader(bytes.NewReader(responseBytes))
			ungzippedBytes, _ := ioutil.ReadAll(gzippedReader)
			bodyString = string(ungzippedBytes)
		} else {
			bodyString = string(responseBytes)
		}
	}

	return &proxyResponse{
		body:         bodyString,
		executedURL:  newPath,
		headers:      resp.Header,
		responseCode: resp.StatusCode,
	}, nil
}

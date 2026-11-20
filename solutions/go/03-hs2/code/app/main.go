package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

var hop = map[string]bool{"Connection": true, "Keep-Alive": true, "Proxy-Authenticate": true,
	"Proxy-Authorization": true, "Te": true, "Trailer": true, "Transfer-Encoding": true,
	"Upgrade": true, "Content-Length": true}

func copyHeaders(destination, source http.Header) {
	excluded := map[string]bool{}
	for _, name := range strings.Split(strings.Join(source.Values("Connection"), ","), ",") {
		excluded[http.CanonicalHeaderKey(strings.TrimSpace(name))] = true
	}
	for key, values := range source {
		if hop[key] || excluded[key] || strings.HasPrefix(strings.ToLower(key), "x-waf-") {
			continue
		}
		destination[key] = append([]string(nil), values...)
	}
}

func reject(w http.ResponseWriter, status int, rule string, diagnostics http.Header) {
	for key, values := range diagnostics {
		w.Header()[key] = values
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-WAF-Match", rule)
	w.Header().Set("Connection", "close")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"rule": rule}})
}

func main() {
	origin := flag.String("origin", "", "Origin web application URL")
	detector := flag.String("detector", "", "ML detector URL")
	policyPath := flag.String("policy", "", "Trusted policy JSON file")
	flag.Parse()
	_ = detector
	_ = policyPath
	client := &http.Client{Timeout: 3 * time.Second,
		Transport:     &http.Transport{DisableCompression: true},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && r.URL.RequestURI() == "/health" {
			w.WriteHeader(200)
			return
		}
		diagnostics := http.Header{}
		var reader io.Reader = r.Body
		body, err := io.ReadAll(reader)
		if err != nil {
			reject(w, 400, "invalid_request", diagnostics)
			return
		}
		request, err := http.NewRequestWithContext(r.Context(), r.Method, strings.TrimRight(*origin, "/")+r.URL.RequestURI(), bytes.NewReader(body))
		if err != nil {
			reject(w, 502, "origin_unavailable", diagnostics)
			return
		}
		copyHeaders(request.Header, r.Header)
		response, err := client.Do(request)
		if err != nil {
			reject(w, 502, "origin_unavailable", diagnostics)
			return
		}
		defer response.Body.Close()
		var responseReader io.Reader = response.Body
		content, err := io.ReadAll(responseReader)
		if err != nil {
			reject(w, 502, "origin_unavailable", diagnostics)
			return
		}
		copyHeaders(w.Header(), response.Header)
		for key, values := range diagnostics {
			w.Header()[key] = values
		}
		w.WriteHeader(response.StatusCode)
		w.Write(content)
	})
	server := &http.Server{Addr: "127.0.0.1:8080", Handler: handler}
	if err := server.ListenAndServe(); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}

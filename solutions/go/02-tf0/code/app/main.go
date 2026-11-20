package main

import (
	"flag"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

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
		request, err := http.NewRequestWithContext(r.Context(), r.Method, strings.TrimRight(*origin, "/")+r.URL.RequestURI(), r.Body)
		if err != nil {
			http.Error(w, "origin unavailable", 502)
			return
		}
		request.Header = r.Header.Clone()
		request.ContentLength = r.ContentLength
		response, err := client.Do(request)
		if err != nil {
			http.Error(w, "origin unavailable", 502)
			return
		}
		defer response.Body.Close()
		for key, values := range response.Header {
			w.Header()[key] = values
		}
		w.WriteHeader(response.StatusCode)
		io.Copy(w, response.Body)
	})
	server := &http.Server{Addr: "127.0.0.1:8080", Handler: handler}
	if err := server.ListenAndServe(); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}

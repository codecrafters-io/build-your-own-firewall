package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

func main() {
	origin := flag.String("origin", "", "Origin chat completions API URL")
	flag.String("detector", "", "Classifier service URL")
	policyPath := flag.String("policy", "", "Policy JSON file path")
	flag.Parse()

	var policy struct {
		Mode     string `json:"mode"`
		Denylist []struct {
			ID   string `json:"id"`
			Term string `json:"term"`
		} `json:"denylist"`
	}
	data, err := os.ReadFile(*policyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.Unmarshal(data, &policy); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	target, err := url.Parse(*origin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(target)
		},
	}
	http.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Could not read request", http.StatusBadRequest)
			return
		}
		var request struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(body, &request); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		matchedRule := ""
		for _, rule := range policy.Denylist {
			for _, message := range request.Messages {
				if strings.Contains(message.Content, rule.Term) {
					matchedRule = rule.ID
					break
				}
			}
			if matchedRule != "" {
				break
			}
		}
		if matchedRule != "" {
			if policy.Mode != "log" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{
					"message": "Blocked by firewall policy", "type": "firewall_block", "rule": matchedRule,
				}})
				return
			}
			w.Header().Set("X-Firewall-Match", matchedRule)
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		proxy.ServeHTTP(w, r)
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

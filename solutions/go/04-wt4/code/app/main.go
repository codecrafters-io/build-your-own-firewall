package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

type normalizationPolicy struct {
	ZeroWidth  []string          `json:"zero_width"`
	Homoglyphs map[string]string `json:"homoglyphs"`
}

func inspectionTexts(text string, settings *normalizationPolicy) []string {
	if settings == nil {
		return []string{text}
	}
	normalize := func(value string) string {
		for _, character := range settings.ZeroWidth {
			value = strings.ReplaceAll(value, character, "")
		}
		value = cases.Fold().String(norm.NFKC.String(value))
		var result strings.Builder
		for _, character := range value {
			if replacement, ok := settings.Homoglyphs[string(character)]; ok {
				result.WriteString(replacement)
			} else {
				result.WriteRune(character)
			}
		}
		return result.String()
	}
	candidates := []string{normalize(text)}
	decoded, err := base64.StdEncoding.DecodeString(text)
	if err == nil && utf8.Valid(decoded) {
		candidates = append(candidates, normalize(string(decoded)))
	}
	return candidates
}

// messageTexts returns every string the model reads from the request's messages.
// It reports false when the request can't be inspected, so the caller fails closed.
func messageTexts(body []byte) ([]string, bool) {
	var request struct {
		Messages []struct {
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if json.Unmarshal(body, &request) != nil || request.Messages == nil {
		return nil, false
	}
	var texts []string
	for _, message := range request.Messages {
		if len(message.Content) == 0 || string(message.Content) == "null" {
			continue
		}
		var text string
		if json.Unmarshal(message.Content, &text) == nil {
			texts = append(texts, text)
			continue
		}
		var parts []struct {
			Type string  `json:"type"`
			Text *string `json:"text"`
		}
		if json.Unmarshal(message.Content, &parts) != nil {
			return nil, false
		}
		for _, part := range parts {
			if part.Type != "text" || part.Text == nil {
				return nil, false
			}
			texts = append(texts, *part.Text)
		}
	}
	return texts, true
}

func main() {
	origin := flag.String("origin", "", "Origin chat completions API URL")
	flag.String("detector", "", "Classifier service URL")
	policyPath := flag.String("policy", "", "Policy JSON file path")
	flag.Parse()

	var policy struct {
		Normalization *normalizationPolicy `json:"normalization"`
		Mode          string               `json:"mode"`
		Denylist      []struct {
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
		texts, ok := messageTexts(body)
		if !ok {
			http.Error(w, "Could not inspect request", http.StatusBadRequest)
			return
		}
		var inspection []string
		for _, text := range texts {
			inspection = append(inspection, inspectionTexts(text, policy.Normalization)...)
		}
		matchedRule := ""
		for _, rule := range policy.Denylist {
			for _, text := range inspection {
				if strings.Contains(text, rule.Term) {
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

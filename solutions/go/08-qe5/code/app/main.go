package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
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

type inspectedMessage struct {
	Role  string
	Texts []string
}

// Keep message boundaries so a classifier result is paired with its own clues.
func inspectMessages(body []byte) ([]inspectedMessage, bool) {
	var request struct {
		Messages []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if json.Unmarshal(body, &request) != nil || request.Messages == nil {
		return nil, false
	}
	var messages []inspectedMessage
	for _, message := range request.Messages {
		current := inspectedMessage{Role: message.Role}
		if len(message.Content) == 0 || string(message.Content) == "null" {
			messages = append(messages, current)
			continue
		}
		var text string
		if json.Unmarshal(message.Content, &text) == nil {
			current.Texts = []string{text}
			messages = append(messages, current)
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
			current.Texts = append(current.Texts, *part.Text)
		}
		messages = append(messages, current)
	}
	return messages, true
}

type heuristicSignal struct {
	ID     string `json:"id"`
	Weight int    `json:"weight"`
	Kind   string `json:"kind"`
	Term   string `json:"term"`
	Other  string `json:"other"`
	MaxGap int    `json:"max_gap"`
}

type heuristicPolicy struct {
	Threshold int               `json:"threshold"`
	Signals   []heuristicSignal `json:"signals"`
}

// nearTerms checks all non-overlapping occurrences in either order.
func nearTerms(text, first, second string, maxGap int) bool {
	positions := func(term string) []int {
		var found []int
		for start := 0; start <= len(text)-len(term); start++ {
			if strings.HasPrefix(text[start:], term) {
				found = append(found, start)
			}
		}
		return found
	}
	for _, left := range positions(first) {
		for _, right := range positions(second) {
			if end := left + len(first); end <= right && utf8.RuneCountInString(text[end:right]) <= maxGap {
				return true
			}
			if end := right + len(second); end <= left && utf8.RuneCountInString(text[end:left]) <= maxGap {
				return true
			}
		}
	}
	return false
}

func scoreHeuristics(candidates []string, settings *heuristicPolicy) (int, []string) {
	score := 0
	var signals []string
	for _, signal := range settings.Signals {
		for _, text := range candidates {
			matched := signal.Kind == "contains" && strings.Contains(text, signal.Term)
			if signal.Kind == "near" {
				matched = nearTerms(text, signal.Term, signal.Other, signal.MaxGap)
			}
			if matched {
				score += signal.Weight
				signals = append(signals, signal.ID)
				break
			}
		}
	}
	return score, signals
}

type classifierPolicy struct {
	Threshold   int    `json:"threshold"`
	FailureMode string `json:"failure_mode"`
}

// Keep each result associated with its message for later decision rules.
func classifyMessages(ctx context.Context, endpoint string, body []byte) ([]int, bool) {
	var request struct {
		Messages []json.RawMessage `json:"messages"`
	}
	if json.Unmarshal(body, &request) != nil {
		return nil, false
	}
	scores := make([]int, len(request.Messages))
	failures := make([]bool, len(request.Messages))
	client := &http.Client{
		Transport:     &http.Transport{},
		Timeout:       200 * time.Millisecond,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	defer client.CloseIdleConnections()
	var workers sync.WaitGroup
	for index, message := range request.Messages {
		workers.Add(1)
		go func(index int, message json.RawMessage) {
			defer workers.Done()
			failures[index] = true
			outgoing, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(endpoint, "/")+"/classify", bytes.NewReader(message))
			if err != nil {
				return
			}
			outgoing.Header.Set("Content-Type", "application/json")
			response, err := client.Do(outgoing)
			if err != nil {
				return
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				return
			}
			data, err := io.ReadAll(response.Body)
			if err != nil {
				return
			}
			var result struct {
				Score *int `json:"score"`
			}
			if json.Unmarshal(data, &result) != nil || result.Score == nil || *result.Score < 0 || *result.Score > 100 {
				return
			}
			scores[index] = *result.Score
			failures[index] = false
		}(index, message)
	}
	workers.Wait()
	for _, failed := range failures {
		if failed {
			return nil, false
		}
	}
	return scores, true
}

type scoreThresholds struct {
	ClassifierHigh   int `json:"classifier_high"`
	ClassifierMedium int `json:"classifier_medium"`
	HeuristicMedium  int `json:"heuristic_medium"`
}

type decisionPolicy struct {
	Default scoreThresholds `json:"default"`
	Tool    scoreThresholds `json:"tool"`
}

func decisionRule(messages []inspectedMessage, scores []int, settings *decisionPolicy, heuristics *heuristicPolicy, normalization *normalizationPolicy) string {
	for index, message := range messages {
		thresholds := settings.Default
		if message.Role == "tool" {
			thresholds = settings.Tool
		}
		if scores[index] >= thresholds.ClassifierHigh {
			return "classifier_high"
		}
		heuristicScore := 0
		if heuristics != nil {
			var candidates []string
			for _, text := range message.Texts {
				candidates = append(candidates, inspectionTexts(text, normalization)...)
			}
			heuristicScore, _ = scoreHeuristics(candidates, heuristics)
		}
		if scores[index] >= thresholds.ClassifierMedium && heuristicScore >= thresholds.HeuristicMedium {
			return "combined_scores"
		}
	}
	return ""
}

type dlpPolicy struct {
	Patterns []string `json:"patterns"`
	Canary   string   `json:"canary"`
}

// Only message content is rewritten. Raw JSON preserves unrelated fields and numbers.
func redactRequest(body []byte, patterns []*regexp.Regexp) []byte {
	var request map[string]json.RawMessage
	json.Unmarshal(body, &request)
	var messages []map[string]json.RawMessage
	json.Unmarshal(request["messages"], &messages)
	placeholders := map[string]string{}
	changed := false
	redact := func(text string) string {
		var result strings.Builder
		for len(text) > 0 {
			start, end := -1, -1
			for _, pattern := range patterns {
				match := pattern.FindStringIndex(text)
				if match != nil && (start < 0 || match[0] < start) {
					start, end = match[0], match[1]
				}
			}
			if start < 0 {
				result.WriteString(text)
				break
			}
			value := text[start:end]
			placeholder, exists := placeholders[value]
			if !exists {
				placeholder = fmt.Sprintf("[REDACTED_%d]", len(placeholders)+1)
				placeholders[value] = placeholder
			}
			result.WriteString(text[:start])
			result.WriteString(placeholder)
			text = text[end:]
			changed = true
		}
		return result.String()
	}
	for _, message := range messages {
		var text string
		if json.Unmarshal(message["content"], &text) == nil && string(message["content"]) != "null" {
			message["content"], _ = json.Marshal(redact(text))
		} else {
			var parts []map[string]json.RawMessage
			if json.Unmarshal(message["content"], &parts) == nil && parts != nil {
				for _, part := range parts {
					json.Unmarshal(part["text"], &text)
					part["text"], _ = json.Marshal(redact(text))
				}
				message["content"], _ = json.Marshal(parts)
			}
		}
	}
	if !changed {
		return body
	}
	request["messages"], _ = json.Marshal(messages)
	output, _ := json.Marshal(request)
	return output
}

func containsCanary(body []byte, canary string) bool {
	if strings.Contains(string(body), canary) {
		return true
	}
	var value any
	if json.Unmarshal(body, &value) != nil {
		return false
	}
	var contains func(any) bool
	contains = func(value any) bool {
		switch item := value.(type) {
		case string:
			return strings.Contains(item, canary)
		case []any:
			for _, child := range item {
				if contains(child) {
					return true
				}
			}
		case map[string]any:
			for _, child := range item {
				if contains(child) {
					return true
				}
			}
		}
		return false
	}
	return contains(value)
}

func main() {
	origin := flag.String("origin", "", "Origin chat completions API URL")
	detector := flag.String("detector", "", "Classifier service URL")
	policyPath := flag.String("policy", "", "Policy JSON file path")
	flag.Parse()

	var policy struct {
		DLP           *dlpPolicy           `json:"dlp"`
		Decisions     *decisionPolicy      `json:"decisions"`
		Classifier    *classifierPolicy    `json:"classifier"`
		Heuristics    *heuristicPolicy     `json:"heuristics"`
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

	var redactionPatterns []*regexp.Regexp
	if policy.DLP != nil {
		for _, pattern := range policy.DLP.Patterns {
			redactionPatterns = append(redactionPatterns, regexp.MustCompile(pattern))
		}
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
	if policy.DLP != nil {
		proxy.ModifyResponse = func(response *http.Response) error {
			body, err := io.ReadAll(response.Body)
			response.Body.Close()
			if err != nil {
				return err
			}
			if containsCanary(body, policy.DLP.Canary) {
				body, _ = json.Marshal(map[string]any{"error": map[string]string{
					"message": "Blocked outgoing canary", "type": "firewall_block", "rule": "canary_leak",
				}})
				response.StatusCode = http.StatusForbidden
				response.Header = http.Header{"Content-Type": []string{"application/json"}}
			}
			response.Body = io.NopCloser(bytes.NewReader(body))
			response.ContentLength = int64(len(body))
			response.Header.Set("Content-Length", strconv.Itoa(len(body)))
			return nil
		}
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
		messages, ok := inspectMessages(body)
		if !ok {
			http.Error(w, "Could not inspect request", http.StatusBadRequest)
			return
		}
		var inspection []string
		for _, message := range messages {
			for _, text := range message.Texts {
				inspection = append(inspection, inspectionTexts(text, policy.Normalization)...)
			}
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
		heuristicScore := 0
		if policy.Heuristics != nil {
			score, signals := scoreHeuristics(inspection, policy.Heuristics)
			heuristicScore = score
			w.Header().Set("X-Firewall-Score", strconv.Itoa(score))
			if len(signals) > 0 {
				w.Header().Set("X-Firewall-Signals", strings.Join(signals, ","))
			}
			if matchedRule == "" && policy.Decisions == nil && score >= policy.Heuristics.Threshold {
				matchedRule = "heuristic_score"
			}
		}
		detectorUnavailable := false
		if policy.Classifier != nil {
			scores, available := classifyMessages(r.Context(), *detector, body)
			if available {
				highest := 0
				for _, score := range scores {
					if score > highest {
						highest = score
					}
				}
				w.Header().Set("X-Firewall-Classifier-Score", strconv.Itoa(highest))
				w.Header().Set("X-Firewall-Detector-Status", "ok")
				if matchedRule == "" {
					if policy.Decisions != nil {
						matchedRule = decisionRule(messages, scores, policy.Decisions, policy.Heuristics, policy.Normalization)
					} else if highest >= policy.Classifier.Threshold {
						matchedRule = "classifier_score"
					}
				}
			} else {
				w.Header().Set("X-Firewall-Detector-Status", "unavailable")
				detectorUnavailable = policy.Classifier.FailureMode == "fail_closed"
				if detectorUnavailable {
					matchedRule = "detector_unavailable"
				} else if matchedRule == "" && policy.Heuristics != nil && heuristicScore >= policy.Heuristics.Threshold {
					matchedRule = "heuristic_score"
				}
			}
		}
		if matchedRule != "" {
			if policy.Mode != "log" || detectorUnavailable {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{
					"message": "Blocked by firewall policy", "type": "firewall_block", "rule": matchedRule,
				}})
				return
			}
			w.Header().Set("X-Firewall-Match", matchedRule)
		}
		if policy.DLP != nil {
			body = redactRequest(body, redactionPatterns)
		}
		r.ContentLength = int64(len(body))
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

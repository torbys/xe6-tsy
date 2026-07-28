package qwen

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/1024XEngineer/xe6-tsy/services/realtime-audio/translate"
)

func TestProviderTranslatesWithQwenChatCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/compatible-mode/v1/chat/completions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("authorization = %q", r.Header.Get("Authorization"))
		}
		var request struct {
			Model          string `json:"model"`
			EnableThinking bool   `json:"enable_thinking"`
			Messages       []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if request.Model != "qwen3.6-flash" || request.EnableThinking || len(request.Messages) != 2 || request.Messages[1].Content != "你好" {
			t.Errorf("request = %#v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hello"}}],"usage":{"prompt_tokens":4,"completion_tokens":2}}`))
	}))
	defer server.Close()

	provider, err := NewProvider(Config{APIKey: "test-key", BaseURL: server.URL + "/compatible-mode/v1"})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	result, err := provider.Translate(context.Background(), translate.Request{Text: "你好", SourceLanguage: "zh-CN", TargetLanguage: "en-US"})
	if err != nil {
		t.Fatalf("Translate() error = %v", err)
	}
	if result.Text != "hello" || result.Provider != "aliyun" || result.Model != "qwen3.6-flash" || result.InputTokens != 4 || result.OutputTokens != 2 {
		t.Fatalf("result = %#v", result)
	}
}

func TestProviderDoesNotFollowRedirects(t *testing.T) {
	targetCalled := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetCalled = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"unexpected"}}]}`))
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer source.Close()

	provider, err := NewProvider(Config{APIKey: "test-key", BaseURL: source.URL})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	_, err = provider.Translate(context.Background(), translate.Request{Text: "你好", SourceLanguage: "zh-CN", TargetLanguage: "en-US"})
	if err == nil || !strings.Contains(err.Error(), "HTTP 302") {
		t.Fatalf("Translate() error = %v, want HTTP 302", err)
	}
	if targetCalled {
		t.Fatal("translation client followed a redirect")
	}
}

func TestProviderRejectsProtocolErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status_code":400,"message":"bad request"}`))
	}))
	defer server.Close()

	provider, err := NewProvider(Config{APIKey: "test-key", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewProvider() error = %v", err)
	}
	_, err = provider.Translate(context.Background(), translate.Request{Text: "你好", SourceLanguage: "zh-CN", TargetLanguage: "en-US"})
	if err == nil || !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("Translate() error = %v, want protocol status error", err)
	}
}

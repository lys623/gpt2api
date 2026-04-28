package image

import (
	"reflect"
	"testing"

	"github.com/432539/gpt2api/internal/upstream/chatgpt"
)

func TestFilterOutReferenceFileIDs(t *testing.T) {
	refs := []*chatgpt.UploadedFile{
		{FileID: "file_uploaded_ref"},
		{FileID: " file-service://file_uploaded_ref_with_prefix "},
		nil,
	}

	refSet := referenceUploadFileIDSet(refs)
	got := filterOutReferenceFileIDs([]string{
		"file_uploaded_ref",
		"file_generated_result",
		"sed:file_uploaded_ref",
		"file_uploaded_ref_with_prefix",
		"file-service://file_uploaded_ref_with_prefix",
		"sed:file_generated_sediment",
	}, refSet)

	want := []string{
		"file_generated_result",
		"sed:file_generated_sediment",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filterOutReferenceFileIDs() = %#v, want %#v", got, want)
	}
}

func TestTaskErrorDetailKeepsUpstreamMessage(t *testing.T) {
	got := taskErrorDetail(ErrUpstreamRejected, "非常抱歉，该提示可能违反了关于与第三方内容相似性的防护限制。")
	want := "upstream_rejected: 非常抱歉，该提示可能违反了关于与第三方内容相似性的防护限制。"
	if got != want {
		t.Fatalf("taskErrorDetail() = %q, want %q", got, want)
	}
}

func TestTruncateIsRuneSafe(t *testing.T) {
	got := truncate("非常抱歉abcdef", 4)
	if got != "非常抱歉" {
		t.Fatalf("truncate() = %q, want 非常抱歉", got)
	}
}

func TestAssistantFailureCode(t *testing.T) {
	got := assistantFailureCode("非常抱歉，该提示可能违反了防护限制。", ErrPollTimeout)
	if got != ErrUpstreamRejected {
		t.Fatalf("assistantFailureCode() = %q, want %q", got, ErrUpstreamRejected)
	}
	got = assistantFailureCode("It looks like you've uploaded an image! How can I assist you with it? Would you like to modify or generate a new image based on it?", ErrPollTimeout)
	if got != ErrUpstreamRejected {
		t.Fatalf("assistantFailureCode(upload handoff) = %q, want %q", got, ErrUpstreamRejected)
	}
	got = assistantFailureCode("图片仍在生成中", ErrPollTimeout)
	if got != ErrPollTimeout {
		t.Fatalf("assistantFailureCode(non-rejection) = %q, want %q", got, ErrPollTimeout)
	}
}

func TestAssistantFailureCodeClassifiesPoll429AsRateLimited(t *testing.T) {
	msg := `conversation poll hit 3 consecutive 429 after 25 attempts: conversation poll upstream http=429 body={"detail":"Too many requests"}`
	got := assistantFailureCode(msg, ErrUpstream)
	if got != ErrRateLimited {
		t.Fatalf("assistantFailureCode(poll 429) = %q, want %q", got, ErrRateLimited)
	}
}

func TestAssistantFailureCodeClassifiesPlusPlanLimitAsRateLimited(t *testing.T) {
	msg := "You've hit the plus plan limit for image generation requests. You can create more images when the limit resets in 1 hour and 11 minutes."
	got := assistantFailureCode(msg, ErrUpstream)
	if got != ErrRateLimited {
		t.Fatalf("assistantFailureCode(plus plan limit) = %q, want %q", got, ErrRateLimited)
	}
}

func TestSkippedMainlineIsNotClassifiedAsRejected(t *testing.T) {
	err := &chatgpt.UpstreamError{Status: 400, Message: "f/conversation failed", Body: `{"skipped_mainline":true}`}
	var r Runner
	if got := r.classifyUpstream(err); got != ErrNetworkTransient {
		t.Fatalf("classifyUpstream() = %q, want %q", got, ErrNetworkTransient)
	}

	msg := runnerErrorMessage(err)
	if msg == "" || msg == `{"skipped_mainline":true}` {
		t.Fatalf("runnerErrorMessage() = %q, want friendly message", msg)
	}
}

func TestSkippedMainlineDetectionIsRobust(t *testing.T) {
	bodies := []string{
		`{"skipped_mainline":true}`,
		"{\n  \"skipped_mainline\" : true\n}",
		`{"error":{"skipped_mainline":true}}`,
	}
	for _, body := range bodies {
		if !upstreamBodySkippedMainline(body) {
			t.Fatalf("upstreamBodySkippedMainline(%q) = false, want true", body)
		}
	}
}

package image

import (
	"bytes"
	stdimage "image"
	"image/color"
	"image/png"
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
		"sed:file_uploaded_ref",
		"sed:file_generated_sediment",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filterOutReferenceFileIDs() = %#v, want %#v", got, want)
	}
}

func TestMatchesReferenceImageContent(t *testing.T) {
	ref := testPNG(t, color.RGBA{R: 240, G: 120, B: 20, A: 255})
	other := testPNG(t, color.RGBA{R: 20, G: 120, B: 240, A: 255})

	fps := referenceImageFingerprints([]ReferenceImage{{Data: ref, FileName: "ref.png"}})
	if !matchesReferenceImage(ref, fps) {
		t.Fatalf("matchesReferenceImage(ref) = false, want true")
	}
	if matchesReferenceImage(other, fps) {
		t.Fatalf("matchesReferenceImage(other) = true, want false")
	}
}

func TestTaskErrorDetailKeepsUpstreamMessage(t *testing.T) {
	got := taskErrorDetail(ErrUpstreamRejected, "非常抱歉，该提示可能违反了关于与第三方内容相似性的防护限制。")
	want := "upstream_rejected: 非常抱歉，该提示可能违反了关于与第三方内容相似性的防护限制。"
	if got != want {
		t.Fatalf("taskErrorDetail() = %q, want %q", got, want)
	}
}

func testPNG(t *testing.T, c color.RGBA) []byte {
	t.Helper()
	img := imageNewRGBA(8, 8, c)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png encode: %v", err)
	}
	return buf.Bytes()
}

func imageNewRGBA(w, h int, c color.RGBA) *stdimage.RGBA {
	img := stdimage.NewRGBA(stdimage.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
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
	got = assistantFailureCode("图片仍在生成中", ErrPollTimeout)
	if got != ErrPollTimeout {
		t.Fatalf("assistantFailureCode(non-rejection) = %q, want %q", got, ErrPollTimeout)
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

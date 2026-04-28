package account

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFlexibleStringListAcceptsStringsAndObjects(t *testing.T) {
	var got flexibleStringList
	data := []byte(`[
		"image_gen",
		{"feature_name":"voice_mode"},
		{"name":"canvas"},
		{"code":"search"},
		{"unexpected":true}
	]`)
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	want := flexibleStringList{
		"image_gen",
		"voice_mode",
		"canvas",
		"search",
		`{"unexpected":true}`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestFlexibleStringListAcceptsSingleObject(t *testing.T) {
	var got flexibleStringList
	if err := json.Unmarshal([]byte(`{"feature":"image_gen"}`), &got); err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}
	want := flexibleStringList{"image_gen"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

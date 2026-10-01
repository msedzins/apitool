package tui

import "testing"

func TestStructuredJSONSupportsNumericObjectKeys(t *testing.T) {
	text := `{"123":"before"}`
	scalars, err := jsonScalars(text)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := setJSONScalar(text, scalars[0].path, "after")
	if err != nil {
		t.Fatal(err)
	}
	result, err := jsonScalars(updated)
	if err != nil {
		t.Fatal(err)
	}
	if got := result[0].value; got != "after" {
		t.Fatalf("numeric-key value = %#v", got)
	}
}

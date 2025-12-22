package commands

import "testing"

func TestParseMessagesFlags(t *testing.T) {
	f := parseMessagesFlags("authbackend", "question", 10, "authbackend")
	if f.FromID != "authbackend" || f.Type != "question" || f.Limit != 10 || f.ReaderID != "authbackend" {
		t.Fatalf("unexpected filter: %#v", f)
	}
}

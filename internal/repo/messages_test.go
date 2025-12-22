package repo

import "testing"

func TestMessagesFilterByType(t *testing.T) {
	db := openTestDB(t)

	_ = CreateMessage(db, Message{ID: "m1", FromID: "agent-1", Type: "say", Content: "hello"})
	_ = CreateMessage(db, Message{ID: "m2", FromID: "agent-1", Type: "question", Content: "help"})

	msgs, err := ListMessages(db, MessageFilter{Type: "question"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(msgs) != 1 || msgs[0].ID != "m2" {
		t.Fatalf("unexpected messages: %#v", msgs)
	}
}

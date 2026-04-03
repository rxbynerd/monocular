package sse

import (
	"testing"
)

func TestFeedLineSimpleEvent(t *testing.T) {
	p := NewSSEParser()

	ev, err := p.FeedLine(`data: {"directory":"/d","payload":{"type":"session.created","properties":{"sessionID":"s1"}}}`)
	if err != nil {
		t.Fatal(err)
	}
	if ev != nil {
		t.Fatal("expected nil event before blank line")
	}

	ev, err = p.FeedLine("")
	if err != nil {
		t.Fatal(err)
	}
	if ev == nil {
		t.Fatal("expected event after blank line")
	}
	if ev.Directory != "/d" {
		t.Errorf("directory = %q, want /d", ev.Directory)
	}
	if ev.Payload.Type != "session.created" {
		t.Errorf("type = %q, want session.created", ev.Payload.Type)
	}
	if ev.Payload.Properties["sessionID"] != "s1" {
		t.Errorf("sessionID = %v, want s1", ev.Payload.Properties["sessionID"])
	}
}

func TestFeedLineMissingDirectory(t *testing.T) {
	p := NewSSEParser()
	p.FeedLine(`data: {"payload":{"type":"server.connected","properties":{}}}`)
	ev, err := p.FeedLine("")
	if err != nil {
		t.Fatal(err)
	}
	if ev == nil {
		t.Fatal("expected event")
	}
	if ev.Directory != "" {
		t.Errorf("directory = %q, want empty", ev.Directory)
	}
	if ev.Payload.Type != "server.connected" {
		t.Errorf("type = %q, want server.connected", ev.Payload.Type)
	}
}

func TestFeedLineMultilineData(t *testing.T) {
	p := NewSSEParser()
	p.FeedLine(`data: {"directory":"/d",`)
	p.FeedLine(`data: "payload":{"type":"test","properties":{}}}`)
	ev, err := p.FeedLine("")
	if err != nil {
		t.Fatal(err)
	}
	if ev == nil {
		t.Fatal("expected event from multiline data")
	}
	if ev.Payload.Type != "test" {
		t.Errorf("type = %q, want test", ev.Payload.Type)
	}
}

func TestFeedLineCommentIgnored(t *testing.T) {
	p := NewSSEParser()
	ev, err := p.FeedLine(": this is a comment")
	if err != nil {
		t.Fatal(err)
	}
	if ev != nil {
		t.Fatal("comment should not produce event")
	}
}

func TestFeedLineIDAndRetryIgnored(t *testing.T) {
	p := NewSSEParser()

	ev, err := p.FeedLine("id: 123")
	if err != nil || ev != nil {
		t.Fatal("id: line should be ignored")
	}

	ev, err = p.FeedLine("retry: 5000")
	if err != nil || ev != nil {
		t.Fatal("retry: line should be ignored")
	}
}

func TestFeedLineMalformedJSON(t *testing.T) {
	p := NewSSEParser()
	p.FeedLine("data: {not valid json")
	_, err := p.FeedLine("")
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestFeedLineEmptyLineWithoutData(t *testing.T) {
	p := NewSSEParser()
	ev, err := p.FeedLine("")
	if err != nil {
		t.Fatal(err)
	}
	if ev != nil {
		t.Fatal("empty line without prior data should not produce event")
	}
}

func TestFeedLineDataWithoutSpace(t *testing.T) {
	// "data:" without space after colon should still work
	p := NewSSEParser()
	p.FeedLine(`data:{"payload":{"type":"test","properties":{}}}`)
	ev, err := p.FeedLine("")
	if err != nil {
		t.Fatal(err)
	}
	if ev == nil {
		t.Fatal("expected event")
	}
	if ev.Payload.Type != "test" {
		t.Errorf("type = %q, want test", ev.Payload.Type)
	}
}

func TestFeedLineMultipleEvents(t *testing.T) {
	p := NewSSEParser()

	// First event
	p.FeedLine(`data: {"payload":{"type":"ev1","properties":{}}}`)
	ev1, err := p.FeedLine("")
	if err != nil {
		t.Fatal(err)
	}
	if ev1 == nil || ev1.Payload.Type != "ev1" {
		t.Fatalf("expected ev1, got %v", ev1)
	}

	// Second event
	p.FeedLine(`data: {"payload":{"type":"ev2","properties":{}}}`)
	ev2, err := p.FeedLine("")
	if err != nil {
		t.Fatal(err)
	}
	if ev2 == nil || ev2.Payload.Type != "ev2" {
		t.Fatalf("expected ev2, got %v", ev2)
	}
}

func TestReset(t *testing.T) {
	p := NewSSEParser()
	p.FeedLine(`data: partial data`)
	p.Reset()

	// After reset, empty line should not produce event
	ev, err := p.FeedLine("")
	if err != nil {
		t.Fatal(err)
	}
	if ev != nil {
		t.Fatal("expected nil after reset")
	}
}

func TestFeedLineHeartbeat(t *testing.T) {
	p := NewSSEParser()
	p.FeedLine(`data: {"payload":{"type":"server.heartbeat","properties":{}}}`)
	ev, err := p.FeedLine("")
	if err != nil {
		t.Fatal(err)
	}
	if ev == nil {
		t.Fatal("expected heartbeat event")
	}
	if ev.Payload.Type != "server.heartbeat" {
		t.Errorf("type = %q, want server.heartbeat", ev.Payload.Type)
	}
	if ev.Directory != "" {
		t.Errorf("heartbeat should have empty directory, got %q", ev.Directory)
	}
}

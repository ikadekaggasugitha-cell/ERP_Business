package eventbus

import (
	"testing"
)

func TestTopicAndDLQMapping(t *testing.T) {
	if got := TopicFor("gl.journal.posted"); got != "erp.gl.journal.posted.v1" {
		t.Fatalf("TopicFor = %q", got)
	}
	if got := DLQTopic("erp.gl.journal.posted.v1"); got != "erp.gl.journal.posted.v1.dlq" {
		t.Fatalf("DLQTopic = %q", got)
	}
}


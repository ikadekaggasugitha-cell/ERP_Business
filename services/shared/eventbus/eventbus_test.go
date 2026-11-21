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

func TestPartitionKeyCombinesOrgAndAggregate(t *testing.T) {
	if got := PartitionKey("org-1", "agg-9"); got != "org-1:agg-9" {
		t.Fatalf("PartitionKey = %q, want org-1:agg-9", got)
	}
	if got := PartitionKey("org-1", ""); got != "org-1" {
		t.Fatalf("PartitionKey org only = %q, want org-1", got)
	}
	if got := PartitionKey("", "agg-9"); got != "agg-9" {
		t.Fatalf("PartitionKey agg only = %q, want agg-9", got)
	}
}


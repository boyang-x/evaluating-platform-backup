package repository

import (
	"strings"
	"testing"
)

func TestPlatformDataPublishedWhereClauseRequiresPublishedPublicRows(t *testing.T) {
	where, _ := platformDataPublishedWhereClause("sub_type", "", 0)

	for _, want := range []string{"status = 'published'", "visibility = 'public'"} {
		if !strings.Contains(where, want) {
			t.Fatalf("where clause %q missing %q", where, want)
		}
	}
}

func TestPlatformDataPublishedWhereClauseKeepsSubtypeFilter(t *testing.T) {
	where, args := platformDataPublishedWhereClause("sub_type", "jailbreak_question", 0)

	if !strings.Contains(where, "sub_type = $1") {
		t.Fatalf("where clause %q missing subtype placeholder", where)
	}
	if len(args) != 1 || args[0] != "jailbreak_question" {
		t.Fatalf("args = %#v, want subtype argument", args)
	}
}

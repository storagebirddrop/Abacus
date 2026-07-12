package common

import "testing"

func TestSyntheticExternalID_Deterministic(t *testing.T) {
	id1 := SyntheticExternalID("2024-01-15T10:00:00Z", "buy", "0.01", "500.00", "1.50")
	id2 := SyntheticExternalID("2024-01-15T10:00:00Z", "buy", "0.01", "500.00", "1.50")
	if id1 != id2 {
		t.Fatalf("same input produced different IDs: %q vs %q", id1, id2)
	}
	if id1 == "" {
		t.Fatal("expected a non-empty ID")
	}
}

func TestSyntheticExternalID_DifferentInputsDiffer(t *testing.T) {
	id1 := SyntheticExternalID("2024-01-15T10:00:00Z", "buy", "0.01", "500.00", "1.50")
	id2 := SyntheticExternalID("2024-01-16T10:00:00Z", "buy", "0.01", "500.00", "1.50")
	if id1 == id2 {
		t.Fatal("different timestamps produced the same ID")
	}
}

func TestSyntheticExternalID_NoAmbiguousConcatenation(t *testing.T) {
	// Without a separator between parts, ("ab","c") and ("a","bc") would hash
	// identically. Confirm the separator prevents that.
	id1 := SyntheticExternalID("ab", "c")
	id2 := SyntheticExternalID("a", "bc")
	if id1 == id2 {
		t.Fatal("concatenation ambiguity: (\"ab\",\"c\") collided with (\"a\",\"bc\")")
	}
}

package rag

import "testing"

func TestVectorTableUsesSafeIdentifierAndLiteral(t *testing.T) {
	table, err := vectorTable("Team_Index_01")
	if err != nil || table != "kb_team_index_01" {
		t.Fatalf("table=%q err=%v", table, err)
	}
	if _, err := vectorTable("bad-name;drop table"); err == nil {
		t.Fatal("expected invalid identifier rejection")
	}
	if actual := vectorLiteral([]float32{1, 0.5, -2}); actual != "[1,0.5,-2]" {
		t.Fatalf("unexpected vector literal: %s", actual)
	}
}

func TestSplitTextKeepsUTF8BoundariesAndOverlap(t *testing.T) {
	chunks := SplitText("甲乙丙丁戊己", 3, 1)
	if len(chunks) != 3 || chunks[0] != "甲乙丙" || chunks[1] != "丙丁戊" || chunks[2] != "戊己" {
		t.Fatalf("unexpected chunks: %#v", chunks)
	}
}

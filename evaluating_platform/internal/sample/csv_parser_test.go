package sample

import (
	"testing"
)

func TestParseTwoColumnCSV_ValidData(t *testing.T) {
	data := []byte("1,hello world\n2,foo bar\n3,baz\n")
	rows, err := ParseTwoColumnCSV(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if rows[0].Index != 1 || rows[0].Data != "hello world" {
		t.Errorf("row 0: got {%d, %q}, want {1, \"hello world\"}", rows[0].Index, rows[0].Data)
	}
	if rows[1].Index != 2 || rows[1].Data != "foo bar" {
		t.Errorf("row 1: got {%d, %q}, want {2, \"foo bar\"}", rows[1].Index, rows[1].Data)
	}
	if rows[2].Index != 3 || rows[2].Data != "baz" {
		t.Errorf("row 2: got {%d, %q}, want {3, \"baz\"}", rows[2].Index, rows[2].Data)
	}
}

func TestParseTwoColumnCSV_QuotedFields(t *testing.T) {
	data := []byte("1,\"hello, world\"\n2,\"line1\nline2\"\n")
	rows, err := ParseTwoColumnCSV(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Data != "hello, world" {
		t.Errorf("row 0 data: got %q, want \"hello, world\"", rows[0].Data)
	}
	if rows[1].Data != "line1\nline2" {
		t.Errorf("row 1 data: got %q, want \"line1\\nline2\"", rows[1].Data)
	}
}

func TestParseTwoColumnCSV_EmptyFile(t *testing.T) {
	_, err := ParseTwoColumnCSV([]byte(""))
	if err == nil {
		t.Fatal("expected error for empty file")
	}
	want := "CSV file is empty"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestParseTwoColumnCSV_WrongColumnCount(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"one column", "1\n2\n"},
		{"three columns", "1,hello,extra\n"},
	}
	want := "CSV must have exactly 2 columns: index (integer) and data"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTwoColumnCSV([]byte(tt.data))
			if err == nil {
				t.Fatal("expected error for wrong column count")
			}
			if err.Error() != want {
				t.Errorf("error = %q, want %q", err.Error(), want)
			}
		})
	}
}

func TestParseTwoColumnCSV_NonIntegerIndex(t *testing.T) {
	data := []byte("1,ok\nabc,bad\n")
	_, err := ParseTwoColumnCSV(data)
	if err == nil {
		t.Fatal("expected error for non-integer index")
	}
	want := "column 1 must be integer, got 'abc' at row 2"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestParseTwoColumnCSV_SingleRow(t *testing.T) {
	data := []byte("42,single row data\n")
	rows, err := ParseTwoColumnCSV(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].Index != 42 || rows[0].Data != "single row data" {
		t.Errorf("got {%d, %q}, want {42, \"single row data\"}", rows[0].Index, rows[0].Data)
	}
}

func TestParseTwoColumnCSV_NegativeIndex(t *testing.T) {
	data := []byte("-1,negative index\n")
	rows, err := ParseTwoColumnCSV(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows[0].Index != -1 {
		t.Errorf("index = %d, want -1", rows[0].Index)
	}
}

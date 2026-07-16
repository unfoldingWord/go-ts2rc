package books

import "testing"

func TestLookup(t *testing.T) {
	gen, ok := Lookup("gen")
	if !ok {
		t.Fatal("expected gen")
	}
	if gen.Title != "Genesis" || gen.USFMNum != "01" || gen.Category != "bible-ot" {
		t.Fatalf("unexpected gen: %+v", gen)
	}

	mat, ok := Lookup("MAT")
	if !ok {
		t.Fatal("expected MAT")
	}
	if mat.USFMNum != "41" || mat.Category != "bible-nt" {
		t.Fatalf("unexpected mat: %+v", mat)
	}

	if _, ok := Lookup("notabook"); ok {
		t.Fatal("expected miss")
	}
}

func TestAllBooksCount(t *testing.T) {
	if len(AllBooks) != 66 {
		t.Fatalf("want 66 books, got %d", len(AllBooks))
	}
}

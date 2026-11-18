package data

import "testing"

func TestReferencedBlobNamesKeepsBareFilenames(t *testing.T) {
	names := ReferencedBlobNames([]string{"abc123", "dir/abc123", "", ".", "..", `windows\abc123`})
	if _, ok := names["abc123"]; !ok {
		t.Fatalf("bare filename missing from %v", names)
	}
	if len(names) != 1 {
		t.Fatalf("referenced names = %v, want only the bare filename", names)
	}
}

package dbtype

import (
	"reflect"
	"testing"
)

func TestSupported(t *testing.T) {
	want := []string{MySQL, Postgres, Mongo, Redis}
	if got := Supported(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Supported() = %#v, want %#v", got, want)
	}
}

func TestIsSupported(t *testing.T) {
	for _, dbType := range Supported() {
		if !IsSupported(dbType) {
			t.Fatalf("IsSupported(%q) = false, want true", dbType)
		}
	}
	if IsSupported("sqlite") {
		t.Fatal("IsSupported(sqlite) = true, want false")
	}
}

func TestNames(t *testing.T) {
	if got := Names(); got != "mysql, postgres, mongo, redis" {
		t.Fatalf("Names() = %q", got)
	}
}

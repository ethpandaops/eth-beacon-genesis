package beaconchain

import (
	"reflect"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec/heze"
)

func assertExtends(t *testing.T, extended, base reflect.Type, newFields int) {
	t.Helper()

	if extended.NumField() != base.NumField()+newFields {
		t.Fatalf("expected %d fields, got %d", base.NumField()+newFields, extended.NumField())
	}

	for i := range base.NumField() {
		baseField := base.Field(i)
		extendedField := extended.Field(i)

		if baseField.Name != extendedField.Name || baseField.Type != extendedField.Type {
			t.Fatalf("field %d: expected %s %s, got %s %s", i, baseField.Name, baseField.Type, extendedField.Name, extendedField.Type)
		}

		for _, tag := range []string{"ssz-index", "ssz-size", "ssz-max", "ssz-type", "dynssz-size", "dynssz-max"} {
			if baseField.Tag.Get(tag) != extendedField.Tag.Get(tag) {
				t.Fatalf("field %s: expected %s tag %q, got %q", baseField.Name, tag, baseField.Tag.Get(tag), extendedField.Tag.Get(tag))
			}
		}
	}
}

func TestEIP8450TypesExtendHeze(t *testing.T) {
	assertExtends(t, reflect.TypeFor[EIP8450BeaconState](), reflect.TypeFor[heze.BeaconState](), 4)
	assertExtends(t, reflect.TypeFor[EIP8450BeaconBlockBody](), reflect.TypeFor[heze.BeaconBlockBody](), 1)
}

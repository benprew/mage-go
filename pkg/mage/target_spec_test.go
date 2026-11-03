package mage

import (
	"reflect"
	"testing"
)

func TestTargetSpecificationHasNoAnnouncementState(t *testing.T) {
	targetType := reflect.TypeFor[Target]()
	for _, name := range []string{"Choose", "Chosen", "IsChosen", "Reset"} {
		if _, ok := targetType.MethodByName(name); ok {
			t.Errorf("Target exposes mutable announcement method %s", name)
		}
	}
	if _, ok := reflect.TypeFor[BaseTarget]().FieldByName("chosen"); ok {
		t.Error("BaseTarget stores announcement choices")
	}
}

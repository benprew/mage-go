package mage

import (
	"reflect"

	"github.com/google/uuid"
)

func cloneTokenAbility(ability Ability) Ability {
	value := reflect.ValueOf(ability)
	if value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		panic("token ability must be a non-nil struct pointer")
	}
	clone := reflect.New(value.Elem().Type())
	clone.Elem().Set(value.Elem())
	base := clone.Elem().FieldByName("BaseAbility")
	if !base.IsValid() || !base.CanAddr() {
		panic("token ability must embed BaseAbility")
	}
	state, ok := base.Addr().Interface().(*BaseAbility)
	if !ok {
		panic("token ability must embed BaseAbility")
	}
	state.id = uuid.New()
	state.source = uuid.Nil
	state.controller = uuid.Nil
	return clone.Interface().(Ability)
}

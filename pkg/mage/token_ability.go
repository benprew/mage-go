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
	result := clone.Interface().(Ability)
	if holder, ok := result.(*StaticAbilityHolder); ok {
		holder.Effects = make([]ContinuousEffect, len(holder.Effects))
		original := ability.(*StaticAbilityHolder)
		for j, effect := range original.Effects {
			if cloned, ok := effect.(deepCloneableEffect); ok {
				holder.Effects[j] = cloned.cloneEffect()
			} else {
				// Each copy needs its own source identity.
				value := reflect.ValueOf(effect)
				if value.Kind() == reflect.Pointer && value.Elem().Kind() == reflect.Struct {
					clone := reflect.New(value.Elem().Type())
					clone.Elem().Set(value.Elem())
					holder.Effects[j] = clone.Interface().(ContinuousEffect)
				} else {
					holder.Effects[j] = effect
				}
			}
		}
	}
	return result
}

package mage

import (
	"testing"
)

func TestDirectDestructionInspection(t *testing.T) {
	target := DeclarePermanentTarget("subject", IsLand)
	set := &destructionInspectionSetTrap{}
	effect := Sequence(Destroy(target), CompositeEffects("batch", DestroyEachWithoutRegeneration(set)), DestroyWithoutRegeneration(This))
	instructions, ok := InspectDirectDestruction(effect)
	if !ok || len(instructions) != 3 {
		t.Fatalf("instructions = %v, supported = %v", instructions, ok)
	}
	if instructions[0].Subject != target || instructions[0].Subjects != nil || instructions[0].CannotRegenerate {
		t.Fatal("singular instruction lost its target declaration")
	}
	if instructions[1].Subject != nil || instructions[1].Subjects != set || !instructions[1].CannotRegenerate {
		t.Fatal("batch instruction lost its set or regeneration restriction")
	}
	if instructions[2].Subject != This || instructions[2].Subjects != nil || !instructions[2].CannotRegenerate {
		t.Fatal("sequence order or singular regeneration restriction changed")
	}
	instructions[0].Subject = This
	again, ok := InspectDirectDestruction(effect)
	if !ok || again[0].Subject != target {
		t.Fatal("caller changed the declaration through the inspection result")
	}
	for _, empty := range []ResolvingEffect{Sequence(), CompositeEffects("empty")} {
		if instructions, ok := InspectDirectDestruction(empty); !ok || len(instructions) != 0 {
			t.Fatal("empty unconditional sequence was not supported")
		}
	}
}

type destructionInspectionSetTrap struct{}

func (*destructionInspectionSetTrap) permanentSet() {}
func (*destructionInspectionSetTrap) resolvePermanents(*EffectContext) []*Permanent {
	panic("inspection resolved a collection")
}

type destructionInspectionTrap struct{}

func (destructionInspectionTrap) Text() string { panic("inspection called Text") }
func (destructionInspectionTrap) Properties() EffectProperties {
	panic("inspection called Properties")
}
func (destructionInspectionTrap) Apply(*EffectContext) error { panic("inspection executed effect") }

func TestDirectDestructionUnsupported(t *testing.T) {
	count := BindNumber("destroyed")
	var typedNil *destroySubjectEffect
	var nilSequence *sequenceEffect
	var nilComposite *compositeEffect
	for name, effect := range map[string]ResolvingEffect{
		"nil":            nil,
		"typed nil":      typedNil,
		"nil sequence":   nilSequence,
		"nil composite":  nilComposite,
		"nil subject":    Destroy(nil),
		"nil set":        DestroyEach(nil),
		"unknown":        destructionInspectionTrap{},
		"prefix":         Sequence(Destroy(This), destructionInspectionTrap{}),
		"suffix":         CompositeEffects("unknown", destructionInspectionTrap{}, Destroy(This)),
		"nil child":      Sequence(Destroy(This), nil),
		"condition":      IfState(SourceTappedCondition(), Destroy(This)),
		"number result":  DestroyWithResult(This, DestroyedThisWayInto(count)),
		"batch result":   DestroyEach(MatchingPermanents(IsLand), PermanentsPutIntoGraveyardThisWayInto(count)),
		"invalid result": DestroyEach(MatchingPermanents(IsLand), nil),
	} {
		t.Run(name, func(t *testing.T) {
			if instructions, ok := InspectDirectDestruction(effect); ok || instructions != nil {
				t.Fatalf("unsupported declaration returned a partial plan: %v, %v", instructions, ok)
			}
		})
	}
}

func TestDirectDestructionInspectionDoesNotResolveOperands(t *testing.T) {
	filter := NewPermanentFilter("opaque", func(*Permanent, *Game) bool {
		panic("inspection ran a filter")
	})
	target := DeclarePermanentTarget("target", filter)
	set := MatchingPermanents(filter)
	for _, effect := range []ResolvingEffect{Destroy(target), DestroyEach(set)} {
		if instructions, ok := InspectDirectDestruction(effect); !ok || len(instructions) != 1 {
			t.Fatal("inspection evaluated an operand")
		}
	}
}

package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestManaValueOfSpellStackXAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	variable := NewInstant("Variable", "{X}{X}{U}")
	variable.SetOwner(b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: variable.ID(), Card: variable, Controller: b.PlayerID(), XValue: 3}
	g.pushStack(obj)
	spell := DeclareSpellTarget("spell")
	value := ManaValueOfSpell(spell)
	ctx := &EffectContext{Game: g, targetBindings: map[*targetDeclaration][]capturedPermanent{spell.targetDeclaration: {{id: obj.ID}}}}
	if got := value.resolveValue(ctx); got != 7 {
		t.Fatalf("stack mana value = %d, want 7", got)
	}
	clone := g.Clone()
	cloneCtx := *ctx
	cloneCtx.Game = clone
	if got := value.resolveValue(&cloneCtx); got != 7 {
		t.Fatalf("clone mana value = %d", got)
	}
	clone.stack.RemoveByID(obj.ID)
	if got := value.resolveValue(&cloneCtx); got != 0 {
		t.Fatalf("departed clone spell = %d", got)
	}
	if got := value.resolveValue(ctx); got != 7 {
		t.Fatalf("original changed = %d", got)
	}
	g.stack.RemoveByID(obj.ID)
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: variable.ID(), Card: variable, Controller: b.PlayerID(), XValue: 5})
	if got := value.resolveValue(ctx); got != 0 {
		t.Fatalf("recast card matched old stack identity: %d", got)
	}
}

func TestManaValueOfSpellResolutionAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := NewInstant("Victim", "{3}{U}")
	victim.SetOwner(b.PlayerID())
	target := &StackObject{ID: uuid.New(), SourceID: victim.ID(), Card: victim, Controller: b.PlayerID()}
	g.pushStack(target)
	spell := DeclareSpellTarget("spell")
	ability := Spell(GainLifeFor(You, ManaValueOfSpell(spell)), WithTargets(spell))
	card := NewInstant("Reader", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{target.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.ResolveTopOfStack()
	if got := a.Life(); got != 24 {
		t.Fatalf("life = %d, want 24", got)
	}
	for _, tc := range []struct {
		name string
		ref  StackObjectRef
	}{
		{"nil", nil},
		{"undeclared", DeclareSpellTarget("other")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(You, ManaValueOfSpell(tc.ref)), WithTargets(spell))))
			if err := ValidateCardDefinition(bad.Name(), bad); err == nil || !strings.Contains(err.Error(), "spell") {
				t.Fatalf("validation error = %v", err)
			}
		})
	}
}

func TestManaValueOfSpellCopiedFrameAndTargetLegality(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	firstCard := NewInstant("First", "{X}{R}")
	secondCard := NewArtifact("Second", "{4}")
	firstCard.SetOwner(b.PlayerID())
	secondCard.SetOwner(b.PlayerID())
	first := &StackObject{ID: uuid.New(), SourceID: firstCard.ID(), Card: firstCard, Controller: b.PlayerID(), XValue: 2}
	second := &StackObject{ID: uuid.New(), SourceID: secondCard.ID(), Card: secondCard, Controller: b.PlayerID()}
	g.pushStack(first)
	g.pushStack(second)
	allowed := true
	filter := NewCardFilter("allowed", func(card Card) bool { return allowed && card.HasType(core.TypeInstant) })
	spell := DeclareSpellTarget("spell")
	ability := Spell(GainLifeFor(You, ManaValueOfSpell(spell)), WithTargets(spell))
	reader := NewInstant("Reader", "", WithAbility(ability))
	reader.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: reader.ID(), Card: reader, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	copyObj := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	copyObj.Targets[0] = second.ID
	copyObj.TargetZones[second.ID] = g.zoneOfTarget(second.ID)
	g.ResolveTopOfStack()
	if got := a.Life(); got != 24 {
		t.Fatalf("copy used wrong spell: life = %d", got)
	}
	g.ResolveTopOfStack()
	if got := a.Life(); got != 27 {
		t.Fatalf("original lost its target: life = %d", got)
	}
	filtered := DeclareSpellTarget("filtered", filter)
	filteredAbility := Spell(GainLifeFor(You, ManaValueOfSpell(filtered)), WithTargets(filtered))
	filteredReader := NewInstant("Filtered reader", "", WithAbility(filteredAbility))
	filteredReader.SetOwner(a.PlayerID())
	filteredObj := &StackObject{ID: uuid.New(), SourceID: filteredReader.ID(), Card: filteredReader, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID}, TargetSpecs: filteredAbility.Targets(), Effects: filteredAbility.Effects()}
	g.pushStack(filteredObj)
	allowed = false
	g.ResolveTopOfStack()
	if got := a.Life(); got != 27 {
		t.Fatalf("illegal target supplied mana value: life = %d", got)
	}
}

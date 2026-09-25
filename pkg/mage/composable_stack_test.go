package mage

import (
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func stackTestSpell(name string, owner uuid.UUID) Card {
	card := NewInstant(name, "")
	card.SetOwner(owner)
	return card
}

func TestTypedCounterUsesStackIdentityAndTargetOrder(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := stackTestSpell("Victim", b.PlayerID())
	first := &StackObject{ID: uuid.New(), SourceID: victim.ID(), Card: victim, Controller: b.PlayerID()}
	g.pushStack(first)
	copyObj := g.CopyStackObjectDirect(first, b.PlayerID(), false)
	if copyObj == nil {
		t.Fatal("copy failed")
	}
	if got := g.zoneOfTarget(copyObj.ID); got != ZoneStack {
		t.Fatalf("copy target zone = %v, want stack", got)
	}
	player := DeclarePlayerTarget("player")
	spell := DeclareSpellTarget("spell")
	body := Sequence(GainLifeFor(player, Fixed(1)), Counter(spell))
	ability := Spell(body, WithTargets(player, spell))
	card := NewInstant("Counter", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Counter", card); err != nil {
		t.Fatal(err)
	}
	possible := spell.Possible(a.PlayerID(), card, g)
	if !slices.Contains(possible, first.ID) || !slices.Contains(possible, copyObj.ID) {
		t.Fatalf("possible stack objects = %v", possible)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{a.PlayerID(), copyObj.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	g.ResolveTopOfStack()
	if g.stack.FindByID(copyObj.ID) != nil || g.stack.FindByID(first.ID) == nil || len(b.Graveyard()) != 0 || a.Life() != 21 {
		t.Fatal("counter changed the wrong stack object or lost the earlier effect")
	}
}

func TestTypedCounterLegalityAndUncounterable(t *testing.T) {
	for _, change := range []string{"none", "removed", "uncounterable", "filter changed"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			options := []CardOption{}
			if change == "uncounterable" {
				options = append(options, WithUncounterable())
			}
			victim := NewArtifact("Victim", "", options...)
			victim.SetOwner(b.PlayerID())
			first := &StackObject{ID: uuid.New(), SourceID: victim.ID(), Card: victim, Controller: b.PlayerID()}
			g.pushStack(first)
			allow := true
			filter := NewCardFilter("artifact", func(c Card) bool { return allow && c.HasType(TypeArtifact) })
			spell := DeclareSpellTarget("spell", filter)
			ability := Spell(Sequence(Counter(spell), GainLifeFor(You, Fixed(1))), WithTargets(spell))
			card := NewInstant("Counter", "", WithAbility(ability))
			card.SetOwner(a.PlayerID())
			if err := ValidateCardDefinition("Counter", card); err != nil {
				t.Fatal(err)
			}
			obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
			g.pushStack(obj)
			if change == "removed" {
				g.stack.RemoveByID(first.ID)
			}
			if change == "filter changed" {
				allow = false
			}
			g.ResolveTopOfStack()
			wantLife := 21
			if change == "removed" || change == "filter changed" {
				wantLife = 20
			}
			if a.Life() != wantLife {
				t.Fatalf("life = %d, want %d", a.Life(), wantLife)
			}
			countered := change == "none"
			if (g.stack.FindByID(first.ID) == nil) != (countered || change == "removed") {
				t.Fatal("wrong stack object status")
			}
			if (len(b.Graveyard()) == 1) != countered {
				t.Fatal("wrong graveyard status")
			}
		})
	}
}

func TestTypedCounterBindingValidation(t *testing.T) {
	spell := DeclareSpellTarget("spell")
	for _, tc := range []struct {
		name     string
		body     ResolvingEffect
		declared []TargetBinding
		want     string
	}{
		{"nil", Counter(nil), nil, ".spell"},
		{"undeclared", Counter(spell), nil, ".spell"},
		{"valid", Counter(spell), []TargetBinding{spell}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewInstant("Test", "", WithAbility(Spell(tc.body, WithTargets(tc.declared...))))
			err := ValidateCardDefinition("Test", card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestTypedCounterRecastAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	victim := stackTestSpell("Victim", b.PlayerID())
	first := &StackObject{ID: uuid.New(), SourceID: victim.ID(), Card: victim, Controller: b.PlayerID()}
	g.pushStack(first)
	spell := DeclareSpellTarget("spell")
	ability := Spell(Counter(spell), WithTargets(spell))
	card := NewInstant("Counter", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if clone.stack.FindByID(first.ID) != nil || g.stack.FindByID(first.ID) == nil {
		t.Fatal("clone changed original stack")
	}
	g.stack.RemoveByID(first.ID)
	second := &StackObject{ID: uuid.New(), SourceID: victim.ID(), Card: victim, Controller: b.PlayerID()}
	g.pushStack(second)
	if slices.Contains(spell.Possible(a.PlayerID(), card, g), first.ID) || !slices.Contains(spell.Possible(a.PlayerID(), card, g), second.ID) {
		t.Fatal("spell target did not distinguish casts")
	}
	g.stack.RemoveByID(obj.ID)
	g.ResolveStackObject(obj)
	if g.stack.FindByID(second.ID) == nil || len(b.Graveyard()) != 0 {
		t.Fatal("old target countered a later cast")
	}
}

func TestTypedCounterRejectsInvalidAnnouncement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spell := DeclareSpellTarget("spell", NewCardFilter("artifact", func(c Card) bool { return c.HasType(TypeArtifact) }))
	ability := Spell(Counter(spell), WithTargets(spell))
	card := NewInstant("Counter", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	wrong := stackTestSpell("Wrong", b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: wrong.ID(), Card: wrong, Controller: b.PlayerID()}
	g.pushStack(obj)
	if err := g.validateActionTargets(a.PlayerID(), card, ability.Targets(), []uuid.UUID{obj.ID}, 0, "spell"); err == nil {
		t.Fatal("accepted a spell outside the target filter")
	}
	if err := g.validateActionTargets(a.PlayerID(), card, ability.Targets(), []uuid.UUID{uuid.New()}, 0, "spell"); err == nil {
		t.Fatal("accepted an absent stack object")
	}
	if g.stack.FindByID(obj.ID) == nil {
		t.Fatal("invalid announcement changed the stack")
	}
}

func TestTypedCounterCopiedRetargetedInstance(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	firstCard := stackTestSpell("First", b.PlayerID())
	secondCard := stackTestSpell("Second", b.PlayerID())
	first := &StackObject{ID: uuid.New(), SourceID: firstCard.ID(), Card: firstCard, Controller: b.PlayerID()}
	second := &StackObject{ID: uuid.New(), SourceID: secondCard.ID(), Card: secondCard, Controller: b.PlayerID()}
	g.pushStack(first)
	g.pushStack(second)
	spell := DeclareSpellTarget("spell")
	ability := Spell(Counter(spell), WithTargets(spell))
	card := NewInstant("Counter", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	copyObj := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	copyObj.Targets[0] = second.ID
	copyObj.TargetZones[second.ID] = g.zoneOfTarget(second.ID)
	g.ResolveTopOfStack()
	if g.stack.FindByID(second.ID) != nil || g.stack.FindByID(first.ID) == nil || obj.Targets[0] != first.ID {
		t.Fatal("copied counter did not use its own target frame")
	}
	g.ResolveTopOfStack()
	if g.stack.FindByID(first.ID) != nil || len(b.Graveyard()) != 2 {
		t.Fatal("original counter did not retain its target")
	}
}

func TestTypedCounterModeScope(t *testing.T) {
	spell := DeclareSpellTarget("spell")
	valid := SpellModes(ModeOf("counter", Counter(spell), WithTargets(spell)), ModeOf("life", GainLifeFor(You, Fixed(1))))
	if err := ValidateCardDefinition("Modes", NewInstant("Modes", "", WithAbility(valid))); err != nil {
		t.Fatal(err)
	}
	invalid := SpellModes(ModeOf("counter", Counter(spell)), ModeOf("life", GainLifeFor(You, Fixed(1)), WithTargets(spell)))
	if err := ValidateCardDefinition("Modes", NewInstant("Modes", "", WithAbility(invalid))); err == nil {
		t.Fatal("accepted a spell binding from another mode")
	}
}

func TestCounterSpellOnStackPreservesAbilityCountering(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := &StackObject{ID: uuid.New(), SourceID: uuid.New(), Controller: b.PlayerID(), IsAbility: true}
	g.pushStack(ability)
	g.CounterSpellOnStack(ability.SourceID)
	if g.stack.FindByID(ability.ID) != nil {
		t.Fatal("legacy counter did not remove the ability")
	}
}

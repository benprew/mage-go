package mage

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTargetSubtypeDomain(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	wall := g.PutOnBattlefield(NewCreature("Wall", "", 0, 5, WithSubTypes("Wall")), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 5), a.PlayerID())
	binding := DeclarePermanentTarget("wall", AnyPermanent, TargetHasSubtype("Wall"))
	card := NewInstant("Tap", "", Spell(TapSubject(binding), WithTargets(binding)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	spec := card.Abilities()[0].(*ActionDefinition).Targets()[0]
	if got := TargetRequiredSubtypes(binding); !slices.Equal(got, []string{"Wall"}) {
		t.Fatalf("binding domain = %v", got)
	}
	if got := TargetRequiredSubtypes(spec); !slices.Equal(got, []string{"Wall"}) {
		t.Fatalf("domain = %v", got)
	}
	if got := spec.Possible(a.PlayerID(), SpellTargetSource(card), g); !slices.Equal(got, []uuid.UUID{wall.ID()}) || slices.Contains(got, other.ID()) {
		t.Fatalf("possible = %v", got)
	}
	binding.spec.(*PermanentTarget).Filters = nil
	if got := spec.Possible(a.PlayerID(), SpellTargetSource(card), g); !slices.Equal(got, []uuid.UUID{wall.ID()}) || binding.spec.(*PermanentTarget).Filter().Match(other, g) {
		t.Fatal("additional filters replaced the enforced constraint")
	}
	result := TargetRequiredSubtypes(spec)
	result[0] = "Elf"
	if TargetRequiredSubtypes(spec)[0] != "Wall" {
		t.Fatal("query changed the declaration")
	}
	g.PhaseOut(wall.ID())
	if len(spec.Possible(a.PlayerID(), SpellTargetSource(card), g)) != 0 || TargetRequiredSubtypes(spec)[0] != "Wall" {
		t.Fatal("phasing changed the domain or allowed an absent target")
	}
	g.RemoveFromBattlefield(g.FindPermanentIncludingPhased(wall.ID()))
	if TargetRequiredSubtypes(spec)[0] != "Wall" {
		t.Fatal("departure changed the domain")
	}
	if len(TargetRequiredSubtypes(TargetPermanent(HasSubType("Wall")))) != 0 || len(TargetRequiredSubtypes(TargetPlayer())) != 0 || len(TargetRequiredSubtypes(nil)) != 0 {
		t.Fatal("inferred an undeclared constraint")
	}
}

func TestTargetSubtypeDomainDeclarations(t *testing.T) {
	for _, subtype := range []string{"", " Wall", "Wall "} {
		binding := DeclarePermanentTarget("bad", AnyPermanent, TargetHasSubtype(subtype))
		card := NewInstant("Bad", "", Spell(TapSubject(binding), WithTargets(binding)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatalf("accepted subtype %q", subtype)
		}
		binding.spec = &countBoundTarget{inner: binding.spec, count: 1}
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("wrapper hid an invalid subtype")
		}
	}
	binding := DeclarePermanentTargets("both", 0, 2, AnyPermanent, TargetHasSubtype("Wall"), TargetHasSubtype("Elf"), TargetHasSubtype("Wall"))
	spec := binding.targetSpec()
	for _, wrapped := range []Target{spec, TargetRandom(spec), &countBoundTarget{inner: spec, count: 2}} {
		if got := TargetRequiredSubtypes(wrapped); !slices.Equal(got, []string{"Wall", "Elf"}) {
			t.Fatalf("wrapped domain = %v", got)
		}
	}
	g, a, _ := newSpellCopyGame()
	dual := g.PutOnBattlefield(NewCreature("Both", "", 1, 5, WithSubTypes("Wall", "Elf")), a.PlayerID())
	g.PutOnBattlefield(NewCreature("Wall", "", 0, 5, WithSubTypes("Wall")), a.PlayerID())
	if got := spec.Possible(a.PlayerID(), nil, g); !slices.Equal(got, []uuid.UUID{dual.ID()}) {
		t.Fatalf("conjunctive candidates = %v", got)
	}
}

func TestTargetSubtypeDomainCallers(t *testing.T) {
	g, a, _ := newSpellCopyGame()
	wall := g.PutOnBattlefield(NewCreature("Wall", "", 0, 5, WithSubTypes("Wall")), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 1, 5), a.PlayerID())
	binding := DeclarePermanentTarget("host", AnyPermanent, TargetHasSubtype("Wall"))
	aura := NewAura("Aura", "", WithCastTarget(binding))
	aura.SetOwner(a.PlayerID())
	wall.GrantBaseAttr(core.Shroud)
	if !auraHostIsLegal(aura, wall, g) || auraHostIsLegal(aura, other, g) {
		t.Error("attachment did not use the enforced domain")
	}
	if sameSpellTextTargetSpec(binding.spec, DeclarePermanentTarget("host", AnyPermanent).spec) || sameSpellTextTargetSpec(binding.spec, DeclarePermanentTarget("host", AnyPermanent, TargetHasSubtype("Elf")).spec) {
		t.Error("text continuation accepted a changed subtype constraint")
	}
	if !sameSpellTextTargetSpec(binding.spec, DeclarePermanentTarget("host", AnyPermanent, TargetHasSubtype("Wall")).spec) {
		t.Fatal("text continuation rejected the same subtype constraint")
	}
	g.AddContinuousEffect(SetSubtypes(wall.ID(), SubtypeCreature, core.Indefinite, "Elf"))
	if auraHostIsLegal(aura, wall, g) {
		t.Fatal("attachment ignored a current subtype change")
	}
}

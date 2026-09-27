package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestAttachedAurasDynamicStaticCount(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	value := MultiplyValues(Fixed(2), CountPermanents(AttachedAuras(This)))
	card := NewCreature("Synthetic wombat", "", 0, 1, WithAbility(Static(While(Controls(IsCreature), ModifyPT(This, value, value)))))
	host := g.PutOnBattlefield(card, a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 2), b.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Aura", ""), b.PlayerID())
	equipment := g.PutOnBattlefield(NewArtifact("Equipment", "", WithSubTypes("Equipment")), a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	g.Attach(equipment.ID(), host.ID())
	if power, toughness := g.FindPermanent(host.ID()).CurrentPower(g), g.FindPermanent(host.ID()).CurrentToughness(g); power != 2 || toughness != 3 {
		t.Fatalf("one Aura and one Equipment give %d/%d", power, toughness)
	}
	g.Attach(aura.ID(), other.ID())
	if got := g.FindPermanent(host.ID()).CurrentPower(g); got != 0 {
		t.Fatalf("power after reattachment = %d", got)
	}
	g.Attach(aura.ID(), host.ID())
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(aura.ID()))
	if got := clone.FindPermanent(host.ID()).CurrentPower(clone); got != 0 {
		t.Fatalf("clone power = %d", got)
	}
	if got := g.FindPermanent(host.ID()).CurrentPower(g); got != 2 {
		t.Fatalf("original power = %d", got)
	}
}

func TestAttachedAurasIncarnationPhasingAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := g.PutOnBattlefield(NewCreature("Host", "", 2, 2), a.PlayerID())
	aura := g.PutOnBattlefield(NewAura("Aura", ""), b.PlayerID())
	g.Attach(aura.ID(), host.ID())
	count := CountPermanents(AttachedAuras(This))
	ctx := &EffectContext{Game: g, SourceID: host.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: host.incarnationID}
	check := func(want int) {
		t.Helper()
		if got := count.resolveValue(ctx); got != want {
			t.Fatalf("attached Aura count = %d, want %d", got, want)
		}
	}
	check(1)
	g.PhaseOut(aura.ID())
	check(0)
	g.PhaseIn([]uuid.UUID{aura.ID()})
	check(1)
	phased := g.PhaseOut(host.ID())
	check(0)
	g.PhaseIn(phased)
	check(1)
	g.RemoveFromBattlefield(host)
	g.PutOnBattlefield(host.Card, a.PlayerID())
	check(0)
	if got := count.resolveValue(&EffectContext{Game: g, SourceID: host.ID(), Controller: a.PlayerID()}); got != 0 {
		t.Fatalf("returned host inherited %d Auras", got)
	}
	valid := NewCreature("Valid", "", 1, 1, WithAbility(Static(While(Controls(IsCreature), ModifyPT(This, CountPermanents(AttachedAuras(This)), Fixed(0))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	invalid := NewInstant("Invalid", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(AttachedAuras(nil))))))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("accepted nil attachment host")
	}
	target := DeclarePermanentTarget("host", IsCreature)
	undeclared := NewInstant("Undeclared", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(AttachedAuras(target))))))
	if err := ValidateCardDefinition(undeclared.Name(), undeclared); err == nil {
		t.Fatal("accepted undeclared attachment host target")
	}
}

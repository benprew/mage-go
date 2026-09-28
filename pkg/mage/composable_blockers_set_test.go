package mage

import "testing"

func TestBlockersOfCurrentCombatIncarnations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 3, 3), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 3, 3), a.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 2), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 2), b.PlayerID())
	unrelated := g.PutOnBattlefield(NewCreature("Unrelated", "", 2, 2), b.PlayerID())
	g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
	g.combat.AddAttackerIncarnation(other.ID(), other.incarnationID, b.PlayerID())
	g.combat.AddBlockerIncarnation(first.ID(), first.incarnationID, attacker.ID())
	g.combat.AddBlockerIncarnation(second.ID(), second.incarnationID, attacker.ID())
	g.combat.AddBlockerIncarnation(unrelated.ID(), unrelated.incarnationID, other.ID())
	ctx := &EffectContext{Game: g, SourceID: attacker.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: attacker.incarnationID}
	count := CountPermanents(BlockersOf(This))
	if got := count.resolveValue(ctx); got != 2 {
		t.Fatalf("blocker count = %d, want 2", got)
	}
	clone := g.Clone()
	clone.RemoveFromCombat(first.ID())
	if got := count.resolveValue(&EffectContext{Game: clone, SourceID: attacker.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: attacker.incarnationID}); got != 1 {
		t.Fatalf("clone blocker count = %d, want 1", got)
	}
	clone.RemoveFromCombat(second.ID())
	if got := count.resolveValue(&EffectContext{Game: clone, SourceID: attacker.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: attacker.incarnationID}); got != 0 {
		t.Fatalf("blocked attacker with no blockers has %d blockers", got)
	}
	if got := count.resolveValue(ctx); got != 2 {
		t.Fatalf("original blocker count = %d, want 2", got)
	}
	phaseClone := g.Clone()
	phaseClone.PhaseOut(first.ID())
	if got := count.resolveValue(&EffectContext{Game: phaseClone, SourceID: attacker.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: attacker.incarnationID}); got != 1 {
		t.Fatalf("phased blocker count = %d, want 1", got)
	}
	phaseClone.PhaseOut(attacker.ID())
	if got := count.resolveValue(&EffectContext{Game: phaseClone, SourceID: attacker.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: attacker.incarnationID}); got != 0 {
		t.Fatalf("phased attacker count = %d, want 0", got)
	}
	g.RemoveFromBattlefield(second)
	g.PutOnBattlefield(second.Card, b.PlayerID())
	if got := count.resolveValue(ctx); got != 1 {
		t.Fatalf("returned blocker count = %d, want 1", got)
	}
	g.RemoveFromBattlefield(attacker)
	g.PutOnBattlefield(attacker.Card, a.PlayerID())
	if got := count.resolveValue(ctx); got != 0 {
		t.Fatalf("returned attacker count = %d, want 0", got)
	}
}

func TestBlockersOfDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("attacker", IsCreature)
	good := NewInstant("Good", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(BlockersOf(target))), WithTargets(target))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	if info := InspectDefinition(BlockersOf(target)); info.Role != "permanent-set" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
	for _, subject := range []PermanentRef{nil, target, From(BindPermanent("missing"))} {
		bad := NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(BlockersOf(subject))))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatalf("invalid blocker set accepted: %+v", InspectDefinition(BlockersOf(subject)))
		}
	}
}

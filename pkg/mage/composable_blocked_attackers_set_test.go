package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestBlockedAttackersOfCurrentCombatIncarnations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 2), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 2), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 2), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 2, 2), b.PlayerID())
	unrelated := g.PutOnBattlefield(NewCreature("Unrelated", "", 2, 2), b.PlayerID())
	for _, attacker := range []*Permanent{first, second, other} {
		g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
	}
	g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, first.ID())
	g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, second.ID())
	g.combat.AddBlockerIncarnation(unrelated.ID(), unrelated.incarnationID, other.ID())
	ctx := &EffectContext{Game: g, SourceID: blocker.ID(), Controller: b.PlayerID(), sourceCaptured: true, sourceIncarnation: blocker.incarnationID}
	count := CountPermanents(BlockedAttackersOf(This))
	if got := count.resolveValue(ctx); got != 2 {
		t.Fatalf("blocked attackers = %d, want 2", got)
	}
	clone := g.Clone()
	clone.RemoveFromCombat(first.ID())
	cloneCtx := &EffectContext{Game: clone, SourceID: blocker.ID(), Controller: b.PlayerID(), sourceCaptured: true, sourceIncarnation: blocker.incarnationID}
	if got := count.resolveValue(cloneCtx); got != 1 {
		t.Fatalf("clone count = %d, want 1", got)
	}
	if got := count.resolveValue(ctx); got != 2 {
		t.Fatalf("original count after clone edit = %d", got)
	}
	phase := g.Clone()
	phase.PhaseOut(second.ID())
	phaseCtx := &EffectContext{Game: phase, SourceID: blocker.ID(), Controller: b.PlayerID(), sourceCaptured: true, sourceIncarnation: blocker.incarnationID}
	if got := count.resolveValue(phaseCtx); got != 1 {
		t.Fatalf("phased attacker count = %d", got)
	}
	phase.PhaseOut(blocker.ID())
	if got := count.resolveValue(phaseCtx); got != 0 {
		t.Fatalf("phased blocker count = %d", got)
	}
	g.RemoveFromBattlefield(first)
	g.PutOnBattlefield(first.Card, a.PlayerID())
	if got := count.resolveValue(ctx); got != 1 {
		t.Fatalf("returned attacker count = %d", got)
	}
	g.RemoveFromBattlefield(blocker)
	g.PutOnBattlefield(blocker.Card, b.PlayerID())
	if got := count.resolveValue(ctx); got != 0 {
		t.Fatalf("returned blocker count = %d", got)
	}
}

func TestBlockedAttackersOfDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("blocker", IsCreature)
	valid := NewInstant("Valid", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(BlockedAttackersOf(target))), WithTargets(target))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	if info := InspectDefinition(BlockedAttackersOf(target)); info.Role != "permanent-set" || len(info.Children) != 1 {
		t.Fatalf("inspection = %+v", info)
	}
	for _, subject := range []PermanentRef{nil, target, From(BindPermanent("missing"))} {
		bad := NewInstant("Bad", "", WithAbility(Spell(GainLifeFor(You, CountPermanents(BlockedAttackersOf(subject))))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid blocked-attacker set accepted")
		}
	}
}

func TestBlockedAttackersOfTargetResolution(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 2), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 2, 2), b.PlayerID())
	g.combat.AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
	g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, attacker.ID())
	target := DeclarePermanentTarget("blocker", IsCreature)
	card := NewInstant("Count", "", Spell(GainLifeFor(You, CountPermanents(BlockedAttackersOf(target))), WithTargets(target)))
	ability := card.Abilities()[0].(*SpellAbility)
	push := func(game *Game) {
		game.pushStack(&StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{blocker.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	}
	push(g)
	clone := g.Clone()
	clone.ResolveStack()
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("clone life = %d, want 21", got)
	}
	if got := a.Life(); got != 20 {
		t.Fatalf("original life = %d, want 20", got)
	}
	g.RemoveFromBattlefield(blocker)
	g.PutOnBattlefield(blocker.Card, b.PlayerID())
	g.ResolveStack()
	if got := a.Life(); got != 20 {
		t.Fatalf("returned target life = %d, want 20", got)
	}
}

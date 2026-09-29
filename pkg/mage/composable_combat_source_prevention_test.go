package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCombatSourcePreventionScopeAndLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	blocked := g.PutOnBattlefield(NewCreature("Blocked", "", 3, 3), a.PlayerID())
	unrelated := g.PutOnBattlefield(NewCreature("Unrelated", "", 3, 3), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 3, 3), b.PlayerID())
	g.combat.AddAttackerIncarnation(blocked.ID(), blocked.incarnationID, b.PlayerID())
	g.combat.AddAttackerIncarnation(unrelated.ID(), unrelated.incarnationID, b.PlayerID())
	g.combat.AddBlockerIncarnation(blocker.ID(), blocker.incarnationID, blocked.ID())
	ctx := &EffectContext{Game: g, SourceID: blocker.ID(), Controller: b.PlayerID(), sourceCaptured: true, sourceIncarnation: blocker.incarnationID}
	if err := PreventCombatDamageBy(This).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if err := PreventCombatDamageByEach(BlockedAttackersOf(This)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.damage.SetResolvingCombatDamage(true)
	clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), 2, blocked.ID())
	clone.DealDamageToPlayer(clone.GetPlayer(a.PlayerID()), 2, blocker.ID())
	clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), 2, unrelated.ID())
	clone.DealDamageToPermanent(clone.FindPermanent(blocker.ID()), 1, blocked.ID())
	clone.DealDamageToPermanent(clone.FindPermanent(blocker.ID()), 1, unrelated.ID())
	clone.damage.SetResolvingCombatDamage(false)
	clone.DealDamageToPlayer(clone.GetPlayer(b.PlayerID()), 1, blocked.ID())
	if got := clone.GetPlayer(b.PlayerID()).Life(); got != 17 {
		t.Fatalf("defender life = %d, want 17", got)
	}
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 20 {
		t.Fatalf("attacker life = %d, want 20", got)
	}
	if got := clone.FindPermanent(blocker.ID()).Damage; got != 1 {
		t.Fatalf("blocker damage = %d, want 1", got)
	}
	if b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.effects.RemoveEndOfCombat()
	g.damage.SetResolvingCombatDamage(true)
	g.DealDamageToPlayer(b, 1, blocked.ID())
	if b.Life() != 19 {
		t.Fatal("prevention survived combat")
	}
}

func TestCombatSourcePreventionIncarnation(t *testing.T) {
	for _, change := range []string{"return", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			p := g.PutOnBattlefield(NewCreature("Source", "", 2, 2), a.PlayerID())
			if err := PreventCombatDamageBy(This).Apply(&EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, a.PlayerID())
			case "phase":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			}
			g.damage.SetResolvingCombatDamage(true)
			g.DealDamageToPlayer(b, 1, p.ID())
			want := 20
			if change == "return" {
				want = 19
			}
			if b.Life() != want {
				t.Fatalf("life = %d, want %d", b.Life(), want)
			}
		})
	}
}

func TestCombatSourcePreventionDeclaration(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	valid := NewInstant("Valid", "", Spell(Sequence(PreventCombatDamageBy(target), PreventCombatDamageByEach(BlockedAttackersOf(target))), WithTargets(target)))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{PreventCombatDamageBy(nil), PreventCombatDamageByEach(nil), PreventCombatDamageBy(target), PreventCombatDamageByEach(BlockedAttackersOf(target))} {
		bad := NewInstant("Bad", "", Spell(effect))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("invalid prevention accepted")
		}
	}
	if info := InspectDefinition(PreventCombatDamageBy(target)); info.Role != "install-combat-source-prevention" {
		t.Fatalf("inspection = %+v", info)
	}
}

func TestCombatSourcePreventionCapturedDeparture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 2), a.PlayerID())
	source := g.CaptureDamageSource(p.ID())
	if err := PreventCombatDamageBy(This).Apply(&EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(p.Card, a.PlayerID())
	g.damage.SetResolvingCombatDamage(true)
	g.DealDamageToPlayerFromSource(b, 2, source)
	if b.Life() != 20 {
		t.Fatal("captured source damage was not prevented")
	}
	g.DealDamageToPlayer(b, 1, p.ID())
	if b.Life() != 19 {
		t.Fatal("new incarnation inherited prevention")
	}
}

func TestCombatSourcePreventionTargetReturn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Target", "", 2, 2), a.PlayerID())
	target := DeclarePermanentTarget("creature", IsCreature)
	card := NewInstant("Prevention", "", Spell(PreventCombatDamageBy(target), WithTargets(target)))
	ability := card.Abilities()[0].(*SpellAbility)
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.RemoveFromBattlefield(p)
	g.PutOnBattlefield(p.Card, a.PlayerID())
	g.ResolveStack()
	g.damage.SetResolvingCombatDamage(true)
	g.DealDamageToPlayer(b, 1, p.ID())
	if b.Life() != 19 {
		t.Fatal("returned target received prevention")
	}
}

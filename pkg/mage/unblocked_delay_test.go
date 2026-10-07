package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestUnblockedDelayLifecycle(t *testing.T) {
	for _, change := range []string{"none", "control", "phase", "phase-cycle", "return", "source-return", "source-phase", "source-abilities"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			card := NewCreature("Watched", "", 3, 8)
			card.SetOwner(a.PlayerID())
			watched := g.PutOnBattlefield(card, a.PlayerID())
			saved := BindPermanent("watched")
			target := DeclarePermanentTarget("creature", IsCreature)
			g.resolution.targetSpecs = []Target{target.targetDeclaration}
			effect := Sequence(Snapshot(target, saved), WhenAttacksUnblockedThisTurn(From(saved), GainLifeFor(You, PowerOfWithLKI(From(saved)))))
			if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{watched.ID()}); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 20 {
				t.Fatal("trigger ran during installation")
			}
			switch change {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: watched.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "phase":
				g.PhaseOut(watched.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(watched.ID()))
			case "return":
				g.BouncePermanentToHand(watched)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			case "source-phase":
				g.PhaseOut(source.ID())
			case "source-abilities":
				g.AddContinuousEffect(RemoveAllAbilities(source.ID(), EndOfTurn))
				g.ApplyContinuousEffects()
			case "source-return":
				g.BouncePermanentToHand(source)
				a.RemoveFromHand(source.ID())
				g.PutOnBattlefield(source.Card, b.PlayerID())
			}
			clone := g.Clone()
			for _, game := range []*Game{g, clone} {
				for range 2 {
					game.combat.Reset()
					game.combat.AddAttacker(watched.ID(), b.PlayerID())
					game.doDeclareBlockers()
					game.PutTriggersOnStack()
					game.ResolveStack()
				}
				want := 26
				if change == "phase" || change == "return" {
					want = 20
				}
				if game.GetPlayer(a.PlayerID()).Life() != want || game.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatalf("life A=%d B=%d want A=%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life(), want)
				}
			}
			g.RunStepWithPriority(Cleanup)
			if len(g.DelayedTriggers()) != 0 || len(clone.DelayedTriggers()) != 1 {
				t.Fatal("expiry changed clone")
			}
		})
	}
}

func TestUnblockedDelayPendingAndLegality(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watched := g.PutOnBattlefield(NewCreature("Watched", "", 4, 8), a.PlayerID())
	blocker := g.PutOnBattlefield(NewCreature("Blocker", "", 1, 8), b.PlayerID())
	saved := BindPermanent("watched")
	effect := Sequence(Snapshot(This, saved), WhenAttacksUnblockedThisTurn(From(saved), GainLifeFor(You, PowerOfWithLKI(From(saved)))))
	if err := ApplyEffect(g, effect, watched.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.combat.AddAttacker(watched.ID(), b.PlayerID())
	g.combat.AddBlocker(blocker.ID(), watched.ID())
	g.combat.Groups[0].BlockerIDs = nil
	g.doDeclareBlockers()
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("departed blocker made attacker unblocked")
	}
	g.combat.Reset()
	g.combat.AddAttacker(watched.ID(), b.PlayerID())
	g.doDeclareBlockers()
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("unblocked trigger not queued")
	}
	clone := g.Clone()
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.BouncePermanentToHand(g.FindPermanent(watched.ID()))
	a.RemoveFromHand(watched.ID())
	g.PutOnBattlefield(watched.Card, b.PlayerID()).AddCounter(P1P1, 9)
	g.RunStepWithPriority(Cleanup)
	g.ResolveStack()
	clone.ResolveStack()
	if a.Life() != 24 || b.Life() != 24 || clone.GetPlayer(a.PlayerID()).Life() != 24 {
		t.Fatal("pending capture, LKI, copy or cleanup changed")
	}
}

func TestUnblockedDelayDeclarations(t *testing.T) {
	saved := BindPermanent("unbound")
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: NewCreature("Valid", "", 1, 1, WithAbility(Activate(Costs(), WhenAttacksUnblockedThisTurn(This, GainLifeFor(You, Fixed(1))))))},
		{Name: "nil subject", Card: NewInstant("Bad", "", Spell(WhenAttacksUnblockedThisTurn(nil, GainLifeFor(You, Fixed(1))))), WantErr: true, WantErrSubstr: "delayed"},
		{Name: "nil body", Card: NewCreature("Bad", "", 1, 1, WithAbility(Activate(Costs(), WhenAttacksUnblockedThisTurn(This, nil)))), WantErr: true, WantErrSubstr: "delayed"},
		{Name: "unbound", Card: NewInstant("Bad", "", Spell(WhenAttacksUnblockedThisTurn(From(saved), GainLifeFor(You, Fixed(1))))), WantErr: true, WantErrSubstr: "producer"},
	})
}

func TestUnblockedDelayInstallationAndStaleEvents(t *testing.T) {
	for _, change := range []string{"absent-install", "phased-install", "absent-event", "phased-event", "stale-event", "noncreature-event"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Watched", "", 2, 8)
			p := g.PutOnBattlefield(card, a.PlayerID())
			incarnation := p.incarnationID
			if change == "absent-install" {
				g.BouncePermanentToHand(p)
			}
			if change == "phased-install" {
				g.PhaseOut(p.ID())
			}
			if change == "stale-event" {
				g.BouncePermanentToHand(p)
				g.PutOnBattlefield(card, a.PlayerID())
			}
			if err := ApplyEffect(g, WhenAttacksUnblockedThisTurn(This, GainLifeFor(You, Fixed(2))), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if change == "absent-event" {
				g.BouncePermanentToHand(p)
			}
			if change == "phased-event" {
				g.PhaseOut(p.ID())
			}
			if change == "noncreature-event" {
				g.MutablePermanent(p.ID()).RevokeBaseAttr(CardTypeAttr(TypeCreature))
			}
			g.FireEvent(GameEvent{Type: EvtAttacksUnblocked, SourceID: p.ID(), SourceIncarnation: incarnation, PlayerID: a.PlayerID(), TargetID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 0 {
				t.Fatal("invalid event matched delayed subscription")
			}
		})
	}
}

func TestUnblockedDelayTimingAndAPNAP(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watched := g.PutOnBattlefield(NewCreature("Watched", "", 2, 8), a.PlayerID())
	g.combat.AddAttacker(watched.ID(), b.PlayerID())
	g.doDeclareBlockers()
	for _, controller := range []uuid.UUID{b.PlayerID(), a.PlayerID()} {
		if err := ApplyEffect(g, WhenAttacksUnblockedThisTurn(This, GainLifeFor(You, Fixed(2))), watched.ID(), controller, nil); err != nil {
			t.Fatal(err)
		}
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("subscription matched earlier event")
	}
	g.combat.Reset()
	g.combat.AddAttacker(watched.ID(), b.PlayerID())
	g.doDeclareBlockers()
	g.PutTriggersOnStack()
	if g.StackSize() != 2 || g.StackPeek().Controller != b.PlayerID() {
		t.Fatal("APNAP placement changed")
	}
	clone := g.Clone()
	g.RunStepWithPriority(Cleanup)
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 || len(g.DelayedTriggers()) != 0 {
		t.Fatal("cleanup changed pending bodies")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 22 || len(clone.DelayedTriggers()) != 2 {
		t.Fatal("pending clone lost subscriptions")
	}
}

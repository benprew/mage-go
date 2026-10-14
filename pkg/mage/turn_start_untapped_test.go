package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTurnStartUntappedFacts(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	landCard := NewLand("Land")
	land := g.PutOnBattlefield(landCard, a.PlayerID())
	tapped := g.PutOnBattlefield(NewLand("Tapped"), a.PlayerID())
	g.TapPermanent(tapped)
	phased := g.PutOnBattlefield(NewLand("Phased"), a.PlayerID())
	g.PhaseOut(phased.ID())
	g.PutOnBattlefield(NewLand("Other"), b.PlayerID())
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 10), a.PlayerID())
	ctx := &EffectContext{Game: g, SourceID: source.ID(), Controller: a.PlayerID()}
	value := UntappedLandCountAtTurnStart(You)
	condition := PermanentStartedTurnUntapped(This)
	state := StateContext{SourceID: source.ID(), ControllerID: a.PlayerID(), SourceIncarnation: source.incarnationID}
	if value.resolveValue(ctx) != 0 || condition.checkState(g, state) {
		t.Fatal("query created history")
	}
	g.RunStepWithPriority(core.Untap)
	if value.resolveValue(ctx) != 1 || !condition.checkState(g, state) {
		t.Fatal("capture must precede phasing and untapping")
	}
	g.TapPermanent(source)
	g.TapPermanent(land)
	g.RemoveFromBattlefield(land)
	g.PutOnBattlefield(landCard, b.PlayerID())
	g.BeginActiveTurn()
	if value.resolveValue(ctx) != 1 || !condition.checkState(g, state) {
		t.Fatal("later state changed history")
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	returned := g.PutOnBattlefield(source.Card, a.PlayerID())
	if !condition.checkState(g, state) || condition.checkState(g, StateContext{SourceID: returned.ID(), ControllerID: a.PlayerID(), SourceIncarnation: returned.incarnationID}) {
		t.Fatal("saved history or returned visit")
	}
	if !condition.checkState(clone, state) {
		t.Fatal("clone lost history")
	}
	clone.PhaseOut(source.ID())
	if !condition.checkState(clone, state) {
		t.Fatal("phasing erased history")
	}
	clone.PhaseIn([]uuid.UUID{source.ID()})
	if !condition.checkState(clone, state) {
		t.Fatal("phase cycle lost history")
	}
	g.GrantExtraTurn(a.PlayerID())
	g.AdvanceToNextTurn()
	if value.resolveValue(ctx) != 0 {
		t.Fatal("uncaptured new turn read old history")
	}
	g.BeginActiveTurn()
	if value.resolveValue(ctx) != 2 || clone.UntappedLandsAtTurnStart(a.PlayerID()) != 1 {
		t.Fatal("extra turn or clone history")
	}
	if !condition.checkState(g, StateContext{SourceID: returned.ID(), ControllerID: a.PlayerID(), SourceIncarnation: returned.incarnationID}) {
		t.Fatal("new turn did not capture new visit")
	}
}

func TestTurnStartUntappedSkippedUntap(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
	tapped := g.PutOnBattlefield(NewLand("Tapped"), a.PlayerID())
	g.TapPermanent(tapped)
	g.PutOnBattlefield(NewEnchantment("Skip", "", WithAbility(Static(SkipUntapSteps()))), a.PlayerID())
	g.RunStepWithPriority(core.Untap)
	if !g.FindPermanent(tapped.ID()).Tapped {
		t.Fatal("untap replacement did not skip")
	}
	g.RunStepWithPriority(core.Upkeep)
	if g.UntappedLandsAtTurnStart(a.PlayerID()) != 1 {
		t.Fatal("skipped untap lost turn start")
	}
	g.AdvanceToNextTurn()
	g.BeginActiveTurn()
	if g.UntappedLandsAtTurnStart(a.PlayerID()) != 1 || g.UntappedLandsAtTurnStart(b.PlayerID()) != 0 {
		t.Fatal("snapshot must include both players")
	}
}

func TestTurnStartUntappedDefinition(t *testing.T) {
	for _, card := range []Card{
		NewInstant("Nil", "", WithAbility(Spell(GainLifeFor(You, UntappedLandCountAtTurnStart(nil))))),
		NewCreature("Nil", "", 1, 1, WithAbility(At(YourUpkeep, GainLifeFor(You, Fixed(1)), WithInterveningIf(PermanentStartedTurnUntapped(nil))))),
		NewCreature("Target", "", 1, 1, WithAbility(At(YourUpkeep, GainLifeFor(You, Fixed(1)), WithInterveningIf(PermanentStartedTurnUntapped(DeclarePermanentTarget("target", IsCreature)))))),
	} {
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration accepted", card.Name())
		}
	}
}

func TestTurnStartUntappedHistoricalTrigger(t *testing.T) {
	for _, change := range []string{"depart", "return", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("History", "", 1, 10, WithAbility(At(YourUpkeep, GainLifeFor(You, Fixed(2)), WithInterveningIf(PermanentStartedTurnUntapped(This)))))
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.RunStepWithPriority(core.Untap)
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			switch change {
			case "depart", "return":
				g.RemoveFromBattlefield(p)
				if change == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(p.ID())
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			g.ResolveStack()
			if a.Life() != 22 || b.Life() != 20 {
				t.Fatal("historical condition lost saved visit or controller")
			}
		})
	}
}

package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedBranchOperands(t *testing.T) {
	noSource := GainLifeFor(You, Fixed(0))
	source := GainLifeFor(You, PowerOfWithLKI(This))
	shapes := map[string]ResolvingEffect{
		"positive-value":       IfPositive(PowerOfWithLKI(This), noSource),
		"positive-body":        IfPositive(Fixed(0), source),
		"card-body":            IfCardHasType(FromCard(BindCard("card")), core.TypeCreature, source),
		"cast-body":            IfCast(ControlledCreatureSubtypeAtCast("Wolf"), source),
		"subtype-subject":      IfPermanentHasEnchantmentSubtype(This, "Aura", noSource),
		"subtype-body":         IfPermanentHasEnchantmentSubtype(AttachedPermanent, "Aura", source),
		"activation-condition": IfActivationsAtLeast(BindActivationHistory("ability"), 1, noSource),
		"activation-body":      IfActivationsAtLeast(BindActivationHistory("ability"), 0, source),
		"coin-player":          FlipCoin(ControllerOfWithLKI(This), OnLoss(noSource)),
		"coin-win":             FlipCoin(You, OnWin(source), OnLoss(noSource)),
		"coin-loss":            FlipCoin(You, OnWin(noSource), OnLoss(source)),
		"control-player":       UnlessPermanentCameUnderControlThisTurn(AttachedPermanent, ControllerOfWithLKI(This), noSource),
		"state-body":           IfState(IsYourTurn(), source),
		"player-producer":      SnapshotPlayer(ControllerOfWithLKI(This), BindPlayer("player")),
	}
	conditions := map[string]StateCondition{
		"tapped":                 SourceTappedCondition(),
		"tapped-lki":             SourceTappedStateWithLKI(true),
		"blocking":               SourceBlockingCondition(),
		"attacked":               SourceAttackedThisTurnCondition(),
		"attack-or-block-turn":   SourceAttackedOrBlockedThisTurnCondition(),
		"attack-or-block-combat": SourceAttackedOrBlockedThisCombatCondition(),
		"attack-history":         SourceAttackHistoryThisTurn(false),
		"damage-history":         SourceDamagedOpponentThisTurnCondition(),
		"counters":               SourceHasCounters(core.P1P1, 1),
		"permanent-tapped":       PermanentIsTapped(This),
		"permanent-attacking":    PermanentIsAttacking(This),
		"permanent-blocking":     PermanentIsBlocking(This),
		"permanent-filter":       PermanentMatchesFilter(This, IsCreature),
		"continuous-control":     ControlledSinceTurnStart(This),
		"owner":                  PlayerIsPermanentOwner(This),
		"last-turn-player":       PermanentAttackedDuringLastTurn(AttachedPermanent, ControllerOf(This)),
		"nested":                 AllConditions(IsYourTurn(), NotCondition(AnyCondition(PermanentIsTapped(This), Controls(IsArtifact)))),
	}
	for name, condition := range conditions {
		shapes["state-"+name] = IfState(condition, noSource)
	}
	for name, body := range shapes {
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 2, 4)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := ApplyEffect(g, AtNextModes(EachEndStep, ModeOf("branch", Sequence(body)), ModeOf("duplicate", Sequence(body))), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.PhaseOut(p.ID())
			if got := g.WaitingDelayedPermanentSources(); len(got) != 1 || !got[0].SameObject(saved) {
				t.Fatal("phasing lost conditional operand reference")
			}
			g.PhaseIn([]uuid.UUID{p.ID()})
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			a.RemoveFromHand(card.ID())
			g.PutOnBattlefield(card, b.PlayerID())
			g.ChangeObjectColor(card.ID(), core.Red)
			for range 3 {
				if got := g.WaitingDelayedPermanentSources(); len(got) != 1 || !got[0].SameObject(saved) || got[0].HasColor(core.Red) {
					t.Fatal("conditional operand followed returned object")
				}
			}
			clone := g.Clone()
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("conditional waiting lifetime is shared")
			}
		})
	}
}

func TestWaitingDelayedBranchPurity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 4), a.PlayerID())
	reads := 0
	body := IfPositive(waitingMultiAmount{&reads}, FlipCoin(You, OnWin(GainLifeFor(You, PowerOfWithLKI(This)))))
	g.SetCoinFlipResults([]bool{true, false})
	if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || a.Life() != 20 {
			t.Fatal("query read a value, flipped a coin or ran a branch")
		}
	}
	if !g.FlipCoin(a.PlayerID()) {
		t.Fatal("query consumed the queued coin outcome")
	}
	g.triggers.SetDelayed(nil)
	callback := StateConditionFunc("opaque", []StateDependency{StateBattlefield}, func(StateReader, StateContext) bool {
		reads++
		return true
	})
	for _, body := range []ResolvingEffect{
		IfState(callback, GainLifeFor(You, Fixed(0))),
		IfState(PermanentIsTapped(AttachedPermanent), GainLifeFor(You, Fixed(0))),
		IfPositive(Fixed(1), AtNext(EachEndStep, Destroy(This))),
		FlipCoin(You, OnLoss(AtNext(EachEndStep, Destroy(This)))),
		IfState(Controls(IsCreature), GainLifeFor(You, Fixed(0))),
		IfCardLacksType(FromCard(BindCard("card")), core.TypeCreature, GainLifeFor(You, Fixed(0))),
		IfCast(ControlledCreatureSubtypeAtCast("Wolf"), GainLifeFor(You, Fixed(0))),
	} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 0 {
		t.Fatal("query inferred opaque, implicit or future reads")
	}
	for _, body := range []ResolvingEffect{
		IfPositive(nil, Destroy(This)),
		IfState(nil, Destroy(This)),
		FlipCoin(You, OnWin(nil)),
		IfPermanentHasEnchantmentSubtype(This, "Wolf", Destroy(This)),
		IfActivationsAtLeast(nil, 1, Destroy(This)),
	} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))) == nil {
			t.Fatal("invalid conditional declaration accepted")
		}
	}
}

func TestWaitingDelayedBranchIntervening(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	if err := ApplyEffect(g, AtNext(EachEndStep, GainLifeFor(You, Fixed(2)), WithInterveningIf(SourceTappedCondition())), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("waiting false intervening condition lost its explicit source")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	if len(g.WaitingDelayedPermanentSources()) != 1 || len(g.triggers.Pending()) != 0 {
		t.Fatal("false intervening condition consumed its wait")
	}
	g.TapPermanent(p)
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.UntapPermanent(p)
	g.ResolveStack()
	if len(g.WaitingDelayedPermanentSources()) != 0 || a.Life() != 20 {
		t.Fatal("intervening condition was not rechecked at resolution")
	}
}

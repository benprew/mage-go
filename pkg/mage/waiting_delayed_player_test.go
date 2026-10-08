package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedPlayerOperands(t *testing.T) {
	players := OpponentsOf(ControllerOfWithLKI(This))
	history := PlayersDealtDamageBy(This, EachPlayer)
	member := BindPlayer("member")
	shapes := map[string]ResolvingEffect{
		"draw-set":       DrawEach(players, Fixed(0)),
		"draw-value":     DrawEach(EachPlayer, PowerOfWithLKI(This)),
		"discard":        DiscardEach(players),
		"gain-set":       GainLifeEach(players, Fixed(0)),
		"gain-value":     GainLifeEach(EachPlayer, PowerOfWithLKI(This)),
		"lose-set":       LoseLifeEach(players, Fixed(0)),
		"lose-value":     LoseLifeEach(EachPlayer, PowerOfWithLKI(This)),
		"recycle":        ShuffleHandsAndGraveyardsIntoLibraries(players),
		"recycle-draw":   ShuffleHandsIntoLibrariesAndDraw(players),
		"recycle-single": ShuffleGraveyardFor(OwnerOf(This)),
		"history":        DrawEach(PlayersDealtDamageBy(This, players), Fixed(0)),
		"random":         ChooseRandomPlayer(history, member),
		"damage":         DealDamageEachPlayer(DamageSourceOf(AttachedPermanent), players, member, Fixed(0)),
		"mixed-damage":   DealDamageEach(DamageSourceOf(AttachedPermanent), DamageRecipients(history, MatchingPermanents(IsCreature)), Fixed(0)),
		"permanent-set":  TapEach(PermanentsControlledByPlayers(history, IsCreature)),
	}
	for name, body := range shapes {
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 2, 4)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			if err := ApplyEffect(g, AtNext(EachEndStep, Sequence(body)), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.PhaseOut(p.ID())
			if got := g.WaitingDelayedPermanentSources(); len(got) != 1 || !got[0].SameObject(saved) {
				t.Fatal("phasing lost player operand source")
			}
			g.PhaseIn([]uuid.UUID{p.ID()})
			g.BouncePermanentToHand(g.FindPermanent(p.ID()))
			a.RemoveFromHand(card.ID())
			returned := g.PutOnBattlefield(card, b.PlayerID())
			g.ChangeObjectColor(returned.ID(), core.Red)
			for range 3 {
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(saved) || got[0].HasColor(core.Red) {
					t.Fatal("player operand followed returned source")
				}
			}
			clone := g.Clone()
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("clone shares waiting lifetime")
			}
		})
	}
}

func TestWaitingDelayedPlayerPurity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 2, 4), a.PlayerID())
	reads := 0
	body := DrawEach(PlayersDealtDamageBy(This, noDamagePlayers{}), waitingMultiAmount{&reads})
	if err := ApplyEffect(g, AtNextModes(EachEndStep, ModeOf("first", body), ModeOf("second", body)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 {
			t.Fatal("query evaluated empty set or value")
		}
	}
	g.triggers.SetDelayed(nil)
	for _, body := range []ResolvingEffect{
		DrawEach(EachPlayer, Fixed(0)),
		DrawEach(OpponentsOf(You), Fixed(0)),
		DrawEach(OpponentsOf(ControllerOf(AttachedPermanent)), Fixed(0)),
		AtNext(EachEndStep, DrawEach(EachPlayer, PowerOfWithLKI(This))),
	} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 {
		t.Fatal("query inferred implicit or future reference")
	}
	for _, body := range []ResolvingEffect{DrawEach(nil, Fixed(1)), GainLifeEach(EachPlayer, nil), ShuffleHandsAndGraveyardsIntoLibraries(nil), ChooseRandomPlayer(EachPlayer, nil)} {
		if ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body))))) == nil {
			t.Fatal("invalid player declaration accepted")
		}
	}
}

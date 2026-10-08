package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestWaitingDelayedReturnController(t *testing.T) {
	for name, player := range map[string]PlayerRef{
		"controller": ControllerOf(This), "controller-lki": ControllerOfWithLKI(This), "owner": OwnerOf(This),
	} {
		t.Run(name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Source", "", 1, 2)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			saved := g.CaptureDamageSource(p.ID())
			body := ReturnCardToBattlefield(FromCard(BindCard("unavailable")), player)
			if err := ApplyEffect(g, AtNext(EachEndStep, Sequence(body)), p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check := func() {
				t.Helper()
				refs := g.WaitingDelayedPermanentSources()
				if len(refs) != 1 || !refs[0].SameObject(saved) {
					t.Fatal("lost exact controller source", refs)
				}
			}
			check()
			g.PhaseOut(p.ID())
			check()
			g.PhaseIn([]uuid.UUID{p.ID()})
			check()
			g.BouncePermanentToHand(p)
			g.zones.ClearLKI()
			check()
			a.RemoveFromHand(card.ID())
			returned := g.PutOnBattlefield(card, b.PlayerID())
			g.ChangeObjectColor(returned.ID(), core.Red)
			check()
			if g.WaitingDelayedPermanentSources()[0].HasColor(core.Red) {
				t.Fatal("query followed returned source")
			}
			clone := g.Clone()
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
			if len(clone.WaitingDelayedPermanentSources()) != 0 || len(g.WaitingDelayedPermanentSources()) != 1 {
				t.Fatal("clone shares waiting lifetime")
			}
		})
	}
}

type waitingReturnPlayer struct {
	PlayerRef
	reads *int
}

func (p waitingReturnPlayer) resolvePlayer(ctx *EffectContext) (uuid.UUID, bool) {
	*p.reads++
	return p.PlayerRef.resolvePlayer(ctx)
}

func TestWaitingDelayedReturnBoundaries(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Source", "", 1, 2), a.PlayerID())
	reads := 0
	card := FromCard(BindCard("unavailable"))
	for _, body := range []ResolvingEffect{
		ReturnCardToBattlefield(card, You),
		ReturnCardToBattlefield(card, ControllerOfWithLKI(AttachedPermanent)),
		ReturnCardToBattlefield(card, waitingReturnPlayer{ControllerOfWithLKI(This), &reads}),
		AtNext(EachEndStep, ReturnCardToBattlefield(card, ControllerOfWithLKI(This))),
	} {
		if err := ApplyEffect(g, AtNext(EachEndStep, body), p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.WaitingDelayedPermanentSources()) != 0 || reads != 0 {
		t.Fatal("query evaluated opaque player or inferred host or future source")
	}
	body := ReturnCardToBattlefield(card, ControllerOfWithLKI(This))
	if err := ApplyEffect(g, AtNextModes(EachEndStep, ModeOf("normal", body), ModeOf("tapped", ReturnCardToBattlefield(card, ControllerOfWithLKI(This), EnterTapped()))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if len(g.WaitingDelayedPermanentSources()) != 1 || reads != 0 || g.FindPermanent(p.ID()) == nil {
			t.Fatal("modal query lost declaration or ran body")
		}
	}
	target := DeclareGraveyardCardTarget("card", GraveyardCardFilter(IsCreatureCard))
	for _, body := range []ResolvingEffect{ReturnCardToBattlefield(nil, You), ReturnCardToBattlefield(target, nil), ReturnCardToBattlefield(target, FromPlayer(BindPlayer("missing")))} {
		definition := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), AtNext(EachEndStep, body), WithTargets(target))))
		if ValidateCardDefinition(definition.Name(), definition) == nil {
			t.Fatal("invalid return declaration accepted")
		}
	}
}

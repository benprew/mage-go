package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestControllerCastOptionalDrawWaiting(t *testing.T) {
	a := &optionalDrawPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	saved := g.CaptureDamageSource(source.ID())
	if err := ApplyEffect(g, AtNext(EachEndStep, MayDrawCard(ControllerOfWithLKI(This))), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	check := func() {
		t.Helper()
		got := g.WaitingDelayedPermanentSources()
		if len(got) != 1 || !got[0].SameObject(saved) || a.offers != 0 {
			t.Fatal("waiting query lost identity or made a choice")
		}
	}
	check()
	group := g.PhaseOut(source.ID())
	check()
	g.PhaseIn(group)
	g.BouncePermanentToHand(source)
	check()
	g.PutOnBattlefield(source.Card, a.PlayerID())
	check()
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	if len(g.WaitingDelayedPermanentSources()) != 0 || len(clone.WaitingDelayedPermanentSources()) != 1 {
		t.Fatal("waiting lifetime is shared")
	}
}

func TestControllerCastOptionalDraw(t *testing.T) {
	for _, scenario := range []string{"accept", "decline", "replacement", "empty", "absent"} {
		t.Run(scenario, func(t *testing.T) {
			a := &optionalDrawPlayer{BasePlayer: NewBasePlayer("A"), accept: scenario != "decline"}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			if scenario != "empty" {
				a.AddToLibrary(NewLand("Card"))
			}
			if scenario == "replacement" {
				if err := ApplyEffect(g, ReplaceNextDrawThisTurn(You, GainLifeFor(You, Fixed(3))), a.PlayerID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			player := PlayerRef(You)
			if scenario == "absent" {
				player = FromPlayer(BindPlayer("missing"))
			}
			if err := ApplyEffect(g, MayDrawCard(player), a.PlayerID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			wantOffers, wantCards := 1, 0
			if scenario == "accept" {
				wantCards = 1
			}
			if scenario == "absent" {
				wantOffers = 0
			}
			if a.offers != wantOffers || len(a.Hand()) != wantCards || len(b.Hand()) != 0 {
				t.Fatal("wrong optional draw choice or recipient")
			}
			if scenario == "replacement" && a.Life() != 23 {
				t.Fatal("draw replacement did not apply")
			}
			if a.DrewFromEmpty() != (scenario == "empty") {
				t.Fatal("wrong empty-library draw failure")
			}
		})
	}
	for _, player := range []PlayerRef{nil, FromPlayer(BindPlayer("unbound"))} {
		card := NewInstant("Invalid", "", WithAbility(NewAction(ActionSpell, WithEffects(MayDrawCard(player)))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid optional draw declaration accepted")
		}
	}
}

type optionalDrawPlayer struct {
	*BasePlayer
	accept bool
	offers int
}

func (p *optionalDrawPlayer) ChooseMayAbility(string) bool { p.offers++; return p.accept }

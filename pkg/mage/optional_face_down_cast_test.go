package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type optionalFaceDownPlayer struct {
	*BasePlayer
	accept  bool
	offers  int
	onOffer func()
}

func (p *optionalFaceDownPlayer) ChooseMayAbility(string) bool {
	p.offers++
	if p.onOffer != nil {
		p.onOffer()
	}
	return p.accept
}

func TestOptionalFaceDownCast(t *testing.T) {
	for _, scenario := range []string{"accept", "decline", "foreign", "stale", "returned-during-choice", "tax", "paid-tax", "declined-tax", "prohibited", "prohibited-during-choice", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			chooser := &optionalFaceDownPlayer{BasePlayer: a, accept: scenario != "decline" && scenario != "declined-tax"}
			g.SetPlayerAt(0, chooser)
			card := NewCreature("Private", "{U}", 5, 5)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			binding := BindCard("chosen")
			g.resolution.cardBindings = map[*CardBinding]capturedCard{binding: {id: card.ID(), visit: g.handVisit(card.ID()), zone: core.ZoneHand}}
			if scenario == "stale" || scenario == "foreign" {
				a.RemoveFromHand(card.ID())
				if scenario == "stale" {
					a.AddToHand(card)
				} else {
					card.SetOwner(b.PlayerID())
					b.AddToHand(card)
				}
			}
			if scenario == "returned-during-choice" {
				chooser.onOffer = func() { a.RemoveFromHand(card.ID()); a.AddToHand(card) }
			}
			prohibit := func() {
				g.PutOnBattlefield(NewArtifact("Rule", "", WithAbility(Static(PreventSpellCasts(NewCardFilter("creature", func(c Card) bool { return c.HasType(core.TypeCreature) }))))), b.PlayerID())
			}
			if scenario == "prohibited" {
				prohibit()
			}
			if scenario == "prohibited-during-choice" {
				chooser.onOffer = prohibit
			}
			if scenario == "tax" || scenario == "paid-tax" || scenario == "declined-tax" {
				g.PutOnBattlefield(NewArtifact("Tax", "", WithAbility(StaticAbility(IncreaseSpellCostStatic(func(*Game, uuid.UUID, Card, uuid.UUID) bool { return true }, core.ManaCost{Generic: 1}, false)))), b.PlayerID())
			}
			a.ManaPool().Add(core.Colorless, 1)
			if scenario == "tax" {
				a.ManaPool().Clear()
			}
			effect := MayCastHandCardFaceDownWithoutPaying(You, FromCard(binding))
			if scenario == "clone" {
				cloned := g.Clone()
				cloned.SetPlayerAt(0, &optionalFaceDownPlayer{BasePlayer: cloned.GetPlayer(a.PlayerID()).(*SearchPlayer).BasePlayer, accept: true})
				if err := effect.Apply(&EffectContext{Game: cloned, Controller: a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				if cloned.StackSize() != 1 || g.StackSize() != 0 || len(a.Hand()) != 1 || chooser.offers != 0 {
					t.Fatal("clone changed original")
				}
			}
			if err := Sequence(effect, GainLifeFor(You, Fixed(1))).Apply(&EffectContext{Game: g, Controller: a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			cast := scenario == "accept" || scenario == "paid-tax" || scenario == "clone"
			if (g.StackSize() == 1) != cast || a.Life() != 21 {
				t.Fatalf("stack=%d life=%d", g.StackSize(), a.Life())
			}
			offers := 1
			if scenario == "stale" || scenario == "foreign" || scenario == "tax" || scenario == "prohibited" {
				offers = 0
			}
			if chooser.offers != offers {
				t.Fatalf("offers=%d want=%d", chooser.offers, offers)
			}
			wantMana := 1
			if scenario == "tax" || scenario == "paid-tax" {
				wantMana = 0
			}
			if a.ManaPool().TotalMana() != wantMana {
				t.Fatal("incorrect payment")
			}
			if len(g.PublicReveals()) != 0 {
				t.Fatal("private identity revealed")
			}
			if cast {
				if !g.StackPeek().IsFaceDown() {
					t.Fatal("cast face up")
				}
				g.ResolveStack()
				if p := g.FindPermanent(card.ID()); p == nil || !p.FaceDown || p.CurrentPower(g) != 2 {
					t.Fatal("incorrect entry")
				}
			}
		})
	}
}

func TestOptionalFaceDownCastDefinition(t *testing.T) {
	for _, effect := range []ResolvingEffect{MayCastHandCardFaceDownWithoutPaying(nil, FromCard(BindCard("missing"))), MayCastHandCardFaceDownWithoutPaying(You, nil), MayCastHandCardFaceDownWithoutPaying(You, FromCard(BindCard("missing")))} {
		if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", WithAbility(Spell(effect)))); err == nil {
			t.Fatal("accepted invalid declaration")
		}
	}
}

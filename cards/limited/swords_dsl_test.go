package limited

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestSwordsToPlowsharesTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Swords to Plowshares")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	ability := mage.UnwrapAbility(card.Abilities()[0]).(*mage.SpellAbility)
	if _, ok := ability.Targets()[0].(mage.TargetBinding); !ok {
		t.Fatal("missing typed creature target")
	}
	if ability.Effects()[0].Properties().TargetPurposeOverride != mage.AITargetExile {
		t.Fatal("missing exile metadata")
	}
}

func TestSwordsToPlowsharesOutcomes(t *testing.T) {
	for _, scenario := range []string{"exile", "indestructible", "illegal", "return", "own", "zero", "negative", "boost", "control"} {
		t.Run(scenario, func(t *testing.T) {
			g := gametest.NewTestGame(t)
			a, b := g.GetPlayer(gametest.PlayerA), g.GetPlayer(gametest.PlayerB)
			owner := b.PlayerID()
			if scenario == "own" {
				owner = a.PlayerID()
			}
			power, gain := 3, 3
			if scenario == "zero" {
				power, gain = 0, 0
			}
			if scenario == "negative" {
				power, gain = -2, 0
			}
			card := mage.NewCreature("Swords target", "", power, 4)
			card.SetOwner(owner)
			target := g.PutOnBattlefield(card, owner)
			if scenario == "indestructible" {
				target.GrantBaseAttr(core.Indestructible)
			}
			spell, err := mage.CreateCard("Swords to Plowshares")
			if err != nil {
				t.Fatal(err)
			}
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.White, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if scenario == "control" {
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			if scenario == "illegal" || scenario == "return" {
				g.RemoveFromBattlefield(target)
				if scenario == "return" {
					g.PutOnBattlefield(card, owner)
				}
				gain = 0
			}
			if scenario == "boost" {
				target.AddCounter(core.P1P1, 2)
				gain = 5
			}
			g.ResolveStack()
			wantA, wantB := 20, 20+gain
			if scenario == "own" || scenario == "control" {
				wantA, wantB = 20+gain, 20
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("life A=%d B=%d, want A=%d B=%d", a.Life(), b.Life(), wantA, wantB)
			}
			if scenario != "illegal" && scenario != "return" {
				g.AssertExileCount("Swords target", 1)
			}
			survives := scenario == "return"
			if (g.FindPermanent(card.ID()) != nil) != survives {
				t.Fatal("wrong creature destination")
			}
		})
	}
}

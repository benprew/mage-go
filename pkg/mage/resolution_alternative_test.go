package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type resolutionAlternativePlayer struct {
	*BasePlayer
	accept                bool
	mode, offers, choices int
}

func (p *resolutionAlternativePlayer) ChooseMayAbility(string) bool    { p.offers++; return p.accept }
func (p *resolutionAlternativePlayer) ChooseMode([]string, string) int { p.choices++; return p.mode }

func TestResolutionAlternativeManaPolicy(t *testing.T) {
	for _, scenario := range []string{"life-only", "mana-only", "both-life", "both-mana", "decline", "neither", "mana-source", "nested", "rollback", "reduced-only", "reduced-neither"} {
		t.Run(scenario, func(t *testing.T) {
			a := &resolutionAlternativePlayer{BasePlayer: NewBasePlayer("A"), accept: scenario != "decline"}
			g := NewGame(a, NewBasePlayer("B"))
			source := g.PutOnBattlefield(NewArtifact("Source", "", WithStaticAbility(FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(game *Game, id uuid.UUID) error { game.AddActivationCostReduction(id, 3); return nil }))), a.PlayerID())
			g.effects.Rules.ActivationCostReductions[source.ID()] = 3
			if scenario == "mana-only" || scenario == "neither" || scenario == "reduced-neither" {
				a.SetLife(0)
			}
			if scenario == "both-life" {
				a.mode = 1
			}
			if scenario == "mana-only" || scenario == "both-life" || scenario == "both-mana" || scenario == "decline" {
				a.ManaPool().Add(core.Colorless, 3)
			}
			if scenario == "reduced-only" || scenario == "reduced-neither" {
				a.ManaPool().Add(core.Colorless, 1)
			}
			var land *Permanent
			if scenario == "mana-source" {
				land = g.PutOnBattlefield(NewLand("Mana", WithManaAbility(core.Black)), a.PlayerID())
			}
			mana := "{3}"
			if scenario == "mana-source" {
				mana = "{B}"
			}
			cost := EitherCost(ManaCostOf(mana), LifePayCost(1))
			if scenario == "nested" {
				cost = EitherCost(ManaCostOf("{4}"), cost)
			}
			costs := Costs(cost)
			if scenario == "rollback" {
				costs = Costs(cost, LifePayCost(20))
			}
			effect := UnlessPaid(You, costs, GainLifeFor(You, Fixed(5)))
			card := NewInstant("Payment", "", Spell(effect))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			beforeLife, beforeMana := a.Life(), a.ManaPool().TotalMana()
			if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			paid := scenario != "decline" && scenario != "neither" && scenario != "rollback" && scenario != "reduced-neither"
			lifePaid := scenario == "life-only" || scenario == "both-life" || scenario == "nested" || scenario == "reduced-only"
			wantLife := beforeLife
			if lifePaid {
				wantLife--
			}
			if !paid {
				wantLife += 5
			}
			wantMana := beforeMana
			if paid && !lifePaid && land == nil {
				wantMana -= 3
			}
			if a.Life() != wantLife || a.ManaPool().TotalMana() != wantMana {
				t.Fatalf("life/mana=%d/%d want=%d/%d", a.Life(), a.ManaPool().TotalMana(), wantLife, wantMana)
			}
			wantChoices := 0
			if scenario == "both-life" || scenario == "both-mana" || scenario == "mana-source" {
				wantChoices = 1
			}
			wantOffers := 1
			if scenario == "neither" || scenario == "reduced-neither" {
				wantOffers = 0
			}
			if a.choices != wantChoices || a.offers != wantOffers {
				t.Fatalf("choices/offers=%d/%d want=%d/%d", a.choices, a.offers, wantChoices, wantOffers)
			}
			if land != nil && !g.FindPermanent(land.ID()).Tapped {
				t.Fatal("mana ability was not used")
			}
		})
	}
}

func TestResolutionAlternativeKeepsActivationPolicy(t *testing.T) {
	a := &resolutionAlternativePlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	g.effects.Rules.ActivationCostReductions[source.ID()] = 2
	a.ManaPool().Add(core.Colorless, 1)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{EitherCost(ManaCostOf("{3}"), LifePayCost(1))}, ApplyActivationReduction: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || a.ManaPool().TotalMana() != 0 || a.choices != 1 {
		t.Fatal("activation reduction policy changed")
	}
	clone := g.Clone()
	if err := ApplyEffect(clone, UnlessPaid(You, Costs(EitherCost(ManaCostOf("{3}"), LifePayCost(1))), GainLifeFor(You, Fixed(4))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 {
		t.Fatal("clone changed original payment")
	}
}

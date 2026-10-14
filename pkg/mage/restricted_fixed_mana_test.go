package mage

import (
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestRestrictedFixedManaPaths(t *testing.T) {
	for _, path := range []string{"manual", "planned", "replaced"} {
		t.Run(path, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.SetStep(PrecombatMain)
			land := g.PutOnBattlefield(NewLand("Restricted", WithAbility(Activate(Costs(TapCost(This)), AddRestrictedMana([]ManaProduction{{Color: Colorless, Amount: 3}}, ArtifactSpellsOnly{})))), a.PlayerID())
			ability := land.Card.Abilities()[0]
			productions := ManaProductionsForAbility(ability)
			if !IsActivatedManaAbility(ability) || len(productions) != 1 || productions[0].Amount != 3 || productions[0].Restriction == nil {
				t.Fatalf("metadata = %+v", productions)
			}
			ctx := SpellContextForCard(NewArtifact("Eligible", "{3}"))
			if !g.CanAfford(a.PlayerID(), ManaCost{Generic: 3}, ctx) || g.CanAfford(a.PlayerID(), ManaCost{Generic: 3}, nil) || land.Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("query changed state or lost restriction")
			}
			color := Colorless
			if path == "replaced" {
				if err := ApplyEffect(g, ReplaceTappedManaTypeForPlayerThisTurn(You, IsLand, Blue), land.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				color = Blue
			}
			if path == "planned" {
				spell := NewArtifact("Eligible", "{3}")
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(land.ID()).Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 1 {
					t.Fatal("planned payment did not execute")
				}
				return
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			pool := a.ManaPool()
			if g.StackSize() != 0 || pool.Count(color) != 3 || !pool.CanPay(ManaCost{Generic: 3}, ctx) || pool.CanPay(ManaCost{Generic: 3}, nil) || pool.CanPay(ManaCost{Generic: 3}, SpellContextForCard(NewCreature("Wrong", "{3}", 1, 1))) {
				t.Fatal("production or replacement lost restriction")
			}
			clone := g.Clone()
			if err := clone.GetPlayer(a.PlayerID()).ManaPool().Pay(ManaCost{Generic: 3}, ctx); err != nil || pool.TotalMana() != 3 {
				t.Fatal("clone pool or restriction changed")
			}
		})
	}
}

func TestRestrictedFixedManaDeclarations(t *testing.T) {
	for _, effect := range []Effect{
		AddRestrictedMana([]ManaProduction{{Color: AnyColor, Amount: 1}}, ArtifactSpellsOnly{}),
		AddRestrictedMana([]ManaProduction{{Color: Colorless, Amount: -1}}, ArtifactSpellsOnly{}),
		AddRestrictedMana([]ManaProduction{{Color: Colorless, Amount: 0}}, ArtifactSpellsOnly{}),
		AddRestrictedMana([]ManaProduction{{Color: Blue, Amount: 1, AnyCombination: true}}, ArtifactSpellsOnly{}),
		AddRestrictedMana([]ManaProduction{{Color: Colorless, Amount: 1}}, nil),
		AddRestrictedMana(nil, ArtifactSpellsOnly{}),
	} {
		card := NewLand("Invalid", WithAbility(Activate(Costs(TapCost(This)), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid restriction declaration")
		}
	}
}

type restrictedManaBatchProbe struct {
	tappedManaTypeReplacement
	sizes []int
}

func (r *restrictedManaBatchProbe) Replace(action Action, g *Game) Action {
	r.sizes = append(r.sizes, len(action.(*manaProductionAction).productions))
	return r.tappedManaTypeReplacement.Replace(action, g)
}

func TestRestrictedFixedManaBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	productions := []ManaProduction{{Color: Blue, Amount: 1}, {Color: Green, Amount: 2}}
	effect := AddRestrictedMana(productions, CreatureSpellsOnly{})
	productions[0].Color = Red
	land := g.PutOnBattlefield(NewLand("Batch", WithAbility(Activate(Costs(TapCost(This)), effect))), a.PlayerID())
	metadata := ManaProductionsForAbility(land.Card.Abilities()[0])
	if len(metadata) != 2 || metadata[0].Color != Blue {
		t.Fatal("caller changed saved productions")
	}
	metadata[0].Amount = 9
	if ManaProductionsForAbility(land.Card.Abilities()[0])[0].Amount != 1 {
		t.Fatal("metadata query changed declaration")
	}
	probe := &restrictedManaBatchProbe{tappedManaTypeReplacement: tappedManaTypeReplacement{player: a.PlayerID(), filter: IsLand, to: Colorless}}
	g.effects.AddReplacement(probe)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if len(probe.sizes) != 1 || probe.sizes[0] != 2 {
		t.Fatalf("production was split into events: %v", probe.sizes)
	}
	pool := a.ManaPool()
	if pool.Count(Colorless) != 3 || !pool.CanPay(ManaCost{Generic: 3}, &SpellPaymentContext{IsCreature: true}) || pool.CanPay(ManaCost{Generic: 1}, &SpellPaymentContext{IsArtifact: true}) {
		t.Fatal("batch replacement lost restriction")
	}
}

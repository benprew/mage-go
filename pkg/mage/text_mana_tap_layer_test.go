package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextManaTapLayerDeclaration(t *testing.T) {
	for _, bonus := range []bool{false, true} {
		for _, valid := range []bool{false, true} {
			var ability Ability
			if bonus {
				a := NewManaBonusAbility(IsLand, core.Red).WithLayeredManaTapTextRevision()
				if !valid {
					a.AttachedOnly = true
				}
				ability = a
			} else {
				a := WhenEvent(LandTappedForMana(), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredManaTapTextRevision()
				if !valid {
					a.FromGraveyard()
				}
				ability = a
			}
			card := NewEnchantment("Declaration", "", WithAbility(ability))
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != valid {
				t.Fatalf("bonus=%v valid=%v: %v", bonus, valid, err)
			}
		}
	}
}

func TestTextManaTapLayerRejects(t *testing.T) {
	for _, bonus := range []bool{false, true} {
		for _, scenario := range []string{"unmarked", "grant", "face-down", "missing-marker", "changed-kind", "missing-source", "nil-source", "unknown-id", "no-builder", "no-group", "invalid-change"} {
			t.Run(fmt.Sprint(bonus)+"/"+scenario, func(t *testing.T) {
				build := func(marked, changed bool) CardFactory {
					return func() Card {
						var ability Ability
						if bonus {
							a := NewManaBonusAbility(IsLand, core.Red)
							if changed {
								a = NewAttachedManaBonusAbility(core.Red)
							}
							if marked {
								a.WithLayeredManaTapTextRevision()
							} else {
								a.WithManaTapTextRevision()
							}
							ability = a
						} else {
							a := WhenEvent(LandTappedForMana(), GainLifeFor(You, Fixed(1))).(*GenericTriggered)
							if changed {
								a.eventType = core.EvtUpkeep
							}
							if marked {
								a.WithLayeredManaTapTextRevision()
							} else {
								a.WithManaTapTextRevision()
							}
							ability = a
						}
						option := WithAbility(ability)
						if scenario != "no-group" {
							option = WithDefinitionAbilityGroup(0, option)
						}
						return NewEnchantment("Trigger", "", option)
					}
				}
				plan, err := NewCardDefinitionPlan(build(scenario != "unmarked", false))
				if err != nil {
					t.Fatal(err)
				}
				card, err := plan.Build()
				if err != nil {
					t.Fatal(err)
				}
				if scenario != "no-builder" {
					WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) {
						return build(scenario != "missing-marker", scenario == "changed-kind"), nil
					})(card.(*BaseCard))
				}
				a := NewBasePlayer("A")
				g := NewGame(a, NewBasePlayer("B"))
				p := g.PutOnBattlefield(card, a.PlayerID())
				id := p.RuntimeAbilities[0].AbilityID()
				switch scenario {
				case "grant":
					g.MutablePermanent(p.ID()).RuntimeAbilities[0] = &grantedByEffect{Ability: p.RuntimeAbilities[0]}
				case "face-down":
					g.MutablePermanent(p.ID()).FaceDown = true
				case "missing-source":
					g.BouncePermanentToHand(p)
				case "unknown-id":
					id = uuid.New()
				}
				before := p.RuntimeAbilities[0]
				effects := len(g.effects.effects)
				input := p
				if scenario == "nil-source" {
					input = nil
				}
				change, _ := ReplaceColorWord(core.Red, core.Green)
				if scenario == "invalid-change" {
					change = TextChange{}
				}
				if err := g.AddPrintedManaTapTextEffect(input, id, core.EndOfTurn, change); err == nil {
					t.Fatal("invalid installation accepted")
				}
				if len(g.effects.effects) != effects || p.RuntimeAbilities[0] != before || len(p.layeredManaTapOrigins) != 0 {
					t.Fatal("rejection changed live state")
				}
			})
		}
	}
}

func TestTextManaTapLayerBaselineAndContext(t *testing.T) {
	for _, bonus := range []bool{false, true} {
		t.Run(fmt.Sprint(bonus), func(t *testing.T) {
			build := func(color core.Color) CardFactory {
				return func() Card {
					var ability Ability
					if bonus {
						ability = NewManaBonusAbility(IsLand, color).WithLayeredManaTapTextRevision()
					} else {
						ability = WhenEvent(LandTappedForMana(), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredManaTapTextRevision()
					}
					return NewEnchantment("Source", "", WithDefinitionAbilityGroup(0, WithAbility(ability)))
				}
			}
			plan, err := NewCardDefinitionPlan(build(core.Red))
			if err != nil {
				t.Fatal(err)
			}
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			WithTextDefinitionBuilder(func(changes []TextChange) (CardFactory, error) {
				color := core.Red
				for _, change := range changes {
					color = change.ApplyColor(color)
				}
				return build(color), nil
			})(card.(*BaseCard))
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(card, a.PlayerID())
			id := source.RuntimeAbilities[0].AbilityID()
			baseline, _ := ReplaceColorWord(core.Red, core.Green)
			first, _ := ReplaceColorWord(core.Green, core.Blue)
			second, _ := ReplaceColorWord(core.Blue, core.Black)
			if err := g.RevisePrintedManaTapTrigger(source, id, baseline); err != nil {
				t.Fatal(err)
			}
			if err := g.AddPrintedManaTapTextEffect(source, id, core.EndOfTurn, first); err != nil {
				t.Fatal(err)
			}
			if err := g.AddPrintedManaTapTextEffect(g.FindPermanent(source.ID()), id, core.Indefinite, second); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			clone.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			clone.ApplyContinuousEffects()
			clone.GetEffects().RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			check := func(game *Game, want core.Color, controller uuid.UUID) {
				t.Helper()
				p := game.FindPermanent(source.ID())
				ability := p.RuntimeAbilities[0]
				if ability.AbilityID() != id || ability.Controller() != controller || ability.Source() != source.ID() {
					t.Fatal("declaration identity or context changed")
				}
				if bonus && ability.(*ManaBonusAbility).BonusMana != want {
					t.Fatal("text did not start from saved baseline")
				}
			}
			for range 3 {
				g.ApplyContinuousEffects()
			}
			check(g, core.Black, a.PlayerID())
			check(clone, core.Green, b.PlayerID())
			land := clone.PutOnBattlefield(NewLand("Land", WithManaAbility(core.White)), a.PlayerID())
			if err := clone.TapForMana(a.PlayerID(), land.ID()); err != nil {
				t.Fatal(err)
			}
			clone.PutTriggersOnStack()
			clone.ResolveStack()
			if !bonus && (clone.GetPlayer(b.PlayerID()).Life() != 21 || clone.GetPlayer(a.PlayerID()).Life() != 20) {
				t.Fatal("trigger used shared controller context")
			}
			phased := g.PhaseOut(source.ID())
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if err := g.AddPrintedManaTapTextEffect(source, id, core.EndOfTurn, first); err == nil {
				t.Fatal("phased source accepted")
			}
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			check(g, core.Green, a.PlayerID())
			if err := g.AddPrintedManaTapTextEffect(g.FindPermanent(source.ID()), id, core.EndOfTurn, first, second); err != nil {
				t.Fatal(err)
			}
			check(g, core.Black, a.PlayerID())
			remover := g.PutOnBattlefield(NewArtifact("Remover", ""), a.PlayerID())
			remove := RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
			remove.SetSourceID(remover.ID())
			g.AddContinuousEffect(remove)
			g.ApplyContinuousEffects()
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			g.ApplyContinuousEffects()
			check(g, core.Green, a.PlayerID())
			if err := g.RevisePrintedManaTapTrigger(g.FindPermanent(source.ID()), id, baseline); err == nil {
				t.Fatal("direct revision changed surviving layer")
			}
			g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			snapshot := g.LKIAbilities(source.ID())
			if len(snapshot) != 1 || bonus && snapshot[0].(*ManaBonusAbility).BonusMana != core.Green {
				t.Fatal("LKI lost last layer result")
			}
			check(clone, core.Green, b.PlayerID())
		})
	}
}

package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextManaTapTriggerDeclaration(t *testing.T) {
	for _, scenario := range []string{"valid", "other-event", "state", "graveyard", "extra-zone", "duplicate-type"} {
		t.Run(scenario, func(t *testing.T) {
			pattern := LandTappedForMana()
			if scenario == "duplicate-type" {
				pattern = pattern.OfManaType(core.Blue).OfManaType(core.Green)
			}
			trigger := WhenEvent(pattern, AddManaAmount(core.Green, Fixed(1))).(*GenericTriggered).WithManaTapTextRevision()
			switch scenario {
			case "other-event":
				trigger.eventType = core.EvtTapped
			case "state":
				trigger.AsStateTrigger()
			case "graveyard":
				trigger.FromGraveyard()
			case "extra-zone":
				trigger.InZone(core.ZoneHand)
			}
			card := NewEnchantment("Declaration", "", WithAbility(trigger))
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != (scenario == "valid") {
				t.Fatalf("%s: %v", scenario, err)
			}
		})
	}
	for _, scenario := range []string{"fixed", "attached", "produced", "invalid-color", "mixed", "attached-filter"} {
		t.Run(scenario, func(t *testing.T) {
			bonus := NewManaBonusAbility(IsLand, core.Green).WithManaTapTextRevision()
			switch scenario {
			case "attached":
				bonus = NewAttachedManaBonusAbility(core.Green).WithManaTapTextRevision()
			case "produced":
				bonus = NewManaFlareAbility(IsLand).WithManaTapTextRevision()
			case "invalid-color":
				bonus.BonusMana = core.AnyColor
			case "mixed":
				bonus.AttachedOnly, bonus.MatchProduced = true, true
			case "attached-filter":
				bonus.AttachedOnly = true
			}
			card := NewEnchantment("Declaration", "", WithAbility(bonus))
			valid := scenario == "fixed" || scenario == "attached" || scenario == "produced"
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != valid {
				t.Fatalf("%s: %v", scenario, err)
			}
		})
	}
}

func TestTextManaTapTriggerRejects(t *testing.T) {
	for _, bonus := range []bool{false, true} {
		for _, scenario := range []string{"unmarked", "grant", "face-down", "missing-marker", "changed-kind", "missing-source", "nil-source", "unknown-id", "no-builder", "no-group", "invalid-change"} {
			t.Run(fmt.Sprint(bonus)+"/"+scenario, func(t *testing.T) {
				build := func(marked, changed bool) CardFactory {
					return func() Card {
						var ability Ability
						if bonus {
							a := NewAttachedManaBonusAbility(core.Green)
							if changed {
								a = NewManaFlareAbility(IsLand)
							}
							if marked {
								a.WithManaTapTextRevision()
							}
							ability = a
						} else {
							a := WhenEvent(LandTappedForMana(), GainLifeFor(You, Fixed(1))).(*GenericTriggered)
							if changed {
								a.eventType = core.EvtUpkeep
							}
							if marked {
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
				input := p
				if scenario == "nil-source" {
					input = nil
				}
				change, _ := ReplaceColorWord(core.Red, core.Green)
				if scenario == "invalid-change" {
					change = TextChange{}
				}
				if err := g.RevisePrintedManaTapTrigger(input, id, change); err == nil {
					t.Fatal("invalid revision accepted")
				}
				if p.RuntimeAbilities[0] != before {
					t.Fatal("rejected revision changed declaration")
				}
			})
		}
	}
}

func TestTextManaTapTriggerImmediate(t *testing.T) {
	build := func(color core.Color) CardFactory {
		return func() Card {
			producer := BindPlayer("producer")
			trigger := WhenEvent(LandTappedForMana().WithPlayer(producer), AddManaToPlayer(FromPlayer(producer), color, Fixed(1))).(*GenericTriggered).WithManaTapTextRevision()
			return NewEnchantment("Trigger", "", WithDefinitionAbilityGroup(0, WithAbility(trigger)))
		}
	}
	plan, err := NewCardDefinitionPlan(build(core.Green))
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(core.Blue), nil })(card.(*BaseCard))
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(card, a.PlayerID())
	land := g.PutOnBattlefield(NewLand("Land", WithManaAbility(core.Red)), b.PlayerID())
	change, _ := ReplaceColorWord(core.Green, core.Blue)
	if err := g.RevisePrintedManaTapTrigger(source, source.RuntimeAbilities[0].AbilityID(), change); err != nil {
		t.Fatal(err)
	}
	if err := g.TapForMana(b.PlayerID(), land.ID()); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if b.ManaPool().Count(core.Blue) != 1 || b.ManaPool().Count(core.Green) != 0 || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
		t.Fatal("revised mana trigger lost immediate timing or actor")
	}
}

func TestTextManaTapTriggerBonusSemantics(t *testing.T) {
	for _, kind := range []string{"fixed", "produced", "attached"} {
		t.Run(kind, func(t *testing.T) {
			build := func(subtype string, color core.Color) CardFactory {
				return func() Card {
					filter := HasSubType(subtype)
					bonus := NewManaBonusAbility(filter, color)
					if kind == "produced" {
						bonus = NewManaFlareAbility(filter)
					}
					if kind == "attached" {
						bonus = NewAttachedManaBonusAbility(color)
					}
					options := []CardOption{WithDefinitionAbilityGroup(0, WithAbility(bonus.WithManaTapTextRevision()))}
					if kind == "attached" {
						options = append(options, WithSubTypes("Aura"))
					}
					return NewEnchantment("Bonus", "", options...)
				}
			}
			plan, err := NewCardDefinitionPlan(build("Mountain", core.Red))
			if err != nil {
				t.Fatal(err)
			}
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			WithTextDefinitionBuilder(func(changes []TextChange) (CardFactory, error) {
				word, color := core.MountainLandType, core.Red
				for _, change := range changes {
					word = change.ApplyBasicLand(word)
					color = change.ApplyColor(color)
				}
				return build(word.String(), color), nil
			})(card.(*BaseCard))
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(card, a.PlayerID())
			sibling := g.PutOnBattlefield(build("Mountain", core.Red)(), a.PlayerID())
			island := g.PutOnBattlefield(NewLand("Island", WithSubTypes("Island"), WithManaAbility(core.Blue)), b.PlayerID())
			mountain := g.PutOnBattlefield(NewLand("Mountain", WithSubTypes("Mountain"), WithManaAbility(core.Red)), b.PlayerID())
			if kind == "attached" {
				g.Attach(source.ID(), island.ID())
			}
			id := source.RuntimeAbilities[0].AbilityID()
			landChange, _ := ReplaceBasicLandWord(core.MountainLandType, core.IslandLandType)
			colorChange, _ := ReplaceColorWord(core.Red, core.Blue)
			if err := g.RevisePrintedManaTapTrigger(source, id, landChange, colorChange); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			nextLand, _ := ReplaceBasicLandWord(core.IslandLandType, core.SwampLandType)
			nextColor, _ := ReplaceColorWord(core.Blue, core.Green)
			if err := g.RevisePrintedManaTapTrigger(source, id, nextLand, nextColor); err != nil {
				t.Fatal(err)
			}
			if err := clone.AutoTapForCost(b.PlayerID(), core.ManaCost{Blue: 2}); err != nil {
				t.Fatal("clone lost revised planning:", err)
			}
			if clone.GetPlayer(b.PlayerID()).ManaPool().Count(core.Blue) != 2 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
				t.Fatal("clone lost revised production or recipient")
			}
			if island.Tapped || b.ManaPool().TotalMana() != 0 {
				t.Fatal("clone shared payment state")
			}
			if kind != "attached" {
				if err := g.AutoTapForCost(b.PlayerID(), core.ManaCost{Blue: 2}); err == nil {
					t.Fatal("planner retained previous filter")
				}
			} else {
				if err := g.AutoTapForCost(b.PlayerID(), core.ManaCost{Blue: 2}); err == nil {
					t.Fatal("planner retained previous color")
				}
			}
			if island.Tapped || mountain.Tapped || b.ManaPool().TotalMana() != 0 {
				t.Fatal("failed planning leaked state")
			}
			phased := g.PhaseOut(source.ID())
			if err := g.RevisePrintedManaTapTrigger(source, id, nextColor); err == nil {
				t.Fatal("phased source accepted")
			}
			g.PhaseIn(phased)
			remover := g.PutOnBattlefield(NewArtifact("Remover", ""), a.PlayerID())
			removal := RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
			removal.SetSourceID(remover.ID())
			g.AddContinuousEffect(removal)
			g.ApplyContinuousEffects()
			if err := g.RevisePrintedManaTapTrigger(source, id, nextColor); err == nil {
				t.Fatal("removed declaration accepted")
			}
			g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			g.ApplyContinuousEffects()
			swamp := g.PutOnBattlefield(NewLand("Swamp", WithSubTypes("Swamp"), WithManaAbility(core.Black)), b.PlayerID())
			target := swamp
			bonusColor := core.Green
			if kind == "attached" {
				target = island
			}
			if kind == "produced" {
				bonusColor = core.Black
			}
			if err := g.TapForMana(b.PlayerID(), target.ID()); err != nil {
				t.Fatal(err)
			}
			if b.ManaPool().Count(bonusColor) != 1 && kind != "produced" || kind == "produced" && b.ManaPool().Count(core.Black) != 2 {
				t.Fatal("restoration lost ordered revision")
			}
			if source.RuntimeAbilities[0].AbilityID() != id || sibling.RuntimeAbilities[0] == source.RuntimeAbilities[0] {
				t.Fatal("identity or sibling isolation lost")
			}
			original := card.Abilities()[0].(*ManaBonusAbility)
			if original.BonusMana != core.Red && kind != "produced" || !original.AttachedOnly && !original.Filter.Match(mountain, g) {
				t.Fatal("printed origin changed")
			}
			stale := g.FindPermanent(source.ID())
			g.BouncePermanentToHand(stale)
			a.RemoveFromHand(card.ID())
			returned := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.RevisePrintedManaTapTrigger(stale, id, colorChange); err == nil {
				t.Fatal("returned source inherited old visit")
			}
			if returned.RuntimeAbilities[0] != card.Abilities()[0] {
				t.Fatal("return lost printed origin")
			}
		})
	}
}

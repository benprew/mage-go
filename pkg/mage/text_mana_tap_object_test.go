package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextManaTapObjectRejectsAtomic(t *testing.T) {
	for _, bonus := range []bool{false, true} {
		for _, scenario := range []string{"empty", "invalid", "duration", "nil", "face-down", "phased", "departed", "returned", "no-builder", "no-group", "missing-runtime", "changed-kind", "changed-capture", "changed-target", "changed-subtype", "changed-keyword", "changed-recipient", "changed-produced", "changed-mana-symbol", "nil-runtime", "changed-runtime"} {
			t.Run(map[bool]string{false: "trigger", true: "bonus"}[bonus]+"/"+scenario, func(t *testing.T) {
				build := func(changed bool) CardFactory {
					return func() Card {
						var rule Ability = WhenEvent(LandTappedForMana().OfManaType(core.Black), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredManaTapTextRevision()
						if bonus {
							rule = NewManaFlareAbility(IsLand).WithLayeredManaTapTextRevision()
						}
						if changed {
							switch scenario {
							case "changed-kind":
								if bonus {
									rule = WhenEvent(LandTappedForMana(), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredManaTapTextRevision()
								} else {
									rule = NewManaFlareAbility(IsLand).WithLayeredManaTapTextRevision()
								}
							case "changed-capture":
								rule = WhenEvent(LandTappedForMana().WithPlayer(BindPlayer("producer")), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredManaTapTextRevision()
							case "changed-target":
								rule = WhenEvent(LandTappedForMana(), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredManaTapTextRevision()
								rule.(*GenericTriggered).AddTarget(TargetPermanent(IsCreature))
							case "changed-recipient":
								rule = NewControllerManaBonusAbility(IsLand, core.Blue).WithLayeredManaTapTextRevision()
							case "changed-mana-symbol":
								rule = NewManaFlareAbility(IsLand).WithLayeredManaTapTextRevision()
								rule.(*ManaBonusAbility).BonusMana = core.Green
							case "changed-produced":
								rule = NewManaBonusAbility(IsLand, core.Blue).WithLayeredManaTapTextRevision()
							}
						}
						option := WithAbility(rule)
						if scenario != "no-group" {
							option = WithDefinitionAbilityGroup(0, option)
						}
						opts := []CardOption{WithManaTapSpellTextContinuation(), option}
						if changed && scenario == "changed-subtype" {
							opts = append(opts, WithSubTypes("Shrine"))
						}
						if changed && scenario == "changed-keyword" {
							opts = append(opts, WithKeyword(core.Shroud))
						}
						return NewEnchantment("Event", "", opts...)
					}
				}
				plan, err := NewCardDefinitionPlan(build(false))
				if err != nil {
					t.Fatal(err)
				}
				card, err := plan.Build()
				if err != nil {
					t.Fatal(err)
				}
				if scenario != "no-builder" {
					WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(true), nil })(card.(*BaseCard))
				}
				g, a, _ := newSpellCopyGame()
				p := g.PutOnBattlefield(card, a.PlayerID())
				change, _ := ReplaceColorWord(core.Black, core.Blue)
				changes := []TextChange{change}
				duration := core.Indefinite
				switch scenario {
				case "empty":
					changes = nil
				case "invalid":
					changes = []TextChange{{}}
				case "duration":
					duration = core.EndOfCombat
				case "nil":
					p = nil
				case "face-down":
					g.MutablePermanent(p.ID()).FaceDown = true
				case "phased":
					g.PhaseOut(p.ID())
				case "departed":
					g.RemoveFromBattlefield(p)
				case "returned":
					g.RemoveFromBattlefield(p)
					g.PutOnBattlefield(card, a.PlayerID())
				case "nil-runtime":
					if bonus {
						g.MutablePermanent(p.ID()).RuntimeAbilities[0] = (*ManaBonusAbility)(nil)
					} else {
						g.MutablePermanent(p.ID()).RuntimeAbilities[0] = (*GenericTriggered)(nil)
					}
				case "changed-runtime":
					if bonus {
						fresh := *p.RuntimeAbilities[0].(*ManaBonusAbility)
						fresh.controllerRecipient = true
						g.MutablePermanent(p.ID()).RuntimeAbilities[0] = &fresh
					} else {
						fresh := *p.RuntimeAbilities[0].(*GenericTriggered)
						fresh.targets = []Target{TargetPermanent(IsCreature)}
						g.MutablePermanent(p.ID()).RuntimeAbilities[0] = &fresh
					}
				case "missing-runtime":
					g.MutablePermanent(p.ID()).RuntimeAbilities = nil
				}
				before := len(g.effects.effects)
				if err := g.AddPrintedManaTapObjectTextEffect(p, duration, changes...); err == nil {
					t.Fatal("invalid complete context accepted")
				}
				if len(g.effects.effects) != before {
					t.Fatal("failed preflight installed an effect")
				}
			})
		}
	}
}

func TestTextManaTapObjectLifecycle(t *testing.T) {
	for _, bonus := range []bool{false, true} {
		t.Run(map[bool]string{false: "trigger", true: "bonus"}[bonus], func(t *testing.T) {
			g, a, b := newSpellCopyGame()
			var build func([]TextChange) CardFactory
			build = func(changes []TextChange) CardFactory {
				return func() Card {
					landWord, colorWord := core.MountainLandType, core.Red
					for _, change := range changes {
						landWord = change.ApplyBasicLand(landWord)
						colorWord = change.ApplyColor(colorWord)
					}
					player := BindPlayer("producer")
					var rule Ability = WhenEvent(LandTappedForMana().OfManaType(colorWord).WithPlayer(player), GainLifeFor(FromPlayer(player), Fixed(1))).(*GenericTriggered).WithLayeredManaTapTextRevision()
					if bonus {
						rule = NewManaFlareAbility(And(IsLand, HasSubType(landWord.String()))).WithLayeredManaTapTextRevision()
					}
					return NewEnchantment("Event", "", WithManaTapSpellTextContinuation(), WithDefinitionAbilityGroup(0, WithAbility(rule)), WithTextDefinitionBuilder(func(words []TextChange) (CardFactory, error) { return build(words), nil }))
				}
			}
			plan, err := NewCardDefinitionPlan(build(nil))
			if err != nil {
				t.Fatal(err)
			}
			card, err := plan.Build()
			if err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			id := source.RuntimeAbilities[0].AbilityID()
			first, _ := ReplaceBasicLandWord(core.MountainLandType, core.IslandLandType)
			second, _ := ReplaceBasicLandWord(core.IslandLandType, core.ForestLandType)
			if !bonus {
				first, _ = ReplaceColorWord(core.Red, core.Blue)
				second, _ = ReplaceColorWord(core.Blue, core.Green)
			}
			if err := g.AddPrintedManaTapObjectTextEffect(source, core.EndOfTurn, first); err != nil {
				t.Fatal(err)
			}
			if err := g.AddPrintedManaTapObjectTextEffect(g.FindPermanent(source.ID()), core.Indefinite, second); err != nil {
				t.Fatal(err)
			}
			for range 3 {
				g.ApplyContinuousEffects()
			}
			clone := g.Clone()
			tap := func(game *Game, name string, want bool) {
				t.Helper()
				player := game.GetPlayer(b.PlayerID())
				player.ManaPool().Clear()
				land := game.PutOnBattlefield(NewLand(name, WithSubTypes(name), WithManaAbility(map[string]core.Color{"Mountain": core.Red, "Island": core.Blue, "Forest": core.Green}[name])), b.PlayerID())
				life := player.Life()
				if err := game.TapForMana(b.PlayerID(), land.ID()); err != nil {
					t.Fatal(err)
				}
				game.PutTriggersOnStack()
				game.ResolveStack()
				if bonus {
					amount := 1
					if want {
						amount = 2
					}
					if player.ManaPool().TotalMana() != amount {
						t.Fatal("ordered land words or bonus timing lost")
					}
				} else {
					gain := 0
					if want {
						gain = 1
					}
					if player.Life() != life+gain {
						t.Fatal("event filter or recipient lost")
					}
				}
			}
			tap(g, "Forest", true)
			tap(g, "Mountain", false)
			tap(g, "Island", false)
			phased := g.PhaseOut(source.ID())
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			tap(g, "Mountain", false)
			if err := g.AddPrintedManaTapObjectTextEffect(source, core.Indefinite, first); err == nil {
				t.Fatal("phased source accepted")
			}
			g.PhaseIn(phased)
			g.ApplyContinuousEffects()
			tap(g, "Mountain", true)
			tap(g, "Forest", false)
			tap(clone, "Forest", true)
			g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			if g.FindPermanent(source.ID()).RuntimeAbilities[0].AbilityID() != id {
				t.Fatal("control lost declaration")
			}
			tap(g, "Mountain", true)
			remover := g.PutOnBattlefield(NewLand("Removal source"), a.PlayerID())
			removal := RemoveAllAbilities(source.ID(), core.WhileOnBattlefield)
			removal.SetSourceID(remover.ID())
			g.AddContinuousEffect(removal)
			g.ApplyContinuousEffects()
			if err := g.AddPrintedManaTapObjectTextEffect(g.FindPermanent(source.ID()), core.Indefinite, first); err == nil {
				t.Fatal("removed rule accepted")
			}
			tap(g, "Mountain", false)
			g.RemoveFromBattlefield(g.FindPermanent(remover.ID()))
			g.ApplyContinuousEffects()
			tap(g, "Mountain", true)
			values, err := g.CaptureCopiableValues(g.FindPermanent(source.ID()))
			if err != nil {
				t.Fatal(err)
			}
			copyCard, err := values.Build()
			if err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			g.PutOnBattlefield(copyCard, a.PlayerID())
			tap(g, "Mountain", true)
			tap(g, "Forest", false)
		})
	}
}

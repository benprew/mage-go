package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextProtectionPTObjectAtomic(t *testing.T) {
	for _, scenario := range []string{"valid", "reverse", "nil-source", "empty", "invalid", "duration", "face-down", "phased", "returned", "no-builder", "no-group", "swapped", "lost-marker", "changed-condition", "changed-recipient", "changed-name", "changed-protection", "changed-subtype", "extra-rule", "keyword", "missing-protection", "nil-protection", "nil-static", "land-no-match", "missing-land-no-match", "missing-registration"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					effect := SourcePTWhileOpponentControlsNontokenColor(core.Red, 1, 1)
					if scenario == "changed-name" {
						effect = NamedCreaturesPTWhileOpponentControlsNontokenColor("Mixed", core.Red, 1, 1)
					}
					if changed {
						switch scenario {
						case "changed-condition":
							effect = SourcePTWhileOpponentControlsBasicLand("Swamp", 1, 1)
						case "changed-recipient":
							effect = NamedCreaturesPTWhileOpponentControlsNontokenColor("Mixed", core.Red, 1, 1)
						case "changed-name":
							effect = NamedCreaturesPTWhileOpponentControlsNontokenColor("Other", core.Red, 1, 1)
						}
					}
					static := StaticAbility(effect).WithLayeredPTTextRevision()
					protection := ProtectionFromColors(core.Red).WithLayeredColorTextRevision()
					if changed && scenario == "changed-protection" {
						protection = ProtectionFromColors(core.Red, core.Blue).WithLayeredColorTextRevision()
					}
					first, second := WithDefinitionAbilityGroup(1, WithAbility(static)), WithDefinitionAbilityGroup(2, WithAbility(protection))
					if scenario == "no-group" {
						second = WithAbility(protection)
					}
					if changed && scenario == "swapped" {
						first, second = WithDefinitionAbilityGroup(2, WithAbility(static)), WithDefinitionAbilityGroup(1, WithAbility(protection))
					}
					if scenario == "reverse" {
						first, second = second, first
					}
					opts := []CardOption{first, second}
					if !changed || scenario != "lost-marker" {
						opts = append(opts, WithProtectionPTSpellTextContinuation())
					}
					if changed && scenario == "extra-rule" {
						opts = append(opts, WithAbility(NewAction(ActionActivated, WithEffect(Regenerate(This)))))
					}
					if changed && scenario == "changed-subtype" {
						opts = append(opts, WithSubTypes("Beast"))
					}
					if changed && scenario == "keyword" {
						opts = append(opts, WithKeyword(core.Flying))
					}
					return NewCreature("Mixed", "", 2, 2, opts...)
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
			switch scenario {
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "phased":
				g.PhaseOut(p.ID())
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID())
			case "missing-protection", "missing-land-no-match":
				g.MutablePermanent(p.ID()).RuntimeAbilities = p.RuntimeAbilities[:1]
			case "nil-protection":
				g.MutablePermanent(p.ID()).RuntimeAbilities[1] = (*ProtectionAbility)(nil)
			case "nil-static":
				g.MutablePermanent(p.ID()).RuntimeAbilities[0] = (*StaticAbilityHolder)(nil)
			case "missing-registration":
				for i, e := range g.effects.effects {
					if g.printedStaticRegistration(e, p, p.RuntimeAbilities[0].AbilityID()) {
						g.effects.effects = append(g.effects.effects[:i], g.effects.effects[i+1:]...)
						break
					}
				}
			}
			change, _ := ReplaceColorWord(core.Red, core.Blue)
			if scenario == "land-no-match" || scenario == "missing-land-no-match" {
				change, _ = ReplaceBasicLandWord(core.SwampLandType, core.IslandLandType)
			}
			changes := []TextChange{change}
			duration := core.Indefinite
			if scenario == "empty" {
				changes = nil
			}
			if scenario == "invalid" {
				changes = append(changes, TextChange{})
			}
			if scenario == "duration" {
				duration = core.EndOfCombat
			}
			input := p
			if scenario == "nil-source" {
				input = nil
			}
			before, snapshot := len(g.effects.effects), g.FindPermanentIncludingPhased(p.ID())
			err = g.AddPrintedProtectionPTTextEffect(input, duration, changes...)
			if scenario == "valid" || scenario == "reverse" || scenario == "land-no-match" {
				if err != nil {
					t.Fatal(err)
				}
				want := before + 2
				if scenario == "land-no-match" {
					want = before
				}
				if len(g.effects.effects) != want {
					t.Fatal("missing sibling layers")
				}
				return
			}
			if err == nil {
				t.Fatal("invalid paired context accepted")
			}
			if len(g.effects.effects) != before || g.FindPermanentIncludingPhased(p.ID()) != snapshot {
				t.Fatal("failed preflight changed live state")
			}
		})
	}
}

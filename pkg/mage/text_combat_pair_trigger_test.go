package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextCombatPairTriggerDeclaration(t *testing.T) {
	for _, scenario := range []string{"valid", "other-event", "state", "graveyard", "extra-zone", "untyped", "wrong-kind", "wrong-subject"} {
		t.Run(scenario, func(t *testing.T) {
			trigger := WhenEvent(BlocksCreatureMatching(This, IsCreature), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithCombatPairTextRevision()
			switch scenario {
			case "other-event":
				trigger.eventType = core.EvtUpkeep
			case "state":
				trigger.AsStateTrigger()
			case "graveyard":
				trigger.FromGraveyard()
			case "extra-zone":
				trigger.InZone(core.ZoneHand)
			case "untyped":
				trigger.eventDefinition = nil
			case "wrong-kind":
				pattern := trigger.eventDefinition.(filteredCombatPairPattern)
				pattern.kind = core.EvtUpkeep
				trigger.eventDefinition = pattern
			case "wrong-subject":
				pattern := trigger.eventDefinition.(filteredCombatPairPattern)
				pattern.subject = nil
				trigger.eventDefinition = pattern
			}
			card := NewArtifact("Declaration", "", WithAbility(trigger))
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != (scenario == "valid") {
				t.Fatalf("%s: %v", scenario, err)
			}
		})
	}
}

func TestTextCombatPairTriggerRejects(t *testing.T) {
	for _, scenario := range []string{"unmarked", "grant", "face-down", "missing-marker", "changed-event", "missing-source", "nil-source", "unknown-id", "no-builder", "changed-direction", "invalid-change"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(marked, changed bool) CardFactory {
				return func() Card {
					trigger := WhenEvent(BlocksCreatureMatching(This, IsCreature), GainLifeFor(You, Fixed(1))).(*GenericTriggered)
					if marked {
						trigger.WithCombatPairTextRevision()
					}
					if changed && scenario != "changed-direction" {
						trigger.eventType = core.EvtUpkeep
					}
					if scenario == "changed-direction" && changed {
						trigger = WhenEvent(BecomesBlockedByCreatureMatching(This, IsCreature), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithCombatPairTextRevision()
					}
					return NewArtifact("Trigger", "", WithDefinitionAbilityGroup(0, WithAbility(trigger)))
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
					return build(scenario != "missing-marker", scenario == "changed-event" || scenario == "changed-direction"), nil
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
			before := append([]Ability(nil), p.RuntimeAbilities...)
			input := p
			if scenario == "nil-source" {
				input = nil
			}
			change, _ := ReplaceColorWord(core.Red, core.Green)
			if scenario == "invalid-change" {
				change = TextChange{}
			}
			if err := g.RevisePrintedCombatPairTrigger(input, id, change); err == nil {
				t.Fatal("invalid revision accepted")
			}
			if scenario != "missing-source" && len(g.FindPermanent(p.ID()).revisedPrintedTriggers) != 0 {
				t.Fatal("rejected revision changed retention")
			}
			if len(p.RuntimeAbilities) != len(before) {
				t.Fatal("rejected revision changed declarations")
			}
			for i := range before {
				if p.RuntimeAbilities[i] != before[i] {
					t.Fatal("rejected revision changed declaration")
				}
			}
		})
	}
}

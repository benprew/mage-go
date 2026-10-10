package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextUpkeepTriggerDeclaration(t *testing.T) {
	for _, scenario := range []string{"valid", "chosen", "wrong-step", "untyped", "state", "graveyard", "extra-zone", "no-player", "optional", "intervening"} {
		t.Run(scenario, func(t *testing.T) {
			player := BindPlayer("event")
			var pattern EventPattern = TurnStepBegins(YourUpkeep, player)
			if scenario == "chosen" {
				pattern = ChosenPlayerUpkeep(player)
			}
			if scenario == "wrong-step" {
				pattern = TurnStepBegins(YourEndStep, player)
			}
			if scenario == "no-player" {
				pattern = TurnStepBegins(YourUpkeep, nil)
			}
			rule := WhenEvent(pattern, GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithUpkeepTextRevision()
			switch scenario {
			case "untyped":
				rule.eventDefinition = nil
			case "state":
				rule.AsStateTrigger()
			case "graveyard":
				rule.FromGraveyard()
			case "extra-zone":
				rule.InZone(core.ZoneHand)
			case "optional":
				rule.Optional = true
			case "intervening":
				rule.interveningDefinition = PlayerControls(You, IsLand)
			}
			err := ValidateCardDefinition("Declaration", NewArtifact("Declaration", "", WithAbility(rule)))
			if (err == nil) != (scenario == "valid" || scenario == "chosen") {
				t.Fatalf("%s: %v", scenario, err)
			}
		})
	}
}

func TestTextUpkeepTriggerRejectsAtomic(t *testing.T) {
	for _, scenario := range []string{"unmarked", "missing-marker", "changed-player", "changed-choice", "changed-capture", "changed-targets", "changed-bodies", "nil-source", "departed", "returned", "phased", "face-down", "grant", "typed-nil", "unknown-id", "no-builder", "no-group", "invalid-change", "empty-change"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(fresh bool) CardFactory {
				return func() Card {
					player := BindPlayer("event")
					var pattern EventPattern = TurnStepBegins(YourUpkeep, player)
					if fresh && scenario == "changed-player" {
						pattern = TurnStepBegins(EachUpkeep, player)
					}
					if fresh && scenario == "changed-choice" {
						pattern = ChosenPlayerUpkeep(player)
					}
					if fresh && scenario == "changed-capture" {
						pattern = TurnStepBegins(YourUpkeep, nil)
					}
					rule := WhenEvent(pattern, GainLifeFor(You, Fixed(1))).(*GenericTriggered)
					if scenario != "unmarked" && (!fresh || scenario != "missing-marker") {
						rule.WithUpkeepTextRevision()
					}
					if fresh && scenario == "changed-targets" {
						rule.AddTarget(TargetCreature())
					}
					if fresh && scenario == "changed-bodies" {
						rule.AddEffect(GainLifeFor(You, Fixed(2)))
					}
					option := WithAbility(rule)
					if scenario != "no-group" {
						option = WithDefinitionAbilityGroup(0, option)
					}
					return NewArtifact("Trigger", "", option)
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
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := p.RuntimeAbilities[0].AbilityID()
			switch scenario {
			case "departed", "returned":
				g.BouncePermanentToHand(p)
				if scenario == "returned" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "grant":
				g.MutablePermanent(p.ID()).RuntimeAbilities[0] = &grantedByEffect{Ability: p.RuntimeAbilities[0]}
			case "typed-nil":
				g.MutablePermanent(p.ID()).RuntimeAbilities[0] = (*GenericTriggered)(nil)
			case "unknown-id":
				id = uuid.New()
			}
			var before Ability
			if current := g.FindPermanent(p.ID()); current != nil {
				before = current.RuntimeAbilities[0]
			}
			input := p
			if scenario == "nil-source" {
				input = nil
			}
			change, _ := ReplaceColorWord(core.Red, core.Green)
			changes := []TextChange{change}
			if scenario == "invalid-change" {
				changes[0] = TextChange{}
			}
			if scenario == "empty-change" {
				changes = nil
			}
			if err := g.RevisePrintedUpkeepTrigger(input, id, changes...); err == nil {
				t.Fatal("invalid revision accepted")
			}
			if current := g.FindPermanent(p.ID()); current != nil && (current.RuntimeAbilities[0] != before || len(current.revisedPrintedTriggers) != 0) {
				t.Fatal("rejection changed runtime declarations")
			}
		})
	}
}

package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextUpkeepLayerDeclaration(t *testing.T) {
	for _, scenario := range []string{"valid", "other-event", "state", "graveyard", "extra-zone", "missing-revision", "optional", "intervening", "no-player"} {
		t.Run(scenario, func(t *testing.T) {
			trigger := WhenEvent(TurnStepBegins(YourUpkeep, BindPlayer("event player")), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredUpkeepTextRevision()
			switch scenario {
			case "missing-revision":
				trigger.upkeepTextRevision = false
			case "optional":
				trigger.Optional = true
			case "intervening":
				trigger.interveningDefinition = PlayerControls(You, IsLand)
			case "no-player":
				trigger.eventDefinition = turnStepEventPattern{YourUpkeep, nil}
			case "other-event":
				trigger.eventType = core.EvtEndStep
			case "state":
				trigger.AsStateTrigger()
			case "graveyard":
				trigger.FromGraveyard()
			case "extra-zone":
				trigger.InZone(core.ZoneHand)
			}
			card := NewArtifact("Declaration", "", WithAbility(trigger))
			if err := ValidateCardDefinition(card.Name(), card); (err == nil) != (scenario == "valid") {
				t.Fatalf("%s: %v", scenario, err)
			}
		})
	}
}

func TestTextUpkeepLayerRejects(t *testing.T) {
	for _, scenario := range []string{"unmarked", "grant", "face-down", "missing-marker", "changed-event", "changed-context", "missing-source", "nil-source", "unknown-id", "no-builder", "no-group", "invalid-change", "empty-change", "duration", "phased", "typed-nil", "revision-only", "changed-targets", "missing-layer"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(marked, changed bool) CardFactory {
				return func() Card {
					pattern := TurnStepBegins(YourUpkeep, BindPlayer("event player"))
					if changed && scenario == "changed-context" {
						pattern = TurnStepBegins(EachUpkeep, BindPlayer("event player"))
					}
					trigger := WhenEvent(pattern, GainLifeFor(You, Fixed(1))).(*GenericTriggered)
					if marked {
						trigger.WithLayeredUpkeepTextRevision()
						if scenario == "revision-only" || (changed && scenario == "missing-layer") {
							trigger.upkeepTextLayer = false
						}
					}
					if changed && scenario == "changed-targets" {
						trigger.AddTarget(TargetCreature())
					}
					if changed && scenario == "changed-event" {
						trigger.eventType = core.EvtEndStep
					}
					option := WithAbility(trigger)
					if scenario != "no-group" {
						option = WithDefinitionAbilityGroup(0, option)
					}
					return NewArtifact("Trigger", "", option)
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
					return build(scenario != "missing-marker", scenario == "changed-event" || scenario == "changed-context" || scenario == "changed-targets" || scenario == "missing-layer"), nil
				})(card.(*BaseCard))
			}
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			p := g.PutOnBattlefield(card, a.PlayerID())
			id := p.RuntimeAbilities[0].AbilityID()
			switch scenario {
			case "typed-nil":
				g.MutablePermanent(p.ID()).RuntimeAbilities = append(p.RuntimeAbilities, (*GenericTriggered)(nil))
			case "grant":
				g.MutablePermanent(p.ID()).RuntimeAbilities[0] = &grantedByEffect{Ability: p.RuntimeAbilities[0]}
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "missing-source":
				g.BouncePermanentToHand(p)
			case "phased":
				g.PhaseOut(p.ID())
			case "unknown-id":
				id = uuid.New()
			}
			before := append([]Ability(nil), p.RuntimeAbilities...)
			input := p
			if scenario == "nil-source" {
				input = nil
			}
			change, _ := ReplaceColorWord(core.Red, core.Green)
			changes := []TextChange{change}
			duration := core.EndOfTurn
			if scenario == "invalid-change" {
				changes[0] = TextChange{}
			}
			if scenario == "empty-change" {
				changes = nil
			}
			if scenario == "duration" {
				duration = core.EndOfCombat
			}
			effects := len(g.effects.effects)
			if err := g.AddPrintedUpkeepTextEffect(input, id, duration, changes...); err == nil {
				t.Fatal("invalid revision accepted")
			}
			if len(g.effects.effects) != effects || len(p.layeredUpkeepOrigins) != 0 {
				t.Fatal("rejection changed layer state")
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

package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextLandTapLayerDeclaration(t *testing.T) {
	for _, scenario := range []string{"valid", "other-event", "state", "graveyard", "extra-zone"} {
		t.Run(scenario, func(t *testing.T) {
			trigger := WhenEvent(OpponentLandBecomesTapped().OfSubtype("Forest"), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLayeredLandTapTextRevision()
			switch scenario {
			case "other-event":
				trigger.eventType = core.EvtUpkeep
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

func TestTextLandTapLayerRejects(t *testing.T) {
	for _, scenario := range []string{"unmarked", "grant", "face-down", "missing-marker", "changed-event", "changed-capture", "changed-filter", "direct-marker", "missing-source", "nil-source", "unknown-id", "no-builder", "no-group", "invalid-change", "empty-change", "duration", "phased"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(marked, changed bool) CardFactory {
				return func() Card {
					pattern := OpponentLandBecomesTapped().OfSubtype("Forest")
					if changed && scenario == "changed-capture" {
						pattern = pattern.WithLand(BindPermanent("new capture"))
					}
					if changed && scenario == "changed-filter" {
						pattern = OpponentLandBecomesTapped()
					}
					trigger := WhenEvent(pattern, GainLifeFor(You, Fixed(1))).(*GenericTriggered)
					if marked {
						if changed && scenario == "direct-marker" {
							trigger.WithLandTapTextRevision()
						} else {
							trigger.WithLayeredLandTapTextRevision()
						}
					}
					if changed && scenario == "changed-event" {
						trigger.eventType = core.EvtUpkeep
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
					return build(scenario != "missing-marker", scenario == "changed-event" || scenario == "changed-capture" || scenario == "changed-filter" || scenario == "direct-marker"), nil
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
			if err := g.AddPrintedLandTapTextEffect(input, id, duration, changes...); err == nil {
				t.Fatal("invalid revision accepted")
			}
			if len(g.effects.effects) != effects || len(p.layeredLandTapOrigins) != 0 {
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

func TestTextLandTapLayerBindings(t *testing.T) {
	build := func(subtype string) CardFactory {
		return func() Card {
			land := BindPermanent("event land")
			trigger := WhenEvent(OpponentLandBecomesTapped().OfSubtype(subtype).WithLand(land), AddCountersTo(From(land), core.Charge, Fixed(1))).(*GenericTriggered).WithLayeredLandTapTextRevision()
			return NewEnchantment("Bound trigger", "", WithDefinitionAbilityGroup(0, WithAbility(trigger)))
		}
	}
	plan, err := NewCardDefinitionPlan(build("Forest"))
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build("Island"), nil })(card.(*BaseCard))
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(card, a.PlayerID())
	forest := g.PutOnBattlefield(NewLand("Forest", WithSubTypes("Forest")), b.PlayerID())
	islandCard := NewLand("Island", WithSubTypes("Island"))
	island := g.PutOnBattlefield(islandCard, b.PlayerID())
	g.TapPermanent(forest)
	change, _ := ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.AddPrintedLandTapTextEffect(source, source.RuntimeAbilities[0].AbilityID(), core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	g.TapPermanent(island)
	g.PutTriggersOnStack()
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if g.StackSize() != 2 {
		t.Fatal("event bindings lost")
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.BouncePermanentToHand(island)
	b.RemoveFromHand(islandCard.ID())
	returned := g.PutOnBattlefield(islandCard, b.PlayerID())
	clone := g.Clone()
	g.ResolveStack()
	if g.FindPermanent(forest.ID()).Counters[core.Charge] != 1 || g.FindPermanent(returned.ID()).Counters[core.Charge] != 0 {
		t.Fatalf("binding followed another event or returned land: forest=%d returned=%d", g.FindPermanent(forest.ID()).Counters[core.Charge], g.FindPermanent(returned.ID()).Counters[core.Charge])
	}
	if clone.FindPermanent(forest.ID()).Counters[core.Charge] != 0 {
		t.Fatal("clone shared bindings")
	}
	clone.ResolveStack()
	if clone.FindPermanent(forest.ID()).Counters[core.Charge] != 1 || clone.FindPermanent(returned.ID()).Counters[core.Charge] != 0 {
		t.Fatal("clone lost exact bindings")
	}
}

package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextLandTapTriggerDeclaration(t *testing.T) {
	for _, scenario := range []string{"valid", "other-event", "state", "graveyard", "extra-zone"} {
		t.Run(scenario, func(t *testing.T) {
			trigger := WhenEvent(OpponentLandBecomesTapped().OfSubtype("Forest"), GainLifeFor(You, Fixed(1))).(*GenericTriggered).WithLandTapTextRevision()
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

func TestTextLandTapTriggerRejects(t *testing.T) {
	for _, scenario := range []string{"unmarked", "grant", "face-down", "missing-marker", "changed-event", "missing-source", "nil-source", "unknown-id", "no-builder"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(marked, changed bool) CardFactory {
				return func() Card {
					trigger := WhenEvent(OpponentLandBecomesTapped().OfSubtype("Forest"), GainLifeFor(You, Fixed(1))).(*GenericTriggered)
					if marked {
						trigger.WithLandTapTextRevision()
					}
					if changed {
						trigger.eventType = core.EvtUpkeep
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
					return build(scenario != "missing-marker", scenario == "changed-event"), nil
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
			if err := g.RevisePrintedLandTapTrigger(input, id, change); err == nil {
				t.Fatal("invalid revision accepted")
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

func TestTextLandTapTriggerBindings(t *testing.T) {
	build := func(subtype string) CardFactory {
		return func() Card {
			land := BindPermanent("event land")
			trigger := WhenEvent(OpponentLandBecomesTapped().OfSubtype(subtype).WithLand(land), AddCountersTo(From(land), core.Charge, Fixed(1))).(*GenericTriggered).WithLandTapTextRevision()
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
	if err := g.RevisePrintedLandTapTrigger(source, source.RuntimeAbilities[0].AbilityID(), change); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	g.TapPermanent(island)
	g.PutTriggersOnStack()
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

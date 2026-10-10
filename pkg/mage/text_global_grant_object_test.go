package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestTextGlobalGrantObjectAtomic(t *testing.T) {
	for _, scenario := range []string{"valid", "missing-second-runtime", "missing-second-registration", "changed-kind", "changed-grant-form", "changed-keyword", "changed-group", "missing-builder", "missing-group", "empty", "invalid", "duration", "phased", "face-down", "returned"} {
		t.Run(scenario, func(t *testing.T) {
			build := func(changed bool) CardFactory {
				return func() Card {
					first := StaticAbility(GrantKeywordToAllIncludingSource(core.Swampwalk, IsCreature)).WithLayeredGrantTextRevision()
					second := Static(GrantEachActivatedAbility(MatchingPermanents(IsCreature), Costs(), Regenerate(This))).WithLayeredGrantTextRevision()
					if changed && scenario == "changed-kind" {
						second = StaticAbility(BoostAllCreaturesIncludingSelf(1, 1, IsCreature)).WithLayeredPTTextRevision()
					}
					if changed && scenario == "changed-grant-form" {
						second = StaticAbility(GrantKeywordToAllIncludingSource(core.Flying, IsCreature)).WithLayeredGrantTextRevision()
					}
					group := 1
					if changed && scenario == "changed-group" {
						group = 2
					}
					options := []CardOption{WithDefinitionAbilityGroup(0, WithAbility(first)), WithDefinitionAbilityGroup(group, WithAbility(second)), WithKeyword(core.FirstStrike)}
					if scenario == "missing-group" {
						options[1] = WithAbility(second)
					}
					if changed && scenario == "changed-keyword" {
						options = append(options, WithKeyword(core.Flying))
					}
					return NewCreature("Grant", "", 2, 3, options...)
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
			if scenario != "missing-builder" {
				WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build(true), nil })(card.(*BaseCard))
			}
			g, a, _ := newSpellCopyGame()
			p := g.PutOnBattlefield(card, a.PlayerID())
			switch scenario {
			case "missing-second-runtime":
				g.MutablePermanent(p.ID()).RuntimeAbilities = p.RuntimeAbilities[:1]
			case "missing-second-registration":
				id := card.Abilities()[1].AbilityID()
				for i, e := range g.effects.effects {
					if g.printedStaticRegistration(e, p, id) {
						g.effects.effects = append(g.effects.effects[:i], g.effects.effects[i+1:]...)
						break
					}
				}
			case "phased":
				g.PhaseOut(p.ID())
			case "face-down":
				g.MutablePermanent(p.ID()).FaceDown = true
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID())
			}
			change, _ := ReplaceBasicLandWord(core.SwampLandType, core.ForestLandType)
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
			before := len(g.effects.effects)
			snapshot := g.FindPermanentIncludingPhased(p.ID())
			err = g.AddPrintedGlobalGrantTextEffect(p, duration, changes...)
			if scenario == "valid" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("invalid complete grant context accepted")
			}
			if len(g.effects.effects) != before || g.FindPermanentIncludingPhased(p.ID()) != snapshot {
				t.Fatal("failed preflight changed live state")
			}
		})
	}
}

func TestTextGlobalGrantObjectActivationIdentity(t *testing.T) {
	build := func() Card {
		return NewEnchantment("Grant", "", WithDefinitionAbilityGroup(0, WithAbility(Static(GrantEachActivatedAbility(MatchingPermanents(IsCreature), Costs(), GainLifeFor(You, Fixed(1)), ActivationLimit(1))).WithLayeredGrantTextRevision())))
	}
	plan, err := NewCardDefinitionPlan(build)
	if err != nil {
		t.Fatal(err)
	}
	card, err := plan.Build()
	if err != nil {
		t.Fatal(err)
	}
	WithTextDefinitionBuilder(func([]TextChange) (CardFactory, error) { return build, nil })(card.(*BaseCard))
	g, a, b := newSpellCopyGame()
	source := g.PutOnBattlefield(card, a.PlayerID())
	p := g.PutOnBattlefield(NewCreature("Recipient", "", 1, 2), b.PlayerID())
	id := g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID()
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	change, _ := ReplaceColorWord(core.Red, core.Blue)
	if err := g.AddPrintedGlobalGrantTextEffect(source, core.EndOfTurn, change); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		g.ApplyContinuousEffects()
		if g.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID() != id {
			t.Fatal("text pass changed granted identity")
		}
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("text change reset recipient activation limit")
	}
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.ResolveStack()
	clone.ResolveStack()
	if b.Life() != 21 || a.Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 21 {
		t.Fatal("pending body lost recipient controller")
	}
	clone.GetEffects().RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if clone.FindPermanent(p.ID()).RuntimeAbilities[0].AbilityID() != id {
		t.Fatal("expiry changed granted identity")
	}
	if err := clone.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, nil); err == nil {
		t.Fatal("expiry reset recipient limit")
	}
}

package dsl_test

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

// --- from artifact_boost_migration_test.go ---
func TestArtifactBoostTypedDefinitions(t *testing.T) {
	for _, name := range []string{"Elven Lyre", "Coral Helm"} {
		t.Run(name, func(t *testing.T) {
			card, err := mage.CreateCard(name)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
			ability := mage.UnwrapAbility(card.Abilities()[0]).(mage.ActivatedAbility)
			if ability.SorcerySpeed() || len(ability.Targets()) != 1 {
				t.Fatal("expected one target and instant timing")
			}
			if _, ok := ability.Targets()[0].(mage.TargetBinding); !ok {
				t.Fatal("missing typed target")
			}
			metadata := mage.InspectDefinition(ability.Effects()[0])
			if metadata.Role != "until" {
				t.Fatalf("effect role = %q, want until", metadata.Role)
			}
			props := metadata.Properties
			if props.Mass || props.Outcome != mage.OutcomeBenefit || props.PowerBoost != 2 || props.ToughnessBoost != 2 {
				t.Fatalf("boost metadata = %+v", props)
			}
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			want := make([]uuid.UUID, 0, 2)
			for _, player := range []*mage.BasePlayer{a, b} {
				p := g.PutOnBattlefield(mage.NewCreature("Creature", "", 2, 2), player.PlayerID())
				want = append(want, p.ID())
				g.PutOnBattlefield(mage.NewArtifact("Artifact", ""), player.PlayerID())
			}
			got := ability.Targets()[0].Possible(a.PlayerID(), card, g)
			if len(got) != len(want) {
				t.Fatalf("targets = %v, want %v", got, want)
			}
			for _, id := range want {
				if !slices.Contains(got, id) {
					t.Fatalf("missing target %v", id)
				}
			}
		})
	}
}

// --- from creature_entry_keyword_test.go ---
func TestCreatureEntryKeywordPublicDeclaration(t *testing.T) {
	card := dsl.NewCreature("Flying watcher", "", 1, 1, dsl.WithAbility(
		dsl.WhenEvent(dsl.AnotherCreatureYouControlEnters().HasKeyword(core.Flying),
			dsl.GainLife(dsl.You, dsl.Fixed(1)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from entry_cast_colors_test.go ---
func TestEntryCastColorsPublicDSL(t *testing.T) {
	card := dsl.NewCreature("Entry", "{3}", 2, 2, dsl.WithAbility(
		dsl.EntersWithCounters(core.P1P1, dsl.ColorsSpentToCastEnteringPermanent())))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from global_reduction_migration_test.go ---
func TestGlobalReductionTypedDefinitions(t *testing.T) {
	for _, tc := range []struct {
		name             string
		power, toughness int
	}{{"Hell Swarm", -1, 0}, {"Marsh Gas", -2, 0}, {"Bone Flute", -1, 0}, {"Languish", -4, -4}} {
		t.Run(tc.name, func(t *testing.T) {
			card, err := mage.CreateCard(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
				t.Fatal(err)
			}
			ability := mage.UnwrapAbility(card.Abilities()[0])
			var effects []mage.Effect
			switch a := ability.(type) {
			case *mage.SpellAbility:
				if len(a.Targets()) != 0 {
					t.Fatal("collection spell must not target")
				}
				effects = a.Effects()
			case mage.ActivatedAbility:
				if a.SorcerySpeed() || len(a.Targets()) != 0 {
					t.Fatal("expected no targets and instant timing")
				}
				effects = a.Effects()
			default:
				t.Fatalf("unexpected ability %T", ability)
			}
			info := mage.InspectDefinition(effects[0])
			if info.Role != "until" {
				t.Fatalf("effect role = %q, want until", info.Role)
			}
			props := info.Properties
			if !props.Mass || props.Outcome != mage.OutcomeDetriment || props.PowerBoost != tc.power || props.ToughnessBoost != tc.toughness {
				t.Fatalf("reduction metadata = %+v", props)
			}
		})
	}
}

// --- from granted_collection_test.go ---
func TestGrantedCollectionComposition(t *testing.T) {
	target := dsl.TargetPermanent("target", dsl.IsCreature)
	set := dsl.ControlledPermanents(dsl.IsArtifact)
	for _, mod := range []dsl.ContinuousModification{
		dsl.EachActivatedAbilityModification(set, dsl.Costs(), dsl.Regenerate(target), dsl.WithTargets(target), dsl.ActivationLimit(1)),
		dsl.EachActivatedModesModification(set, dsl.Costs(), dsl.ModeOf("regenerate", dsl.Regenerate(target), dsl.WithTargets(target)), dsl.ModeOf("gain", dsl.GainLife(dsl.You, dsl.Fixed(1))), dsl.OnlyAsSorcery()),
	} {
		card := dsl.NewArtifact("Grant", "", dsl.WithAbility(dsl.Activate(dsl.Costs(), dsl.Until(core.EndOfTurn, mod))), dsl.WithAbility(dsl.Static(dsl.While(dsl.Controls(dsl.IsArtifact), mod))))
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		effect := dsl.Until(core.EndOfTurn, mod)
		if !effect.Properties().Mass || effect.Properties().Outcome != mage.OutcomeBenefit {
			t.Fatal("missing mass benefit metadata")
		}
	}
	outer := dsl.BindPermanent("outer")
	invalid := dsl.NewInstant("Invalid", "", dsl.WithAbility(dsl.Spell(dsl.Sequence(dsl.Snapshot(dsl.This, outer), dsl.Until(core.EndOfTurn, dsl.EachActivatedAbilityModification(set, dsl.Costs(), dsl.Regenerate(dsl.From(outer))))))))
	if mage.ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("grant read outer binding")
	}
}

// --- from keyword_removal_test.go ---
func TestTypedKeywordRemovalComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	body := dsl.Until(dsl.EndOfTurn, dsl.RemoveKeyword(target, dsl.Forestwalk))
	cards := []mage.Card{
		dsl.NewInstant("Spell", "", dsl.Spell(body, dsl.WithTargets(target))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.TapCost(dsl.This)), body, dsl.WithTargets(target)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body, dsl.WithTargets(target)))),
		dsl.NewCreature("Static", "", 1, 1, dsl.WithAbility(dsl.Static(dsl.While(dsl.Controls(dsl.IsCreature), dsl.RemoveKeyword(dsl.This, dsl.Forestwalk))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTypedLandwalkRemovalComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	body := dsl.Until(dsl.EndOfTurn, dsl.RemoveLandwalk(target))
	cards := []mage.Card{
		dsl.NewInstant("Spell", "", dsl.Spell(body, dsl.WithTargets(target))),
		dsl.NewArtifact("Activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.TapCost(dsl.This)), body, dsl.WithTargets(target)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body, dsl.WithTargets(target)))),
		dsl.NewCreature("Static", "", 1, 1, dsl.WithAbility(dsl.Static(dsl.While(dsl.Controls(dsl.IsCreature), dsl.RemoveLandwalk(dsl.This))))),
	}
	for _, card := range cards {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from self_cast_colors_test.go ---
func TestSelfCastColorsDSL(t *testing.T) {
	card := dsl.NewInstant("Converge", "", dsl.Spell(
		dsl.Draw(dsl.You, dsl.ColorsSpentToCastThisSpell())))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from self_entry_colors_test.go ---
func TestSelfEntryColorsSpentDSL(t *testing.T) {
	colors := dsl.BindNumber("entry colors")
	card := dsl.NewCreature("Entry colors", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(
		dsl.Enters(dsl.This).WithColorsSpent(colors),
		dsl.GainLife(dsl.You, dsl.FromNumber(colors)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from shield_wall_migration_test.go ---
func TestShieldWallTypedDefinition(t *testing.T) {
	card, err := mage.CreateCard("Shield Wall")
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Shield Wall", card); err != nil {
		t.Fatal(err)
	}
	spell, ok := mage.UnwrapAbility(card.Abilities()[0]).(*mage.SpellAbility)
	if !ok {
		t.Fatalf("ability = %T, want spell", card.Abilities()[0])
	}
	if len(spell.Targets()) != 0 {
		t.Fatal("collection spell must not target")
	}
	info := mage.InspectDefinition(spell.Effects()[0])
	if info.Role != "until" {
		t.Fatalf("effect role = %q, want until", info.Role)
	}
	props := info.Properties
	if !props.Mass || props.Outcome != mage.OutcomeBenefit || props.PowerBoost != 0 || props.ToughnessBoost != 2 {
		t.Fatalf("boost metadata = %+v", props)
	}
}

// --- from skip_untap_test.go ---
func TestSkipNextUntapComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	saved := dsl.BindPermanent("saved")
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.SkipNextUntap(dsl.From(saved)))
	for _, card := range []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(dsl.SkipNextUntap(target), dsl.WithTargets(target)))),
		dsl.NewArtifact("Barl's Cage", "{4}", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.Mana("{3}")), dsl.SkipNextUntap(target), dsl.WithTargets(target)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewInstant("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Skip", dsl.SkipNextUntap(target), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewInstant("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, dsl.SkipNextUntap(dsl.From(saved)))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}
func TestCapturedUntapComposition(t *testing.T) {
	target := dsl.TargetPermanent("creature", dsl.IsCreature)
	saved := dsl.BindPermanent("saved")
	body := dsl.Sequence(dsl.Snapshot(dsl.This, saved), dsl.SkipNextUntapOf(dsl.From(saved), dsl.You))
	for _, card := range []mage.Card{
		dsl.NewInstant("Spell", "", dsl.WithAbility(dsl.Spell(dsl.SkipNextUntapOf(target, dsl.You), dsl.WithTargets(target)))),
		dsl.NewArtifact("Barl's Cage", "{4}", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.Mana("{3}")), dsl.SkipNextUntapOf(target, dsl.You), dsl.WithTargets(target)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), body))),
		dsl.NewInstant("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Skip", dsl.SkipNextUntapOf(target, dsl.You), dsl.WithTargets(target)), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewInstant("Loop", "", dsl.WithAbility(dsl.Spell(dsl.ForEach(dsl.MatchingPermanents(dsl.IsCreature), saved, dsl.SkipNextUntapOf(dsl.From(saved), dsl.You))))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestHomaridShroudActivationComposition(t *testing.T) {
	card := dsl.NewCreature("Homarid Warrior", "{4}{U}", 3, 3,
		dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.Mana("{U}")),
			dsl.Sequence(
				dsl.Until(core.EndOfTurn, dsl.GrantKeyword(dsl.This, core.Shroud)),
				dsl.SkipNextUntapOf(dsl.This, dsl.You),
				dsl.Tap(dsl.This),
			),
		)),
	)
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

// --- from spell_cast_colors_test.go ---
func TestSpellCastColorsSpentDSL(t *testing.T) {
	colors := dsl.BindNumber("colors")
	pattern := dsl.SpellCast().OfType(core.TypeInstant).WithColorsSpent(colors)
	card := dsl.NewArtifact("Watcher", "", dsl.WithAbility(dsl.WhenEvent(pattern,
		dsl.GainLife(dsl.You, dsl.FromNumber(colors)))))
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if got := dsl.InspectDefinition(pattern); got.Role != "event-pattern" || len(got.Children) != 1 {
		t.Fatalf("inspection = %#v", got)
	}
}

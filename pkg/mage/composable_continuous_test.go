package mage

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// --- from composable_ability_removal_test.go ---
func TestRemoveAbilitiesTemporaryCapturesSetAndLayerOrder(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Source", "")
	g.PutOnBattlefield(source, a.PlayerID())
	first := NewCreature("First", "", 2, 2, WithKeyword(Flying))
	g.PutOnBattlefield(first, a.PlayerID())
	if err := ApplyEffect(g, Until(EndOfTurn, GrantKeywordModification(This, CantRegenerate)), first.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if err := ApplyEffect(g, Until(EndOfTurn, RemoveAbilitiesEach(MatchingPermanents(IsCreature))), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(first.ID()).HasKeyword(Flying) || !g.FindPermanent(first.ID()).HasKeyword(CantRegenerate) {
		t.Fatal("removal must remove abilities and preserve rules restrictions")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PhaseIn(g.PhaseOut(first.ID()))
	g.ApplyContinuousEffects()
	if g.FindPermanent(first.ID()).HasKeyword(Flying) {
		t.Fatal("source departure and phasing must preserve temporary removal")
	}
	second := NewCreature("Second", "", 2, 2, WithKeyword(Flying))
	g.PutOnBattlefield(second, b.PlayerID())
	if !g.FindPermanent(second.ID()).HasKeyword(Flying) {
		t.Fatal("the resolving set must not include a later permanent")
	}
	if err := ApplyEffect(g, Until(EndOfTurn, GrantKeywordModification(This, Vigilance)), first.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(first.ID()).HasKeyword(Vigilance) || g.FindPermanent(first.ID()).HasKeyword(Flying) {
		t.Fatal("a later grant must restore only its own ability")
	}
	clone := g.Clone()
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if !clone.FindPermanent(first.ID()).HasKeyword(Flying) || g.FindPermanent(first.ID()).HasKeyword(Flying) {
		t.Fatal("expiry must be isolated between clones")
	}
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	g.PutOnBattlefield(first, a.PlayerID())
	g.ApplyContinuousEffects()
	if !g.FindPermanent(first.ID()).HasKeyword(Flying) {
		t.Fatal("the old effect must not remove abilities from a returned permanent")
	}
}

func TestRemoveAbilitiesStaticSetUpdates(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Source", "", WithAbility(Static(While(Controls(IsArtifact), RemoveAbilitiesEach(MatchingPermanents(IsCreature))))))
	g.PutOnBattlefield(source, a.PlayerID())
	first := NewCreature("First", "", 2, 2, WithKeyword(Flying))
	g.PutOnBattlefield(first, a.PlayerID())
	second := NewCreature("Second", "", 2, 2, WithKeyword(Vigilance))
	g.PutOnBattlefield(second, b.PlayerID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(first.ID()).HasKeyword(Flying) || g.FindPermanent(second.ID()).HasKeyword(Vigilance) {
		t.Fatal("the static set must include later creatures")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.ApplyContinuousEffects()
	if !g.FindPermanent(first.ID()).HasKeyword(Flying) || !g.FindPermanent(second.ID()).HasKeyword(Vigilance) {
		t.Fatal("source departure must end the static removal")
	}
}

func TestRemoveAbilitiesValidationAndMetadata(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	body := Until(EndOfTurn, RemoveAbilities(target))
	info := InspectDefinition(body)
	if info.Properties.Outcome != OutcomeDetriment || info.Properties.GrantedKeyword != 0 || !strings.Contains(info.Text, "remove all abilities") {
		t.Fatalf("removal metadata = %+v", info)
	}
	if err := ValidateCardDefinition("Unbound", NewInstant("Unbound", "", WithAbility(Spell(body)))); err == nil || !strings.Contains(err.Error(), "target binding requires a declaration") {
		t.Fatalf("undeclared target error = %v", err)
	}
	if err := ValidateCardDefinition("Nil", NewInstant("Nil", "", WithAbility(Spell(Until(EndOfTurn, RemoveAbilities(nil)))))); err == nil || !strings.Contains(err.Error(), "supply a non-nil expression") {
		t.Fatalf("nil target error = %v", err)
	}
	if err := ValidateCardDefinition("Valid", NewInstant("Valid", "", WithAbility(Spell(body, WithTargets(target))))); err != nil {
		t.Fatal(err)
	}
}

// --- from composable_base_pt_test.go ---
func TestTypedBasePowerToughness(t *testing.T) {
	for _, change := range []string{"none", "phase", "return", "source leaves", "control", "overwrite"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			card := NewCreature("Subject", "", 4, 5)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			p.AddCounter(P1P1, 1)
			saved := BindPermanent("subject")
			body := Sequence(ChooseRandomPermanent(MatchingPermanents(IsCreature), saved), Until(EndOfTurn,
				ModifyPT(From(saved), Fixed(3), Fixed(3)),
				SetBasePowerToughness(From(saved), Fixed(0), Fixed(2))))
			if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check := func(game *Game, power, toughness int) {
				t.Helper()
				game.ApplyContinuousEffects()
				got := game.FindPermanent(card.ID())
				if got.CurrentPower(game) != power || got.CurrentToughness(game) != toughness {
					t.Fatalf("P/T = %d/%d, want %d/%d", got.CurrentPower(game), got.CurrentToughness(game), power, toughness)
				}
			}
			check(g, 4, 6)
			clone := g.Clone()
			clone.effects.RemoveEndOfTurn()
			check(clone, 5, 6)
			check(g, 4, 6)
			switch change {
			case "phase":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
				check(g, 4, 5)
				return
			case "source leaves":
				g.RemoveFromBattlefield(source)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "overwrite":
				if err := ApplyEffect(g, Until(EndOfTurn, SetBasePowerToughness(This, Fixed(7), Fixed(8))), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				check(g, 11, 12)
				return
			}
			check(g, 4, 6)
		})
	}
}

func TestTypedBasePower(t *testing.T) {
	for _, change := range []string{"none", "phase", "return", "source leaves", "control", "overwrite"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			card := NewCreature("Subject", "", 4, 5)
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			p.AddCounter(P1P1, 1)
			saved := BindPermanent("subject")
			body := Sequence(ChooseRandomPermanent(MatchingPermanents(IsCreature), saved), Until(EndOfTurn,
				ModifyPT(From(saved), Fixed(3), Fixed(3)),
				SetBasePower(From(saved), Fixed(0))))
			if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			check := func(game *Game, power, toughness int) {
				t.Helper()
				game.ApplyContinuousEffects()
				got := game.FindPermanent(card.ID())
				if got.CurrentPower(game) != power || got.CurrentToughness(game) != toughness {
					t.Fatalf("P/T = %d/%d, want %d/%d", got.CurrentPower(game), got.CurrentToughness(game), power, toughness)
				}
			}
			check(g, 4, 9)
			clone := g.Clone()
			clone.effects.RemoveEndOfTurn()
			check(clone, 5, 6)
			check(g, 4, 9)
			switch change {
			case "phase":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
				check(g, 4, 5)
				return
			case "source leaves":
				g.RemoveFromBattlefield(source)
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: source.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
			case "overwrite":
				if err := ApplyEffect(g, Until(EndOfTurn, SetBasePowerToughness(This, Fixed(7), Fixed(8))), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				check(g, 11, 12)
				return
			}
			check(g, 4, 9)
		})
	}
}

func TestTypedOtherThanSourceTargets(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Same name", "", 1, 1)
	g.PutOnBattlefield(source, a.PlayerID())
	own := g.PutOnBattlefield(NewCreature("Same name", "", 2, 2), a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 3, 3), b.PlayerID())
	target := DeclarePermanentTarget("creature", IsCreature, TargetOtherThanSource())
	choices := target.Possible(a.PlayerID(), source, g)
	if len(choices) != 2 || !slices.Contains(choices, own.ID()) || !slices.Contains(choices, other.ID()) {
		t.Fatalf("choices = %v", choices)
	}
	g.PhaseOut(other.ID())
	choices = target.Possible(a.PlayerID(), source, g)
	if len(choices) != 1 || choices[0] != own.ID() {
		t.Fatalf("choices = %v", choices)
	}
}

func TestTypedBasePowerPreservesEarlierToughness(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewCreature("Subject", "", 4, 5), a.PlayerID())
	for _, body := range []ResolvingEffect{
		Until(EndOfTurn, SetBasePowerToughness(This, Fixed(7), Fixed(8))),
		Until(Indefinite, SetBasePower(This, Fixed(-1))),
	} {
		if err := ApplyEffect(g, body, p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	g.ApplyContinuousEffects()
	p = g.FindPermanent(p.ID())
	if p.CurrentPower(g) != -1 || p.CurrentToughness(g) != 8 {
		t.Fatalf("P/T = %d/%d, want -1/8", p.CurrentPower(g), p.CurrentToughness(g))
	}
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	p = g.FindPermanent(p.ID())
	if p.CurrentPower(g) != -1 || p.CurrentToughness(g) != 5 {
		t.Fatalf("P/T = %d/%d, want -1/5", p.CurrentPower(g), p.CurrentToughness(g))
	}
}

// --- from composable_continuous_test.go ---
func TestComposableContinuousCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := NewCreature("First", "", 2, 2)
	first.SetOwner(a.PlayerID())
	g.PutOnBattlefield(first, a.PlayerID())
	effect := Until(EndOfTurn, ModifyEachPT(ControlledPermanents(IsCreature), Fixed(2), Fixed(1)))
	if err := ApplyEffect(g, effect, first.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	later := NewCreature("Later", "", 2, 2)
	later.SetOwner(a.PlayerID())
	g.PutOnBattlefield(later, a.PlayerID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(first.ID()).CurrentPower(g) != 4 || g.FindPermanent(later.ID()).CurrentPower(g) != 2 {
		t.Fatal("temporary membership was not captured")
	}
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	g.PutOnBattlefield(first, a.PlayerID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(first.ID()).CurrentPower(g) != 2 {
		t.Fatal("temporary effect followed a returned card")
	}
}

func TestComposableStaticMembership(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	lord := NewCreature("Lord", "", 2, 2, WithAbility(Static(While(Controls(IsCreature), ModifyEachPT(ControlledPermanents(IsCreature), Fixed(1), Fixed(1))))))
	lord.SetOwner(a.PlayerID())
	g.PutOnBattlefield(lord, a.PlayerID())
	later := NewCreature("Later", "", 2, 2)
	later.SetOwner(a.PlayerID())
	g.PutOnBattlefield(later, a.PlayerID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(later.ID()).CurrentPower(g) != 3 {
		t.Fatal("static membership did not include later entry")
	}
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(lord.ID()))
	clone.ApplyContinuousEffects()
	if clone.FindPermanent(later.ID()).CurrentPower(clone) != 2 || g.FindPermanent(later.ID()).CurrentPower(g) != 3 {
		t.Fatal("static clone isolation failed")
	}
}

func TestComposableSelfPumpLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Source", "", 2, 3)
	card.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	body := Until(EndOfTurn, ModifyPT(This, Fixed(1), Fixed(1)))
	for range 2 {
		if err := ApplyEffect(g, body, p.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	check := func(game *Game, power, toughness int) {
		t.Helper()
		game.ApplyContinuousEffects()
		got := game.FindPermanent(card.ID())
		if got.CurrentPower(game) != power || got.CurrentToughness(game) != toughness {
			t.Fatalf("P/T = %d/%d, want %d/%d", got.CurrentPower(game), got.CurrentToughness(game), power, toughness)
		}
	}
	check(g, 4, 5)
	clone := g.Clone()
	clone.effects.RemoveEndOfTurn()
	check(clone, 2, 3)
	check(g, 4, 5)
	g.PhaseIn(g.PhaseOut(p.ID()))
	check(g, 4, 5)
	g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
	g.PutOnBattlefield(card, a.PlayerID())
	check(g, 2, 3)
}

// --- from composable_creature_entry_keyword_test.go ---
func TestCreatureEntryKeywordFilter(t *testing.T) {
	for _, tc := range []struct {
		name         string
		flying       bool
		opponent     bool
		phased       bool
		sourceReturn bool
		want         int
	}{
		{"controlled flyer", true, false, false, false, 1},
		{"controlled ground creature", false, false, false, false, 0},
		{"opposing flyer", true, true, false, false, 0},
		{"phased watcher", true, false, true, false, 0},
		{"watcher leaves and returns", true, false, false, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().HasKeyword(Flying),
				GainLifeFor(You, Fixed(1)))))
			watcher.SetOwner(a.PlayerID())
			g.PutOnBattlefield(watcher, a.PlayerID())
			if tc.phased {
				g.PhaseOut(watcher.ID())
			}
			var options []CardOption
			if tc.flying {
				options = append(options, WithKeyword(Flying))
			}
			entered := NewCreature("Entered", "", 2, 2, options...)
			entered.SetOwner(b.PlayerID())
			controller := a.PlayerID()
			if tc.opponent {
				controller = b.PlayerID()
			}
			g.PutOnBattlefield(entered, controller)
			g.PutTriggersOnStack()
			if tc.sourceReturn {
				g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
				g.PutOnBattlefield(watcher, b.PlayerID())
			}
			g.RemoveFromBattlefield(g.FindPermanent(entered.ID()))
			g.PutOnBattlefield(entered, b.PlayerID())
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original life")
			}
			g.ResolveStack()
			if got := a.Life() - 20; got != tc.want {
				t.Fatalf("life gained=%d, want %d", got, tc.want)
			}
		})
	}
}

func TestCreatureEntryKeywordDeclaration(t *testing.T) {
	valid := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(
		AnotherCreatureYouControlEnters().HasKeyword(Flying).Nontoken(),
		GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []creatureEntersPattern{
		AnotherCreatureYouControlEnters().HasKeyword(Attr(0)),
		AnotherCreatureYouControlEnters().HasKeyword(Flying).HasKeyword(Vigilance),
	} {
		invalid := NewCreature("Invalid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil {
			t.Fatal("accepted invalid entry keyword filter")
		}
	}
}

func TestCreatureEntryKeywordUsesEffectiveEntryState(t *testing.T) {
	for _, tc := range []struct {
		name    string
		printed bool
		remove  bool
		want    int
	}{
		{"static grant", false, false, 1},
		{"static removal", true, true, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
				AnotherCreatureYouControlEnters().HasKeyword(Flying), GainLifeFor(You, Fixed(1)))))
			g.PutOnBattlefield(watcher, a.PlayerID())
			modifier := GrantEachKeyword(ControlledPermanents(IsCreature), Flying)
			if tc.remove {
				modifier = RemoveAbilitiesEach(MatchingPermanents(HasSubType("Bird")))
			}
			donor := NewEnchantment("Donor", "", WithAbility(Static(While(Controls(IsCreature), modifier))))
			g.PutOnBattlefield(donor, a.PlayerID())
			var options []CardOption
			if tc.printed {
				options = append(options, WithKeyword(Flying))
			}
			options = append(options, WithSubTypes("Bird"))
			g.PutOnBattlefield(NewCreature("Entered", "", 2, 2, options...), a.PlayerID())
			g.PutTriggersOnStack()
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original life")
			}
			g.ResolveStack()
			if got := a.Life() - 20; got != tc.want {
				t.Fatalf("life gained=%d, want %d", got, tc.want)
			}
		})
	}
}

// --- from composable_entry_cast_colors_test.go ---
func TestEntryCastColorsCounters(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(PrecombatMain)
	card := NewCreature("Converge", "{3}", 2, 2, WithAbility(
		EntersWithCounters(P1P1, ColorsSpentToCastEnteringPermanent())))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(Red, 1)
	a.ManaPool().Add(Blue, 1)
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err == nil {
		t.Fatal("unpaid cast succeeded")
	}
	if g.StackSize() != 0 || g.FindPermanent(card.ID()) != nil {
		t.Fatal("failed payment changed entry state")
	}
	a.ManaPool().Add(Colorless, 1)
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*Game{g, clone} {
		p := game.FindPermanent(card.ID())
		if p == nil || p.Counters[P1P1] != 2 {
			t.Fatalf("cast entry counters = %v", p)
		}
	}
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	returned := g.PutOnBattlefield(card, b.PlayerID())
	if returned.Counters[P1P1] != 0 {
		t.Fatalf("uncast entry counters = %d", returned.Counters[P1P1])
	}
}

func TestEntryCastColorsDeclaration(t *testing.T) {
	valid := NewCreature("Valid", "", 1, 1, WithAbility(
		EntersWithCounters(P1P1, ColorsSpentToCastEnteringPermanent())))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	if got := InspectDefinition(valid.Abilities()[0]).Children; len(got) == 0 {
		t.Fatal("entry replacement missing from inspection")
	}
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"nil", NewCreature("Nil", "", 1, 1, WithAbility(EntersWithCounters(P1P1, nil))), "entry amount"},
		{"counter", NewCreature("Counter", "", 1, 1, WithAbility(EntersWithCounters(CounterType(999), ColorsSpentToCastEnteringPermanent()))), "counter type"},
	} {
		if err := ValidateCardDefinition(tc.name, tc.card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
}

func TestEntryCastColorsCounterReplacement(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(PrecombatMain)
	doubler := g.PutOnBattlefield(NewCreature("Doubler", "", 1, 1), a.PlayerID())
	g.AddCounterDoubler(doubler.ID(), P1P1, ControlledBy(a.PlayerID()))
	card := NewCreature("Converge", "{2}", 2, 2, WithAbility(
		EntersWithCounters(P1P1, ColorsSpentToCastEnteringPermanent())))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(Red, 1)
	a.ManaPool().Add(Blue, 1)
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if got := g.FindPermanent(card.ID()).Counters[P1P1]; got != 4 {
		t.Fatalf("replacement counters = %d", got)
	}
}

func TestEntryCastColorsZeroAllowsAdditionalCounters(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	donor := g.PutOnBattlefield(NewCreature("Donor", "", 1, 1), a.PlayerID())
	g.AddETBAdditionalCounters(donor.ID(), P1P1, 1,
		And(IsCreature, ControlledBy(a.PlayerID())), false)
	card := NewCreature("Uncast", "", 2, 2, WithAbility(
		EntersWithCounters(P1P1, ColorsSpentToCastEnteringPermanent())))
	entry := g.PutOnBattlefield(card, a.PlayerID())
	if got := entry.Counters[P1P1]; got != 1 {
		t.Fatalf("additional counters after zero base = %d", got)
	}
}

// --- from composable_granted_collection_test.go ---
func TestGrantedCollectionMembership(t *testing.T) {
	for _, modal := range []bool{false, true} {
		for _, dynamic := range []bool{false, true} {
			g, a, b := newModalGame()
			set := ControlledPermanents(IsArtifact)
			mod := EachActivatedAbilityModification(set, Costs(), GainLifeFor(You, Fixed(2)), ActivationLimit(1))
			if modal {
				mod = EachActivatedModesModification(set, Costs(), ModeOf("gain", GainLifeFor(You, Fixed(2))), ModeOf("counter", AddCountersTo(This, Charge, Fixed(1))), ActivationLimit(1))
			}
			donor := NewEnchantment("Donor", "")
			if dynamic {
				donor = NewEnchantment("Donor", "", WithAbility(Static(While(Controls(IsArtifact), mod))))
			}
			first, second, late, enemy := NewArtifact("First", ""), NewArtifact("Second", ""), NewArtifact("Late", ""), NewArtifact("Enemy", "")
			g.PutOnBattlefield(donor, a.PlayerID())
			g.PutOnBattlefield(first, a.PlayerID())
			g.PutOnBattlefield(second, a.PlayerID())
			g.PutOnBattlefield(enemy, b.PlayerID())
			if !dynamic {
				if err := ApplyEffect(g, Until(EndOfTurn, mod), donor.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.PutOnBattlefield(late, a.PlayerID())
			g.ApplyContinuousEffects()
			wantLate := 0
			if dynamic {
				wantLate = 1
			}
			if len(g.FindPermanent(late.ID()).RuntimeAbilities) != wantLate || len(g.FindPermanent(enemy.ID()).RuntimeAbilities) != 0 {
				t.Fatalf("membership: modal=%v dynamic=%v", modal, dynamic)
			}
			for _, c := range []Card{first, second} {
				a.modes = []int{0}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), c.ID(), 0, nil); err != nil {
					t.Fatal("recipients shared limit", err)
				}
			}
			original := g.Stack().Peek()
			if g.CopyStackObjectDirect(original, b.PlayerID(), false) == nil {
				t.Fatal("missing copy")
			}
			g.PhaseOut(first.ID())
			g.ApplyContinuousEffects()
			g.PhaseIn([]uuid.UUID{first.ID()})
			g.ApplyContinuousEffects()
			a.modes = []int{0}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err == nil {
				t.Fatal("phasing reset limit")
			}
			clone := g.Clone()
			clone.RemoveFromBattlefield(clone.FindPermanent(first.ID()))
			clone.PutOnBattlefield(first, a.PlayerID())
			clone.ApplyContinuousEffects()
			wantReturned := 0
			if dynamic {
				wantReturned = 1
			}
			if len(clone.FindPermanent(first.ID()).RuntimeAbilities) != wantReturned {
				t.Fatal("wrong returned membership")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != 24 || clone.GetPlayer(b.PlayerID()).Life() != 22 || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("copy context or clone isolation")
			}
			g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
			g.ApplyContinuousEffects()
			wantRemaining := 1
			if dynamic {
				wantRemaining = 0
			}
			if len(g.FindPermanent(second.ID()).RuntimeAbilities) != wantRemaining {
				t.Fatal("wrong donor lifetime")
			}
			g.effects.RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			if len(g.FindPermanent(second.ID()).RuntimeAbilities) != 0 {
				t.Fatal("grant did not expire")
			}
			g.ResolveStack()
			if a.Life() != 24 || b.Life() != 22 {
				t.Fatal("expiration cancelled pending abilities")
			}
		}
	}
}

func TestGrantedCollectionEmptyAndSourceDuration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	donor, recipient := NewEnchantment("Donor", ""), NewArtifact("Recipient", "")
	g.PutOnBattlefield(donor, a.PlayerID())
	effect := UntilSourceLeaves(This, EachActivatedAbilityModification(MatchingPermanents(IsArtifact), Costs(), Regenerate(This)))
	if err := ApplyEffect(g, effect, donor.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(recipient, a.PlayerID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 {
		t.Fatal("empty set acquired a later member")
	}
	if err := ApplyEffect(g, effect, donor.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 1 {
		t.Fatal("missing source-bound grant")
	}
	g.PhaseOut(donor.ID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 1 {
		t.Fatal("phasing ended duration")
	}
	g.PhaseIn([]uuid.UUID{donor.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 {
		t.Fatal("source departure did not end grant")
	}
}

func TestGrantedCollectionTargetsAndControl(t *testing.T) {
	for _, modal := range []bool{false, true} {
		g, a, b := newModalGame()
		first, second := NewArtifact("First", ""), NewArtifact("Second", "")
		g.PutOnBattlefield(first, a.PlayerID())
		g.PutOnBattlefield(second, a.PlayerID())
		target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
		mod := EachActivatedAbilityModification(ControlledPermanents(IsArtifact), Costs(RemoveCountersCost(Matrix, 1)), LoseLifeFor(target, Fixed(3)), WithTargets(target), ActivationLimit(1))
		if modal {
			mod = EachActivatedModesModification(ControlledPermanents(IsArtifact), Costs(RemoveCountersCost(Matrix, 1)), ModeOf("lose", LoseLifeFor(target, Fixed(3)), WithTargets(target)), ModeOf("gain", GainLifeFor(You, Fixed(1))), ActivationLimit(1))
		}
		for _, c := range []Card{first, second} {
			g.MutablePermanent(c.ID()).AddCounter(Matrix, 2)
		}
		if err := ApplyEffect(g, Until(Indefinite, mod), first.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		a.modes = []int{0}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		g.AddControlEffect(ControlEffectSpec{SourceID: first.ID(), TargetID: second.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
		g.ApplyContinuousEffects()
		b.modes = []int{0}
		if err := g.ActivateAbilityByIndex(b.PlayerID(), second.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		for _, c := range []Card{first, second} {
			if g.FindPermanent(c.ID()).Counters[Matrix] != 1 {
				t.Fatal("wrong recipient paid cost")
			}
		}
		g.ResolveStack()
		if a.Life() != 17 || b.Life() != 17 {
			t.Fatal("recipient target or controller context shared")
		}
	}
}

// --- from composable_granted_duration_test.go ---
func TestGrantedModificationLifetime(t *testing.T) {
	for _, duration := range []Duration{Indefinite, EndOfTurn, EndOfCombat} {
		t.Run(duration.String(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			donor, recipient := NewAura("Donor", ""), NewCreature("Recipient", "", 2, 2)
			g.PutOnBattlefield(donor, a.PlayerID())
			g.PutOnBattlefield(recipient, b.PlayerID())
			g.Attach(donor.ID(), recipient.ID())
			effect := Until(duration, ActivatedAbilityModification(AttachedPermanent, Costs(RemoveCountersCost(Matrix, 1)), Sequence(AddCountersTo(This, Charge, Fixed(1)), GainLifeFor(You, Fixed(2))), ActivationLimit(1)))
			g.MutablePermanent(recipient.ID()).AddCounter(Matrix, 3)
			if err := ApplyEffect(g, effect, donor.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
			g.ApplyContinuousEffects()
			if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(recipient.ID()).Counters[Matrix] != 2 {
				t.Fatal("counter cost was not paid")
			}
			g.PhaseOut(recipient.ID())
			g.ApplyContinuousEffects()
			g.PhaseIn([]uuid.UUID{recipient.ID()})
			g.ApplyContinuousEffects()
			if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err == nil {
				t.Fatal("phasing reset limit")
			}
			clone := g.Clone()
			clone.RemoveFromBattlefield(clone.FindPermanent(recipient.ID()))
			clone.PutOnBattlefield(recipient, b.PlayerID())
			clone.ApplyContinuousEffects()
			clone.ResolveStack()
			if len(clone.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 || clone.FindPermanent(recipient.ID()).Counters[Charge] != 0 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
				t.Fatal("grant or pending body followed returned recipient")
			}
			g.ResolveStack()
			if g.FindPermanent(recipient.ID()).Counters[Charge] != 1 || b.Life() != 22 || a.Life() != 20 {
				t.Fatal("wrong granted context")
			}
			g.effects.RemoveEndOfTurn()
			g.effects.RemoveEndOfCombat()
			g.ApplyContinuousEffects()
			want := 0
			if duration == Indefinite {
				want = 1
			}
			if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != want {
				t.Fatal("wrong duration or clone shared lifetime")
			}
		})
	}
}

func TestGrantedModificationSeparateGrantsAndControl(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	recipient := NewCreature("Recipient", "", 2, 2)
	g.PutOnBattlefield(recipient, a.PlayerID())
	effect := Until(Indefinite, ActivatedAbilityModification(This, Costs(), GainLifeFor(You, Fixed(1)), ActivationLimit(1)))
	for range 2 {
		if err := ApplyEffect(g, effect, recipient.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: recipient.ID(), TargetID: recipient.ID(), ControllerID: b.PlayerID(), Duration: EndOfTurn})
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err == nil {
		t.Fatal("control change reset limit")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 1, nil); err != nil {
		t.Fatal("separate grants share limit", err)
	}
	g.ResolveStack()
	if a.Life() != 21 || b.Life() != 21 {
		t.Fatal("pending grant changed controller")
	}
}

func TestGrantedModificationStaticIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	modification := ActivatedAbilityModification(This, Costs(), GainLifeFor(You, Fixed(1)), ActivationLimit(1))
	donor := NewCreature("Donor", "", 2, 2, WithAbility(Static(While(Controls(IsCreature), modification, modification))))
	g.PutOnBattlefield(donor, a.PlayerID())
	g.ApplyContinuousEffects()
	for index := range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), donor.ID(), index+1, nil); err != nil {
			t.Fatal(err)
		}
		g.ApplyContinuousEffects()
		if err := g.ActivateAbilityByIndex(a.PlayerID(), donor.ID(), index+1, nil); err == nil {
			t.Fatal("static reevaluation reset limit")
		}
	}
}

func TestGrantedModificationAbsentAndSourceDuration(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	donor, recipient := NewAura("Donor", ""), NewCreature("Recipient", "", 2, 2)
	g.PutOnBattlefield(donor, a.PlayerID())
	g.PutOnBattlefield(recipient, a.PlayerID())
	effect := UntilSourceLeaves(This, ActivatedAbilityModification(AttachedPermanent, Costs(), Regenerate(This)))
	if err := ApplyEffect(g, effect, donor.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.Attach(donor.ID(), recipient.ID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 {
		t.Fatal("absent subject acquired later grant")
	}
	if err := ApplyEffect(g, effect, donor.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 1 {
		t.Fatal("source duration grant missing")
	}
	g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 {
		t.Fatal("source duration grant outlived source")
	}
}

func TestGrantedModificationLayerOrder(t *testing.T) {
	for _, removalFirst := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		recipient := NewCreature("Recipient", "", 2, 2)
		g.PutOnBattlefield(recipient, a.PlayerID())
		if removalFirst {
			g.AddContinuousEffect(RemoveAllAbilities(recipient.ID(), Indefinite))
		}
		if err := ApplyEffect(g, Until(Indefinite, ActivatedAbilityModification(This, Costs(), Regenerate(This))), recipient.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if !removalFirst {
			g.AddContinuousEffect(RemoveAllAbilities(recipient.ID(), Indefinite))
		}
		g.ApplyContinuousEffects()
		want := 0
		if removalFirst {
			want = 1
		}
		if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != want {
			t.Fatal("grant ignored layer timestamp")
		}
	}
}

// --- from composable_granted_limit_test.go ---
func TestStaticGrantedLimit(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(
		MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}")), GainLifeFor(You, Fixed(2)), ActivationLimit(2),
	))))
	recipient, other := NewArtifact("Recipient", ""), NewArtifact("Other", "")
	for _, card := range []Card{donor, recipient, other} {
		g.PutOnBattlefield(card, a.PlayerID())
	}
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err == nil {
		t.Fatal("unpaid activation succeeded")
	}
	a.ManaPool().Add(Colorless, 4)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	for range 3 {
		clone.ApplyContinuousEffects()
	}
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone.ApplyContinuousEffects()
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err == nil {
		t.Fatal("reevaluation reset limit")
	}
	if clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 2 || clone.StackSize() != 2 {
		t.Fatal("rejected activation changed payment or stack")
	}
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), other.ID(), 0, nil); err != nil {
		t.Fatal("recipients share limit", err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
		t.Fatal("clone changed original count", err)
	}
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err == nil {
		t.Fatal("original limit not enforced")
	}
	clone.RemoveFromBattlefield(clone.FindPermanent(donor.ID()))
	clone.ApplyContinuousEffects()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 26 {
		t.Fatal("donor departure stopped pending abilities")
	}
	g.trackers.Turn.ResetForNewTurn(b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
		t.Fatal("new turn retained limit", err)
	}
}

func TestStaticGrantedLimitIdentity(t *testing.T) {
	for _, change := range []string{"reevaluation", "control", "recipient phases", "donor phases", "recipient returns", "donor returns", "second donor"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			rule := GrantEachActivatedAbility(MatchingPermanents(IsArtifact), Costs(), GainLifeFor(You, Fixed(1)), ActivationLimit(1))
			donor := NewEnchantment("Donor", "", WithAbility(Static(rule)))
			recipient := NewArtifact("Recipient", "")
			g.PutOnBattlefield(donor, a.PlayerID())
			g.PutOnBattlefield(recipient, a.PlayerID())
			g.ApplyContinuousEffects()
			if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			controller, index, allowed := a.PlayerID(), 0, false
			switch change {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: donor.ID(), TargetID: recipient.ID(), ControllerID: b.PlayerID(), Duration: EndOfTurn})
				controller = b.PlayerID()
			case "recipient phases":
				g.PhaseOut(recipient.ID())
				g.ApplyContinuousEffects()
				g.PhaseIn([]uuid.UUID{recipient.ID()})
			case "donor phases":
				g.PhaseOut(donor.ID())
				g.ApplyContinuousEffects()
				g.PhaseIn([]uuid.UUID{donor.ID()})
			case "recipient returns":
				g.RemoveFromBattlefield(g.FindPermanent(recipient.ID()))
				g.PutOnBattlefield(recipient, a.PlayerID())
				allowed = true
			case "donor returns":
				g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
				g.PutOnBattlefield(donor, a.PlayerID())
				allowed = true
			case "second donor":
				g.PutOnBattlefield(NewEnchantment("Second donor", "", WithAbility(Static(rule))), a.PlayerID())
				index, allowed = 1, true
			}
			g.ApplyContinuousEffects()
			if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) <= index {
				t.Fatal("missing grant after change")
			}
			err := g.ActivateAbilityByIndex(controller, recipient.ID(), index, nil)
			if (err == nil) != allowed {
				t.Fatalf("allowed = %v, error = %v", allowed, err)
			}
		})
	}
}

func TestStaticGrantedManaLimit(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(
		MatchingPermanents(IsLand), Costs(TapCost(This)), AddManaAmount(Blue, Fixed(1)), ActivationLimit(1),
	)))), a.PlayerID())
	land := NewLand("Recipient")
	g.PutOnBattlefield(land, a.PlayerID())
	g.ApplyContinuousEffects()
	if err := g.AutoTapForCost(a.PlayerID(), ManaCost{Blue: 1}); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 0 || a.ManaPool().Count(Blue) != 1 {
		t.Fatal("mana did not resolve immediately")
	}
	a.ManaPool().Clear()
	g.MutablePermanent(land.ID()).Tapped = false
	g.ApplyContinuousEffects()
	if g.CanAfford(a.PlayerID(), ManaCost{Blue: 1}, nil) {
		t.Fatal("planner ignored limit")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err == nil || g.FindPermanent(land.ID()).Tapped {
		t.Fatal("blocked mana activation paid cost", err)
	}
	g.doCleanupActions()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(land.ID()).Tapped = false
	g.ApplyContinuousEffects()
	a.ManaPool().Clear()
	if g.CanAfford(a.PlayerID(), ManaCost{Blue: 1}, nil) {
		t.Fatal("manual mana activation did not record limit")
	}
}

func TestStaticGrantedLimitValidation(t *testing.T) {
	for _, options := range [][]GrantedActivationOption{{ActivationLimit(0)}, {ActivationLimit(-1)}, {ActivationLimit(1), ActivationLimit(2)}} {
		err := validateFactory("Invalid limit", func() Card {
			return NewEnchantment("Invalid limit", "", WithAbility(Static(GrantEachActivatedAbility(MatchingPermanents(IsArtifact), Costs(), Regenerate(This), options...))))
		})
		if err == nil {
			t.Fatal("invalid limit accepted")
		}
	}
}

func TestStaticGrantedLimitCopiesAndDistinctGrants(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("player")
	rule := GrantEachActivatedAbility(MatchingPermanents(IsArtifact), Costs(), GainLifeFor(target, Fixed(1)), WithTargets(target), ActivationLimit(2))
	donor := NewEnchantment("Donor", "", WithAbility(Static(rule, rule)))
	recipient := NewArtifact("Recipient", "")
	g.PutOnBattlefield(donor, a.PlayerID())
	g.PutOnBattlefield(recipient, a.PlayerID())
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err == nil {
		t.Fatal("missing target accepted")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.GetStack().Peek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal("copy consumed activation", err)
	}
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, []uuid.UUID{a.PlayerID()}); err == nil {
		t.Fatal("limit ignored")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 1, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal("separate grants share count", err)
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 21 {
		t.Fatalf("unexpected life: %d, %d", a.Life(), b.Life())
	}
}

// --- from composable_granted_modal_duration_test.go ---
func TestGrantedModalModificationLifetimeAndCopies(t *testing.T) {
	for _, duration := range []Duration{Indefinite, EndOfTurn, EndOfCombat} {
		t.Run(duration.String(), func(t *testing.T) {
			g, a, b := newModalGame()
			donor, recipient := NewAura("Donor", "", WithCastTarget(TargetArtifact())), NewArtifact("Recipient", "")
			g.PutOnBattlefield(donor, a.PlayerID())
			g.PutOnBattlefield(recipient, b.PlayerID())
			g.Attach(donor.ID(), recipient.ID())
			target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
			effect := Until(duration, ActivatedModesModification(AttachedPermanent, Costs(RemoveCountersCost(Matrix, 1)),
				ModeOf("counter", AddCountersTo(This, Charge, Fixed(1))),
				ModeOf("lose", LoseLifeFor(target, Fixed(3)), WithTargets(target)), ActivationLimit(1)))
			g.MutablePermanent(recipient.ID()).AddCounter(Matrix, 2)
			if err := ApplyEffect(g, effect, donor.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
			g.ApplyContinuousEffects()
			b.modes = []int{1}
			if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			original := g.Stack().Peek()
			a.targets = [][]uuid.UUID{{b.PlayerID()}}
			cp := g.CopyStackObjectDirect(original, a.PlayerID(), true)
			if cp == nil || cp.ModeChoice != 1 || cp.Targets[0] != b.PlayerID() || original.Targets[0] != a.PlayerID() {
				t.Fatal("copy lost mode or independent target")
			}
			g.PhaseOut(recipient.ID())
			g.ApplyContinuousEffects()
			g.PhaseIn([]uuid.UUID{recipient.ID()})
			g.ApplyContinuousEffects()
			b.modes = []int{0}
			if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err == nil {
				t.Fatal("modes or phasing reset shared limit")
			}
			if g.FindPermanent(recipient.ID()).Counters[Matrix] != 1 {
				t.Fatal("failed activation or copy paid a counter")
			}
			clone := g.Clone()
			clone.RemoveFromBattlefield(clone.FindPermanent(recipient.ID()))
			clone.PutOnBattlefield(recipient, b.PlayerID())
			clone.ApplyContinuousEffects()
			clone.ResolveStack()
			if len(clone.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 || clone.GetPlayer(a.PlayerID()).Life() != 17 || clone.GetPlayer(b.PlayerID()).Life() != 17 {
				t.Fatal("returned recipient kept grant or pending modes lost context")
			}
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("clone changed original")
			}
			g.effects.RemoveEndOfTurn()
			g.effects.RemoveEndOfCombat()
			g.ApplyContinuousEffects()
			want := 0
			if duration == Indefinite {
				want = 1
			}
			if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != want {
				t.Fatal("wrong grant duration")
			}
			g.ResolveStack()
			if a.Life() != 17 || b.Life() != 17 || g.FindPermanent(recipient.ID()).Counters[Charge] != 0 {
				t.Fatal("expiration changed pending modes or unselected mode ran")
			}
		})
	}
}

func TestGrantedModalModificationSourceDurationAndWhile(t *testing.T) {
	for _, dynamic := range []bool{false, true} {
		g, a, _ := newModalGame()
		modification := ActivatedModesModification(AttachedPermanent, Costs(),
			ModeOf("counter", AddCountersTo(This, Charge, Fixed(1))),
			ModeOf("gain", GainLifeFor(You, Fixed(2))), ActivationLimit(1))
		donor, recipient := NewAura("Donor", "", WithCastTarget(TargetArtifact())), NewArtifact("Recipient", "")
		if dynamic {
			donor = NewAura("Donor", "", WithCastTarget(TargetArtifact()), WithAbility(Static(While(Controls(IsArtifact), modification))))
		}
		g.PutOnBattlefield(donor, a.PlayerID())
		g.PutOnBattlefield(recipient, a.PlayerID())
		g.Attach(donor.ID(), recipient.ID())
		if !dynamic {
			if err := ApplyEffect(g, UntilSourceLeaves(This, modification), donor.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
		}
		g.ApplyContinuousEffects()
		a.modes = []int{0}
		if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ApplyContinuousEffects()
		if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err == nil {
			t.Fatal("reevaluation reset limit")
		}
		g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
		g.ApplyContinuousEffects()
		if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 {
			t.Fatal("grant outlived donor")
		}
		g.RemoveFromBattlefield(g.FindPermanent(recipient.ID()))
		g.PutOnBattlefield(recipient, a.PlayerID())
		g.ResolveStack()
		if g.FindPermanent(recipient.ID()).Counters[Charge] != 0 {
			t.Fatal("pending mode followed returned source")
		}
	}
}

// --- from composable_granted_mode_test.go ---
func TestStaticGrantedModesChoicesAndCopies(t *testing.T) {
	g, a, b := newModalGame()
	target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
	donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedModes(
		MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}"), TapCost(This)),
		ModeOf("gain", GainLifeFor(You, Fixed(2))),
		ModeOf("lose", LoseLifeFor(target, Fixed(3)), WithTargets(target)),
	))))
	g.PutOnBattlefield(donor, a.PlayerID())
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.ApplyContinuousEffects()
	a.modes, b.modes = []int{0}, []int{1}
	a.ManaPool().Add(Colorless, 1)
	b.ManaPool().Add(Colorless, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if top := g.Stack().Peek(); top.ModeChoice != 0 || len(top.Targets) != 0 {
		t.Fatal("targetless mode acquired another mode's targets")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), second.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	original := g.Stack().Peek()
	if original.ModeChoice != 1 || original.SourceID != second.ID() || original.Targets[0] != a.PlayerID() {
		t.Fatal("recipient lost its mode or source")
	}
	if !g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped || a.ManaPool().TotalMana() != 0 || b.ManaPool().TotalMana() != 0 || a.Life() != 20 {
		t.Fatal("incorrect payment or resolution timing")
	}
	a.targets = [][]uuid.UUID{{b.PlayerID()}}
	cp := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if cp == nil || cp.ModeChoice != 1 || cp.Targets[0] != b.PlayerID() || original.Targets[0] != a.PlayerID() {
		t.Fatal("copy changed mode or original target")
	}
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(donor.ID()))
	clone.RemoveFromBattlefield(clone.FindPermanent(second.ID()))
	clone.ApplyContinuousEffects()
	if len(clone.FindPermanent(first.ID()).RuntimeAbilities) != 0 {
		t.Fatal("donor departure retained the grant")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 19 || clone.GetPlayer(b.PlayerID()).Life() != 17 {
		t.Fatal("pending modes lost their controllers or targets after departure")
	}
	if a.Life() != 20 || b.Life() != 20 || g.StackSize() != 3 {
		t.Fatal("clone changed original game")
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 17 {
		t.Fatal("recipient choices leaked across modes")
	}
}

func TestStaticGrantedModesPaymentAndLegality(t *testing.T) {
	for _, scenario := range []string{"no mana", "wrong controller", "target returns", "recipient returns", "paid"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := newModalGame()
			target := DeclarePermanentTarget("creature", IsCreature, TargetYouControl())
			donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedModes(
				MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}"), TapCost(This)),
				ModeOf("return", Sequence(ReturnToHand(target), GainLifeFor(You, Fixed(2))), WithTargets(target)),
				ModeOf("counter", AddCountersTo(This, Charge, Fixed(1))),
			))))
			g.PutOnBattlefield(donor, a.PlayerID())
			recipient := NewArtifact("Recipient", "")
			g.PutOnBattlefield(recipient, b.PlayerID())
			card := NewCreature("Target", "", 2, 2)
			card.SetOwner(b.PlayerID())
			controller := b.PlayerID()
			if scenario == "wrong controller" {
				controller = a.PlayerID()
			}
			g.PutOnBattlefield(card, controller)
			if scenario == "wrong controller" {
				g.PutOnBattlefield(NewCreature("Legal alternative", "", 1, 1), b.PlayerID())
			}
			b.targets = [][]uuid.UUID{{card.ID()}}
			g.ApplyContinuousEffects()
			mana := 1
			if scenario == "no mana" {
				mana = 0
			}
			b.ManaPool().Add(Colorless, mana)
			targets := []uuid.UUID{card.ID()}
			if scenario == "recipient returns" {
				b.modes = []int{1}
				targets = nil
			}
			err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, targets)
			if scenario == "no mana" || scenario == "wrong controller" {
				if err == nil || g.StackSize() != 0 || g.FindPermanent(recipient.ID()).Tapped || b.ManaPool().TotalMana() != mana {
					t.Fatal("failed activation changed payment or stack")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "target returns" {
				g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
				g.PutOnBattlefield(card, b.PlayerID())
			}
			if scenario == "recipient returns" {
				g.RemoveFromBattlefield(g.FindPermanent(recipient.ID()))
				g.PutOnBattlefield(recipient, b.PlayerID())
			}
			g.ResolveStack()
			wantLife := 20
			if scenario == "paid" {
				wantLife = 22
			}
			if b.Life() != wantLife {
				t.Fatal("mode ignored target legality")
			}
			if scenario == "paid" && (len(b.Hand()) != 1 || b.Hand()[0].ID() != card.ID()) {
				t.Fatal("mode did not return its target")
			}
			if g.FindPermanent(recipient.ID()).Counters[Charge] != 0 {
				t.Fatal("source reference followed a returned card or unselected mode ran")
			}
		})
	}
}

func TestStaticGrantedModesValidation(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	valid := ModeOf("return", ReturnToHand(target), WithTargets(target))
	for _, tc := range []struct {
		name       string
		mode       ModeDefinition
		diagnostic string
	}{
		{"valid", ModeOf("gain", GainLifeFor(You, Fixed(1))), ""},
		{"scope", ModeOf("undeclared", ReturnToHand(target)), "target binding requires a declaration"},
		{"nil", ModeOf("missing", nil), "nil"},
		{"label", ModeOf("", Regenerate(This)), "supply a mode label"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewEnchantment(tc.name, "", WithAbility(Static(GrantEachActivatedModes(MatchingPermanents(IsArtifact), Costs(), valid, tc.mode))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.diagnostic == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.diagnostic) || !strings.Contains(err.Error(), "modes[1]") {
				t.Fatalf("missing mode diagnostic: %v", err)
			}
		})
	}
	for _, modes := range [][]ModalActivationOption{nil, {valid}} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("accepted fewer than two modes")
				}
			}()
			GrantEachActivatedModes(MatchingPermanents(IsArtifact), Costs(), modes...)
		}()
	}
}

func TestStaticGrantedModesUnavailableTarget(t *testing.T) {
	g, a, _ := newModalGame()
	target := DeclarePermanentTarget("creature", IsCreature)
	donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedModes(
		MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}")),
		ModeOf("return", ReturnToHand(target), WithTargets(target)),
		ModeOf("gain", GainLifeFor(You, Fixed(2))),
	))))
	g.PutOnBattlefield(donor, a.PlayerID())
	recipient := NewArtifact("Recipient", "")
	g.PutOnBattlefield(recipient, a.PlayerID())
	g.ApplyContinuousEffects()
	a.ManaPool().Add(Colorless, 1)
	a.modes = []int{0}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.Stack().Peek().ModeChoice != 1 {
		t.Fatal("menu did not map the only legal mode to its definition")
	}
	g.ResolveStack()
	if a.Life() != 22 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("legal mode did not resolve after payment")
	}
}

type grantedModeEffectChooser struct {
	*modalChoosePlayer
	offered []Mode
}

func (p *grantedModeEffectChooser) ChooseModeWithEffects(modes []Mode, _ string, _ *Game) int {
	p.offered = modes
	return 0
}

func TestStaticGrantedModesMenuFailures(t *testing.T) {
	for _, scenario := range []string{"none available", "invalid choice", "optional target", "effect chooser"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, _ := newModalGame()
			target := DeclarePermanentTarget("required", IsCreature)
			second := ModeOf("gain", GainLifeFor(You, Fixed(2)))
			if scenario == "none available" {
				second = ModeOf("regenerate", Regenerate(target), WithTargets(target))
			}
			if scenario == "optional target" {
				optional := DeclarePermanentTargets("optional", 0, 1, IsCreature)
				member := BindPermanent("member")
				second = ModeOf("optional", Sequence(ForEach(optional, member, Regenerate(From(member))), GainLifeFor(You, Fixed(2))), WithTargets(optional))
			}
			donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedModes(
				MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}"), TapCost(This)),
				ModeOf("return", ReturnToHand(target), WithTargets(target)), second,
			))))
			g.PutOnBattlefield(donor, a.PlayerID())
			recipient := NewArtifact("Recipient", "")
			g.PutOnBattlefield(recipient, a.PlayerID())
			g.ApplyContinuousEffects()
			a.ManaPool().Add(Colorless, 1)
			if scenario == "invalid choice" {
				a.modes = []int{-1}
			}
			var chooser *grantedModeEffectChooser
			if scenario == "effect chooser" {
				chooser = &grantedModeEffectChooser{modalChoosePlayer: a}
				g.players[0] = chooser
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, nil)
			if scenario == "none available" || scenario == "invalid choice" {
				if err == nil || a.ManaPool().TotalMana() != 1 || g.FindPermanent(recipient.ID()).Tapped || g.StackSize() != 0 {
					t.Fatal("invalid menu choice changed payment or stack")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if chooser != nil && (len(chooser.offered) != 1 || chooser.offered[0].Label != "gain") {
				t.Fatal("effect chooser received unavailable modes")
			}
			if g.Stack().Peek().ModeChoice != 1 {
				t.Fatal("filtered mode lost its definition index")
			}
			g.ResolveStack()
			if a.Life() != 22 {
				t.Fatal("available mode did not resolve")
			}
		})
	}
}

// --- from composable_granted_subject_test.go ---
func TestStaticGrantedSubjectAttachment(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	donor := NewAura("Donor", "", WithAbility(Static(GrantActivatedAbility(AttachedPermanent,
		Costs(ManaPayment("{1}")), Sequence(AddCountersTo(This, Charge, Fixed(1)), GainLifeFor(You, Fixed(2))), ActivationLimit(1)))))
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	donor.SetOwner(a.PlayerID())
	first.SetOwner(b.PlayerID())
	second.SetOwner(a.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.PutOnBattlefield(donor, a.PlayerID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(first.ID()).RuntimeAbilities) != 0 {
		t.Fatal("unattached donor granted an ability")
	}
	g.Attach(donor.ID(), first.ID())
	g.ApplyContinuousEffects()
	b.ManaPool().Add(Colorless, 2)
	if err := g.ActivateAbilityByIndex(b.PlayerID(), first.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.Attach(donor.ID(), second.ID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(first.ID()).RuntimeAbilities) != 0 || len(g.FindPermanent(second.ID()).RuntimeAbilities) != 1 {
		t.Fatal("grant did not follow attachment")
	}
	g.ResolveStack()
	if g.FindPermanent(first.ID()).Counters[Charge] != 1 || g.FindPermanent(second.ID()).Counters[Charge] != 0 || b.Life() != 22 || a.Life() != 20 {
		t.Fatal("pending ability changed recipient or controller")
	}
	g.Attach(donor.ID(), first.ID())
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(b.PlayerID(), first.ID(), 0, nil); err == nil {
		t.Fatal("reattachment reset activation limit")
	}
	clone := g.Clone()
	clone.PhaseOut(donor.ID())
	clone.ApplyContinuousEffects()
	if len(clone.FindPermanent(first.ID()).RuntimeAbilities) != 0 {
		t.Fatal("phased donor retained grant")
	}
	clone.PhaseIn([]uuid.UUID{donor.ID()})
	clone.ApplyContinuousEffects()
	if len(clone.FindPermanent(first.ID()).RuntimeAbilities) != 1 {
		t.Fatal("grant did not resume")
	}
	if err := clone.ActivateAbilityByIndex(b.PlayerID(), first.ID(), 0, nil); err == nil {
		t.Fatal("phasing reset limit")
	}
	clone.RemoveFromBattlefield(clone.FindPermanent(donor.ID()))
	clone.ApplyContinuousEffects()
	if len(clone.FindPermanent(first.ID()).RuntimeAbilities) != 0 || len(g.FindPermanent(first.ID()).RuntimeAbilities) != 1 {
		t.Fatal("donor lifetime or clone isolation failed")
	}
}

func TestStaticGrantedSubjectModes(t *testing.T) {
	g, a, b := newModalGame()
	target := DeclarePlayerTarget("player")
	donor := NewAura("Donor", "", WithAbility(Static(GrantActivatedModes(AttachedPermanent, Costs(),
		ModeOf("counter", AddCountersTo(This, Charge, Fixed(1))),
		ModeOf("life", GainLifeFor(target, Fixed(3)), WithTargets(target)), ActivationLimit(1)))))
	recipient := NewCreature("Recipient", "", 2, 2)
	g.PutOnBattlefield(recipient, b.PlayerID())
	g.PutOnBattlefield(donor, a.PlayerID())
	g.Attach(donor.ID(), recipient.ID())
	g.ApplyContinuousEffects()
	b.modes = []int{1}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.Stack().Peek().ModeChoice != 1 {
		t.Fatal("lost selected mode")
	}
	g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 20 {
		t.Fatal("lost mode target after donor departure")
	}
}

func TestStaticGrantedSubjectValidation(t *testing.T) {
	for _, subject := range []PermanentRef{DeclarePermanentTarget("outer", IsCreature), From(BindPermanent("outer"))} {
		card := NewEnchantment("Invalid subject", "", WithAbility(Static(GrantActivatedAbility(subject, Costs(), Regenerate(This)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "subjects") {
			t.Fatalf("missing subject diagnostic: %v", err)
		}
	}
	for _, subject := range []PermanentRef{This, AttachedPermanent} {
		card := NewEnchantment("Valid subject", "", WithAbility(Static(GrantActivatedAbility(subject, Costs(), Regenerate(This)))))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestStaticGrantedSubjectReturnedRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	donor := NewAura("Donor", "", WithAbility(Static(GrantActivatedAbility(AttachedPermanent, Costs(),
		Sequence(AddCountersTo(This, Charge, Fixed(1)), GainLifeFor(You, Fixed(2)))))))
	recipient := NewCreature("Recipient", "", 2, 2)
	donor.SetOwner(a.PlayerID())
	recipient.SetOwner(b.PlayerID())
	g.PutOnBattlefield(recipient, b.PlayerID())
	g.PutOnBattlefield(donor, a.PlayerID())
	g.Attach(donor.ID(), recipient.ID())
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(recipient.ID()))
	g.PutOnBattlefield(recipient, b.PlayerID())
	g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
	g.ResolveStack()
	if g.FindPermanent(recipient.ID()).Counters[Charge] != 0 || b.Life() != 22 {
		t.Fatal("pending grant followed returned recipient or stopped remaining instructions")
	}
}

func TestStaticGrantedSubjectNil(t *testing.T) {
	var typedNil *PermanentTargetBinding
	for _, subject := range []PermanentRef{nil, typedNil} {
		for _, modal := range []bool{false, true} {
			func() {
				defer func() {
					if recover() == nil {
						t.Error("nil subject accepted")
					}
				}()
				if modal {
					GrantActivatedModes(subject, Costs(), ModeOf("one", Regenerate(This)), ModeOf("two", GainLifeFor(You, Fixed(1))))
				} else {
					GrantActivatedAbility(subject, Costs(), Regenerate(This))
				}
			}()
		}
	}
}

// --- from composable_granted_target_test.go ---
func TestStaticGrantedTargetActivation(t *testing.T) {
	for _, scenario := range []string{"paid", "wrong controller", "no mana", "donor leaves", "recipient leaves", "target returns", "target phases", "target control changes"} {
		t.Run(scenario, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			target := DeclarePermanentTarget("creature", IsCreature, TargetYouControl())
			donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(
				MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}"), TapCost(This)),
				Sequence(ReturnToHand(target), GainLifeFor(You, Fixed(2))), WithTargets(target),
			))))
			g.PutOnBattlefield(donor, a.PlayerID())
			recipient := NewArtifact("Recipient", "")
			g.PutOnBattlefield(recipient, b.PlayerID())
			card := NewCreature("Target", "", 2, 2)
			card.SetOwner(b.PlayerID())
			controller := b.PlayerID()
			if scenario == "wrong controller" {
				controller = a.PlayerID()
			}
			g.PutOnBattlefield(card, controller)
			g.ApplyContinuousEffects()
			mana := 1
			if scenario == "no mana" {
				mana = 0
			}
			b.ManaPool().Add(Colorless, mana)
			err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, []uuid.UUID{card.ID()})
			if scenario == "wrong controller" || scenario == "no mana" {
				if err == nil || b.ManaPool().TotalMana() != mana || g.FindPermanent(recipient.ID()).Tapped || g.StackSize() != 0 {
					t.Fatal("invalid grant activation changed payment or stack")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if g.StackSize() != 1 || !g.FindPermanent(recipient.ID()).Tapped || b.ManaPool().TotalMana() != 0 || len(b.Hand()) != 0 {
				t.Fatal("grant used incorrect payment or resolution timing")
			}
			clone := g.Clone()
			switch scenario {
			case "donor leaves":
				clone.RemoveFromBattlefield(clone.FindPermanent(donor.ID()))
			case "recipient leaves":
				clone.RemoveFromBattlefield(clone.FindPermanent(recipient.ID()))
			case "target returns":
				clone.RemoveFromBattlefield(clone.FindPermanent(card.ID()))
				clone.PutOnBattlefield(card, b.PlayerID())
			case "target phases":
				clone.PhaseOut(card.ID())
			case "target control changes":
				clone.AddControlEffect(ControlEffectSpec{SourceID: donor.ID(), TargetID: card.ID(), ControllerID: a.PlayerID(), Duration: Indefinite})
			}
			clone.ApplyContinuousEffects()
			clone.ResolveStack()
			want := 1
			if scenario == "target returns" || scenario == "target phases" || scenario == "target control changes" {
				want = 0
			}
			if len(clone.GetPlayer(b.PlayerID()).Hand()) != want || clone.GetPlayer(b.PlayerID()).Life() != 20+2*want || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("grant lost target legality or recipient controller")
			}
			if g.StackSize() != 1 || len(b.Hand()) != 0 || b.Life() != 20 {
				t.Fatal("clone changed the original activation")
			}
		})
	}
}

func TestStaticGrantedTargetCopies(t *testing.T) {
	g, a, b := newSpellCopyGame()
	target := DeclarePlayerTarget("opponent", TargetOpponentOnly())
	donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(
		MatchingPermanents(IsArtifact), Costs(), LoseLifeFor(target, Fixed(3)), WithTargets(target),
	))))
	g.PutOnBattlefield(donor, a.PlayerID())
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.ApplyContinuousEffects()
	for _, entry := range []struct{ player, source, target uuid.UUID }{{a.PlayerID(), first.ID(), b.PlayerID()}, {b.PlayerID(), second.ID(), a.PlayerID()}} {
		if err := g.ActivateAbilityByIndex(entry.player, entry.source, 0, []uuid.UUID{entry.target}); err != nil {
			t.Fatal(err)
		}
	}
	original := g.GetStack().Peek()
	a.chooseQueue = [][]uuid.UUID{{b.PlayerID()}}
	cp := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if cp == nil || cp.Targets[0] != b.PlayerID() || original.Targets[0] != a.PlayerID() || cp.SourceID != second.ID() {
		t.Fatal("grant copy changed original target or recipient source")
	}
	g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
	g.ApplyContinuousEffects()
	g.ResolveStack()
	if a.Life() != 17 || b.Life() != 14 {
		t.Fatalf("grant targets leaked between recipients or copy: %d, %d", a.Life(), b.Life())
	}
}

func TestStaticGrantedTargetValidation(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	valid := GrantEachActivatedAbility(MatchingPermanents(IsCreature), Costs(), Regenerate(target), WithTargets(target))
	card := NewEnchantment("Valid", "", WithAbility(Static(valid)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	undeclared := GrantEachActivatedAbility(MatchingPermanents(IsCreature), Costs(), Regenerate(target))
	card = NewEnchantment("Separate scopes", "", WithAbility(Static(valid, undeclared)))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "target binding requires a declaration") {
		t.Fatalf("sibling grant target escaped its scope: %v", err)
	}
}

func TestStaticGrantedTargetPartialLegality(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	firstTarget := DeclarePermanentTarget("first", IsCreature)
	secondTarget := DeclarePermanentTarget("second", IsCreature)
	donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(
		MatchingPermanents(IsArtifact), Costs(),
		Sequence(ReturnToHand(firstTarget), ReturnToHand(secondTarget)), WithTargets(firstTarget, secondTarget),
	))))
	g.PutOnBattlefield(donor, a.PlayerID())
	recipient := NewArtifact("Recipient", "")
	g.PutOnBattlefield(recipient, a.PlayerID())
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	first.SetOwner(b.PlayerID())
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, []uuid.UUID{first.ID(), second.ID()}); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	g.PutOnBattlefield(first, b.PlayerID())
	g.ResolveStack()
	if len(b.Hand()) != 1 || b.Hand()[0].ID() != second.ID() || g.FindPermanent(first.ID()) == nil {
		t.Fatal("partial legality lost target group positions or incarnation")
	}
}

func TestStaticGrantedTargetManaUsesStack(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclarePlayerTarget("player")
	donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(
		MatchingPermanents(IsArtifact), Costs(), AddManaAmount(Blue, Fixed(1)), WithTargets(target),
	))))
	g.PutOnBattlefield(donor, a.PlayerID())
	recipient := NewArtifact("Recipient", "")
	g.PutOnBattlefield(recipient, b.PlayerID())
	g.ApplyContinuousEffects()
	if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 1 || b.ManaPool().TotalMana() != 0 {
		t.Fatal("targeted mana grant resolved as a mana ability")
	}
	g.ResolveStack()
	if b.ManaPool().Count(Blue) != 1 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("grant mana used the donor or target controller")
	}
}

// --- from composable_granted_timing_test.go ---
func TestStaticGrantedTiming(t *testing.T) {
	for _, timing := range []struct {
		name   string
		option GrantedActivationOption
		step   PhaseStep
	}{{"sorcery", OnlyAsSorcery(), PrecombatMain}, {"step", OnlyDuring(Upkeep), Upkeep}, {"own turn", OnlyOnYourTurn(), Upkeep}} {
		for _, scenario := range []string{"allowed", "wrong turn", "wrong step", "stack occupied"} {
			t.Run(timing.name+"/"+scenario, func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(MatchingPermanents(IsArtifact), Costs(ManaPayment("{1}")), GainLifeFor(You, Fixed(2)), timing.option))))
				g.PutOnBattlefield(donor, a.PlayerID())
				recipient := NewArtifact("Recipient", "")
				g.PutOnBattlefield(recipient, b.PlayerID())
				g.ApplyContinuousEffects()
				g.turns.SetActivePlayerIndex(1)
				g.turns.SetStep(timing.step)
				if scenario == "wrong turn" {
					g.turns.SetActivePlayerIndex(0)
				}
				if scenario == "wrong step" {
					g.turns.SetStep(EndStep)
				}
				if scenario == "stack occupied" {
					filler := NewArtifact("Filler", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
					g.PutOnBattlefield(filler, b.PlayerID())
					if err := g.ActivateAbilityByIndex(b.PlayerID(), filler.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				}
				b.ManaPool().Add(Colorless, 1)
				before := g.StackSize()
				allowed := (scenario != "wrong turn" || timing.name == "step") && (scenario != "wrong step" || timing.name == "own turn") && (scenario != "stack occupied" || timing.name != "sorcery")
				err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil)
				if !allowed {
					if err == nil || b.ManaPool().TotalMana() != 1 || g.StackSize() != before {
						t.Fatal("invalid activation changed payment or stack", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if b.ManaPool().TotalMana() != 0 || g.StackSize() != before+1 {
					t.Fatal("incorrect payment or stack timing")
				}
				clone := g.Clone()
				clone.RemoveFromBattlefield(clone.FindPermanent(donor.ID()))
				clone.turns.SetActivePlayerIndex(0)
				clone.turns.SetStep(EndStep)
				clone.ApplyContinuousEffects()
				clone.ResolveStack()
				if clone.GetPlayer(b.PlayerID()).Life() != 22+before || clone.GetPlayer(a.PlayerID()).Life() != 20 {
					t.Fatal("resolution rechecked timing or used donor controller")
				}
				if b.Life() != 20 || g.StackSize() != before+1 {
					t.Fatal("clone changed original")
				}
			})
		}
	}
}

func TestStaticGrantedTimingValidation(t *testing.T) {
	for _, options := range [][]GrantedActivationOption{{OnlyAsSorcery(), OnlyDuring(Upkeep)}, {OnlyOnYourTurn(), OnlyAsSorcery()}, {OnlyDuring(PhaseStep(-1))}} {
		if err := validateFactory("Invalid grant", func() Card {
			return NewEnchantment("Invalid grant", "", WithAbility(Static(GrantEachActivatedAbility(MatchingPermanents(IsArtifact), Costs(), Regenerate(This), options...))))
		}); err == nil {
			t.Fatal("invalid timing accepted")
		}
	}
}

// --- from composable_keyword_removal_test.go ---
func TestComposableKeywordRemoval(t *testing.T) {
	for _, keyword := range []Attr{Forestwalk, Flying, Banding} {
		for _, printed := range []bool{false, true} {
			t.Run(keyword.String()+map[bool]string{false: "/granted", true: "/printed"}[printed], func(t *testing.T) {
				a, b := NewBasePlayer("A"), NewBasePlayer("B")
				g := NewGame(a, b)
				card := NewCreature("Recipient", "", 2, 2, WithKeyword(Vigilance))
				if printed {
					card = NewCreature("Recipient", "", 2, 2, WithKeyword(Vigilance), WithKeyword(keyword))
				}
				g.PutOnBattlefield(card, a.PlayerID())
				grant := Until(EndOfTurn, GrantKeywordModification(This, keyword))
				apply := func(effect ResolvingEffect) {
					t.Helper()
					if err := ApplyEffect(g, effect, card.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
				if !printed {
					apply(grant)
					apply(grant)
				}
				removal := Until(EndOfTurn, RemoveKeyword(This, keyword))
				apply(removal)
				if g.FindPermanent(card.ID()).HasKeyword(keyword) || !g.FindPermanent(card.ID()).HasKeyword(Vigilance) {
					t.Fatal("removal must remove all instances of only the selected keyword")
				}
				apply(grant)
				if !g.FindPermanent(card.ID()).HasKeyword(keyword) {
					t.Fatal("later grant must restore the keyword")
				}
				apply(removal)
				if g.FindPermanent(card.ID()).HasKeyword(keyword) {
					t.Fatal("later removal must remove the keyword")
				}
				g.effects.RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				if g.FindPermanent(card.ID()).HasKeyword(keyword) != printed {
					t.Fatal("cleanup must restore printed abilities")
				}
			})
		}
	}
}

func TestComposableKeywordRemovalLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewArtifact("Source", "")
	card := NewCreature("Recipient", "", 2, 2, WithKeyword(Forestwalk))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(card, b.PlayerID())
	target := BindPermanent("creature")
	body := Sequence(ChooseRandomPermanent(MatchingPermanents(IsCreature), target), Until(EndOfTurn, RemoveKeyword(From(target), Forestwalk)))
	if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, b.PlayerID())
	g.PhaseIn(g.PhaseOut(card.ID()))
	g.ApplyContinuousEffects()
	if g.FindPermanent(card.ID()).HasKeyword(Forestwalk) {
		t.Fatal("source departure and recipient phasing must preserve removal")
	}
	clone := g.Clone()
	clone.effects.RemoveEndOfTurn()
	clone.ApplyContinuousEffects()
	if !clone.FindPermanent(card.ID()).HasKeyword(Forestwalk) || g.FindPermanent(card.ID()).HasKeyword(Forestwalk) {
		t.Fatal("cleanup must affect only the clone")
	}
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.ApplyContinuousEffects()
	g.PutOnBattlefield(card, b.PlayerID())
	g.ApplyContinuousEffects()
	if !g.FindPermanent(card.ID()).HasKeyword(Forestwalk) {
		t.Fatal("removal must not affect a returned incarnation")
	}
}

func TestComposableKeywordRemovalMetadata(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	body := Until(EndOfTurn, RemoveKeyword(target, Forestwalk))
	info := InspectDefinition(body)
	if info.Properties.Outcome != OutcomeDetriment || info.Properties.GrantedKeyword != 0 || !strings.Contains(info.Text, "remove Forestwalk") {
		t.Fatalf("removal metadata = %+v", info)
	}
	if _, ok := body.(*untilEffect).KeywordGrantTargetIndex([]Target{target.targetDeclaration}); ok {
		t.Fatal("removal is not a keyword grant")
	}
	if err := ValidateCardDefinition("Unbound", NewInstant("Unbound", "", Spell(body))); err == nil {
		t.Fatal("undeclared removal target must fail validation")
	}
}

// --- from composable_keyword_test.go ---
func TestComposableKeywordSourceExit(t *testing.T) {
	for _, keyword := range []Attr{Flying, Banding} {
		t.Run(keyword.String(), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			source, recipient := NewArtifact("Source", ""), NewCreature("Recipient", "", 2, 2)
			g.PutOnBattlefield(source, a.PlayerID())
			g.PutOnBattlefield(recipient, a.PlayerID())
			subjects := ControlledPermanents(IsCreature)
			body := Until(EndOfTurn, GrantEachKeyword(subjects, keyword),
				ModifyEachPT(subjects, Fixed(1), Fixed(0)), SetEachColors(subjects, Green))
			if err := ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			g.ApplyContinuousEffects()
			p := g.FindPermanent(recipient.ID())
			if !p.HasKeyword(keyword) || p.CurrentPower(g) != 3 || !slices.Contains(p.Colors(), Green) {
				t.Fatal("fixed-duration grant ended when its source left")
			}
			g.PutOnBattlefield(source, b.PlayerID())
			g.ApplyContinuousEffects()
			clone := g.Clone()
			clone.effects.RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			p = clone.FindPermanent(recipient.ID())
			if p.HasKeyword(keyword) || p.CurrentPower(clone) != 2 || slices.Contains(p.Colors(), Green) || !g.FindPermanent(recipient.ID()).HasKeyword(keyword) {
				t.Fatal("expiry did not stay in the cloned game")
			}
		})
	}
}

func TestComposableKeywordLayerOrder(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	creature := NewCreature("Creature", "", 2, 2)
	g.PutOnBattlefield(creature, a.PlayerID())
	body := Until(EndOfTurn, GrantKeywordModification(This, Flying))
	if err := ApplyEffect(g, body, creature.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.AddContinuousEffect(RemoveAllAbilities(creature.ID(), EndOfTurn))
	g.ApplyContinuousEffects()
	if g.FindPermanent(creature.ID()).HasKeyword(Flying) {
		t.Fatal("later ability removal did not remove flying")
	}
	if err := ApplyEffect(g, body, creature.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(creature.ID()).HasKeyword(Flying) {
		t.Fatal("later grant did not restore flying")
	}
}

func TestComposableRequiredBlockMetadata(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	body := Until(EndOfTurn, GrantKeywordModification(target, AttrMustBeBlockedIfAble))
	info := InspectDefinition(body)
	if info.Properties.Outcome != OutcomeDetriment || info.Properties.GrantedKeyword != AttrMustBeBlockedIfAble {
		t.Fatalf("required block metadata = %+v", info.Properties)
	}
}

func TestRegenerationProhibitionSurvivesAbilityRemoval(t *testing.T) {
	for _, removalFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "removal after", true: "removal before"}[removalFirst], func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			c := NewCreature("Recipient", "", 2, 2, WithKeyword(Flying))
			c.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(c, a.PlayerID())
			remove := func() { g.AddContinuousEffect(RemoveAllAbilities(p.ID(), EndOfTurn)) }
			if removalFirst {
				remove()
			}
			body := Until(EndOfTurn, GrantKeywordModification(This, CantRegenerate))
			if err := ApplyEffect(g, body, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if !removalFirst {
				remove()
			}
			g.ApplyContinuousEffects()
			if p.HasKeyword(Flying) || !p.HasKeyword(CantRegenerate) {
				t.Fatal("ability removal erased a regeneration prohibition or retained flying")
			}
			clone := g.Clone()
			clone.effects.RemoveEndOfTurn()
			clone.ApplyContinuousEffects()
			if clone.FindPermanent(p.ID()).HasKeyword(CantRegenerate) || !g.FindPermanent(p.ID()).HasKeyword(CantRegenerate) {
				t.Fatal("expiry did not stay in the clone")
			}
			for _, game := range []*Game{g, clone} {
				game.AddRegenerationShield(p.ID())
				if err := ApplyEffect(game, Destroy(This), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if g.FindPermanent(p.ID()) != nil || clone.FindPermanent(p.ID()) == nil {
				t.Fatal("regeneration did not follow prohibition expiry")
			}
		})
	}
}

func TestComposableRegenerationProhibitionMetadata(t *testing.T) {
	body := Until(EndOfTurn, GrantKeywordModification(This, CantRegenerate))
	info := InspectDefinition(body)
	if info.Properties.Outcome != OutcomeDetriment || info.Properties.GrantedKeyword != CantRegenerate {
		t.Fatalf("prohibition metadata = %+v", info.Properties)
	}
}

// --- from composable_land_plays_test.go ---
func TestGrantLandPlaysLifetime(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	body := GrantLandPlays(You, Fixed(1))
	for range 2 {
		if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if g.MaxLandPlays() != 3 {
		t.Fatal("grants did not accumulate")
	}
	clone := g.Clone()
	if err := ApplyEffect(clone, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if clone.MaxLandPlays() != 4 || g.MaxLandPlays() != 3 {
		t.Fatal("clone changed original allowance")
	}
	if err := ApplyEffect(g, body, uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.MaxLandPlays() != 3 || g.ExtraLandPlaysGrantedThisTurn(b.PlayerID()) != 1 {
		t.Fatal("grant used active player instead of resolving controller")
	}
	g.doCleanupActions()
	if g.MaxLandPlays() != 1 || g.ExtraLandPlaysGrantedThisTurn(b.PlayerID()) != 0 {
		t.Fatal("allowance survived cleanup")
	}
}

func TestGrantLandPlaysAmountAndAbsence(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	saved := BindNumber("allowance")
	body := Sequence(SnapshotNumber(Fixed(2), saved), GrantLandPlays(You, FromNumber(saved)))
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{0, -1} {
		if err := ApplyEffect(g, GrantLandPlays(You, Fixed(n)), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
	}
	if g.MaxLandPlays() != 3 {
		t.Fatal("incorrect saved or nonpositive amount")
	}
	calls := 0
	if err := ApplyEffect(g, GrantLandPlays(ControllerOf(This), countedLifeAmount{&calls}), uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("absent subject evaluated amount")
	}
}

func TestGrantLandPlaysValidation(t *testing.T) {
	for _, body := range []ResolvingEffect{
		GrantLandPlays(nil, Fixed(1)),
		GrantLandPlays(You, nil),
		GrantLandPlays(You, FromNumber(BindNumber("missing"))),
		GrantLandPlays(DeclarePlayerTarget("undeclared"), Fixed(1)),
	} {
		card := NewSorcery("Invalid allowance", "", WithAbility(Spell(body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid allowance accepted")
		}
	}
	info := InspectDefinition(GrantLandPlays(You, Fixed(1)))
	if info.Properties.Outcome != OutcomeBenefit || info.Role != "resolving-effect" {
		t.Fatalf("incorrect metadata: %+v", info)
	}
}

// --- from composable_pt_metadata_test.go ---
func TestTemporaryPTMetadata(t *testing.T) {
	for _, delta := range [][2]int{{0, 0}, {2, 1}, {-1, 0}, {0, -2}, {-4, -4}, {2, -1}, {-1, 2}} {
		for _, mass := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/%d/mass=%t", delta[0], delta[1], mass), func(t *testing.T) {
				modification := ModifyPT(This, Fixed(delta[0]), Fixed(delta[1]))
				if mass {
					modification = ModifyEachPT(MatchingPermanents(IsCreature), Fixed(delta[0]), Fixed(delta[1]))
				}
				want := OutcomeBenefit
				if delta[0] < 0 || delta[1] < 0 {
					want = OutcomeDetriment
				}
				for _, body := range []ResolvingEffect{Until(EndOfTurn, modification), UntilSourceLeaves(This, modification)} {
					props := InspectDefinition(body).Properties
					if props.Outcome != want || props.Mass != mass || props.PowerBoost != delta[0] || props.ToughnessBoost != delta[1] {
						t.Fatalf("P/T metadata = %+v, want outcome %v and delta %v", props, want, delta)
					}
				}
			})
		}
	}
}

// --- from composable_self_cast_colors_test.go ---
func TestColorsSpentToCastThisSpellResolutionAndIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spell := NewInstant("Converge test", "", Spell(GainLifeFor(You, ColorsSpentToCastThisSpell())))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	original := g.StackObjects()[0]
	original.CastContext.ColorsSpent = map[Color]int{Red: 2, Blue: 1, Colorless: 3, Green: 0}
	spellCopy := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	spellCopy.CastContext.ColorsSpent[Green] = 1
	clone := g.Clone()
	clone.StackObjects()[0].CastContext.ColorsSpent[White] = 1

	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 23 {
		t.Fatalf("original life totals = %d, %d", a.Life(), b.Life())
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || clone.GetPlayer(b.PlayerID()).Life() != 23 {
		t.Fatalf("clone life totals = %d, %d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life())
	}
}

func TestColorsSpentToCastThisSpellAbsentAndDeclaration(t *testing.T) {
	value := ColorsSpentToCastThisSpell()
	if got := value.resolveValue(&EffectContext{Game: NewGame(NewBasePlayer("A"), NewBasePlayer("B"))}); got != 0 {
		t.Fatalf("absent cast context = %d", got)
	}
	if got := InspectDefinition(value); got.Role != "resolution-value" || !strings.Contains(got.Text, "colors") {
		t.Fatalf("inspection = %#v", got)
	}
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"spell", NewInstant("Spell", "", Spell(GainLifeFor(You, value))), ""},
		{"activation", NewArtifact("Activation", "", WithAbility(Activate(Costs(), GainLifeFor(You, value)))), "spell ability"},
		{"trigger", NewArtifact("Trigger", "", WithAbility(WhenEvent(Enters(This), GainLifeFor(You, value)))), "spell ability"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCardDefinition(tc.card.Name(), tc.card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}

func TestColorsSpentToCastThisSpellPaidCastAndFailedPayment(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spell := NewInstant("Paid converge", "{B}{U}", Spell(GainLifeFor(You, ColorsSpentToCastThisSpell())))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(Black, 1)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err == nil {
		t.Fatal("cast without blue mana succeeded")
	}
	if g.StackSize() != 0 || a.Life() != 20 {
		t.Fatal("failed payment changed stack or life")
	}
	a.ManaPool().Add(Blue, 1)
	a.ManaPool().Add(Colorless, 1)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("paid colors gave life %d", a.Life())
	}
}

// --- from composable_self_entry_colors_test.go ---
func TestSelfEntryColorsSpentFromCastAndTriggerIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(PrecombatMain)
	colors := BindNumber("entry colors")
	creature := NewCreature("Converge entrant", "", 2, 2, WithAbility(WhenEvent(
		Enters(This).WithColorsSpent(colors), GainLifeFor(You, FromNumber(colors)))))
	creature.SetOwner(a.PlayerID())
	a.AddToHand(creature)
	if err := g.CastSpellByName(a.PlayerID(), creature.Name(), nil); err != nil {
		t.Fatal(err)
	}
	obj := g.stack.Pop()
	obj.CastContext.ColorsSpent = map[Color]int{Red: 2, Blue: 1, Colorless: 3, Green: 0}
	g.ResolveStackObject(obj)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack size = %d", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	clone := g.Clone()
	clone.PhaseOut(creature.ID())
	g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
	g.PutOnBattlefield(creature, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatalf("original life = %d, %d", a.Life(), b.Life())
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatal("clone lost trigger capture")
	}
}

func TestSelfEntryColorsSpentWithoutCastAndDeclaration(t *testing.T) {
	colors := BindNumber("entry colors")
	pattern := Enters(This).WithColorsSpent(colors)
	valid := NewCreature("Valid", "", 1, 1, WithAbility(WhenEvent(pattern, GainLifeFor(You, FromNumber(colors)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	if node := InspectDefinition(pattern); node.Role != "event-pattern" || len(node.Children) != 1 {
		t.Fatalf("inspection = %+v", node)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(valid, a.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatalf("uncast entry gained %d life", a.Life()-20)
	}
	for _, tc := range []struct {
		name    string
		pattern EventPattern
		want    string
	}{
		{"nil", Enters(This).WithColorsSpent(nil), "binding"},
		{"unnamed", Enters(This).WithColorsSpent(BindNumber("")), "name"},
		{"repeated", Enters(This).WithColorsSpent(colors).WithColorsSpent(colors), "one"},
		{"wrong event", Dies(This).WithColorsSpent(colors), "entry"},
	} {
		card := NewCreature(tc.name, "", 1, 1, WithAbility(WhenEvent(tc.pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s validation = %v", tc.name, err)
		}
	}
	unbound := NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(Enters(This), GainLifeFor(You, FromNumber(colors)))))
	if err := ValidateCardDefinition(unbound.Name(), unbound); err == nil {
		t.Fatal("accepted unbound capture")
	}
}

func TestSelfEntryColorsSpentPaidCastAndRollback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(PrecombatMain)
	colors := BindNumber("entry colors")
	creature := NewCreature("Paid entrant", "{2}{R}", 2, 2, WithAbility(WhenEvent(
		Enters(This).WithColorsSpent(colors), GainLifeFor(You, FromNumber(colors)))))
	creature.SetOwner(a.PlayerID())
	a.AddToHand(creature)
	a.ManaPool().Add(Red, 1)
	if err := g.CastSpellByName(a.PlayerID(), creature.Name(), nil); err == nil {
		t.Fatal("cast succeeded without generic mana")
	}
	if g.StackSize() != 0 || a.Life() != 20 {
		t.Fatal("failed payment changed state")
	}
	a.ManaPool().Add(Blue, 1)
	a.ManaPool().Add(Colorless, 1)
	if err := g.CastSpellByName(a.PlayerID(), creature.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("life after paid entry = %d", a.Life())
	}
}

// --- from composable_skip_untap_test.go ---
func TestComposableSkipNextUntapSavedLifetime(t *testing.T) {
	for _, change := range []string{"none", "return", "absent", "phase", "phase cycle", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, p := controllerTestGame(t)
			id := p.ID()
			saved := BindPermanent("saved")
			ctx := &EffectContext{Game: g, SourceID: id, Controller: a.PlayerID()}
			if err := Snapshot(This, saved).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, a.PlayerID())
			case "absent":
				g.RemoveFromBattlefield(p)
			case "phase":
				g.PhaseOut(id)
			case "phase cycle":
				g.PhaseIn(g.PhaseOut(id))
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: id, ControllerID: b.PlayerID(), Duration: Indefinite})
			}
			clone := g.Clone()
			ctx.Game = clone
			if err := SkipNextUntap(From(saved)).Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := change == "none" || change == "phase cycle" || change == "control"
			if clone.turns.HasSkipNextUntap(id) != want {
				t.Fatalf("skip = %v, want %v", clone.turns.HasSkipNextUntap(id), want)
			}
			if g.turns.HasSkipNextUntap(id) {
				t.Fatal("clone changed the original")
			}
		})
	}
}

func TestComposableSkipNextUntapTargetAndSourceExit(t *testing.T) {
	g, a, _, p := controllerTestGame(t)
	p.Tapped = true
	target := DeclarePermanentTarget("creature", IsCreature)
	body := SkipNextUntap(target)
	ability := Activate(Costs(), body, WithTargets(target))
	card := NewArtifact("Skip source", "", WithAbility(ability))
	source := g.PutOnBattlefield(card, a.PlayerID())
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: source.ID(), Card: card, IsAbility: true, Controller: a.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.RemoveFromBattlefield(source)
	g.ResolveStack()
	g.doUntap()
	if !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("source departure removed the restriction")
	}
	g.doUntap()
	if g.FindPermanent(p.ID()).Tapped {
		t.Fatal("restriction did not expire")
	}
	if body.Properties().Outcome != OutcomeDetriment {
		t.Fatal("missing detriment metadata")
	}
}

func TestComposableSkipNextUntapValidation(t *testing.T) {
	saved := BindPermanent("saved")
	for _, body := range []ResolvingEffect{
		SkipNextUntap(nil),
		SkipNextUntap(From(saved)),
		SkipNextUntap(DeclarePermanentTarget("undeclared", IsCreature)),
		Sequence(SkipNextUntap(From(saved)), Snapshot(This, saved)),
	} {
		card := NewCreature("Invalid skip", "", 1, 1, WithAbility(Activate(Costs(), body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid subject")
		}
	}
}

// --- from composable_spell_cast_colors_test.go ---
func TestSpellCastDistinctColorsCapturedAtEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	colors := BindNumber("colors")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().OfType(TypeInstant).ByOpponent().WithColorsSpent(colors),
		GainLifeFor(You, FromNumber(colors)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Spell", "", Spell(DrawFor(You, Fixed(0))))
	spell.SetOwner(b.PlayerID())
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: b.PlayerID(),
		CastContext: &CastContext{ColorsSpent: map[Color]int{Red: 2, Blue: 1, Colorless: 3, Green: 0}}}
	g.pushStack(obj)
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	obj.CastContext.ColorsSpent[Green] = 5
	g.RemoveFromBattlefield(g.FindPermanent(watcher.ID()))
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.PutTriggersOnStack()
	g.stack.RemoveByID(obj.ID)
	clone := g.Clone()
	clone.ResolveStack()
	if got := g.GetPlayer(a.PlayerID()).Life(); got != 20 {
		t.Fatalf("clone changed original life = %d", got)
	}
	g.ResolveStack()
	for _, game := range []*Game{g, clone} {
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 22 {
			t.Fatalf("colors spent = %d", got)
		}
	}
}

func TestSpellCastDistinctColorsZeroAndSourcePhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	colors := BindNumber("colors")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().WithColorsSpent(colors), GainLifeFor(You, MaxValue(Fixed(1), FromNumber(colors))))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	permanent := g.FindPermanent(watcher.ID())
	spell := NewInstant("Free", "", Spell(DrawFor(You, Fixed(0))))
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: b.PlayerID(), CastContext: &CastContext{}}
	g.pushStack(obj)
	permanent.PhasedOut = true
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 1 {
		t.Fatalf("phased source stack = %d", got)
	}
	permanent.PhasedOut = false
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	g.stack.RemoveByID(obj.ID)
	g.ResolveStack()
	if got := a.Life(); got != 21 {
		t.Fatalf("zero-color cast life = %d", got)
	}
}

func TestSpellCastDistinctColorsDeclaration(t *testing.T) {
	colors := BindNumber("colors")
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"valid", NewArtifact("Valid", "", WithAbility(WhenEvent(SpellCast().WithColorsSpent(colors), GainLifeFor(You, FromNumber(colors))))), ""},
		{"nil", NewArtifact("Nil", "", WithAbility(WhenEvent(SpellCast().WithColorsSpent(nil), GainLifeFor(You, Fixed(1))))), "binding"},
		{"unnamed", NewArtifact("Unnamed", "", WithAbility(WhenEvent(SpellCast().WithColorsSpent(BindNumber("")), GainLifeFor(You, Fixed(1))))), "name"},
		{"duplicate", NewArtifact("Duplicate", "", WithAbility(WhenEvent(SpellCast().WithColorsSpent(colors).WithColorsSpent(BindNumber("other")), GainLifeFor(You, Fixed(1))))), "one"},
		{"unbound", NewArtifact("Unbound", "", WithAbility(WhenEvent(SpellCast(), GainLifeFor(You, FromNumber(colors))))), "producer"},
		{"wrong zone", NewInstant("Wrong", "", WithAbility(WhenEvent(SpellCast().WithColorsSpent(colors), GainLifeFor(You, FromNumber(colors))))), "battlefield"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCardDefinition(tc.card.Name(), tc.card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	if got := InspectDefinition(SpellCast().WithColorsSpent(colors)); got.Role != "event-pattern" || len(got.Children) != 1 {
		t.Fatalf("inspection = %#v", got)
	}
}

func TestDistinctColorsSpentExcludesColorless(t *testing.T) {
	ctx := &CastContext{ColorsSpent: map[Color]int{
		White: 1, Blue: 2, Black: 1, Red: 1, Green: 1, Colorless: 4,
	}}
	if got := ctx.DistinctColorsSpent(); got != 5 {
		t.Fatalf("distinct colors = %d", got)
	}
	if got := (*CastContext)(nil).DistinctColorsSpent(); got != 0 {
		t.Fatalf("nil cast context colors = %d", got)
	}
}

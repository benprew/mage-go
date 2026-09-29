package mage

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"

	"github.com/google/uuid"
)

// --- from composable_ability_activation_filter_test.go ---
func TestActivatedAbilityArtifactOpponentFilter(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	actor := BindPlayer("actor")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().FromArtifact().ByOpponent().WithActivator(actor),
		GainLifeFor(FromPlayer(actor), Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	other := NewArtifact("Other", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(other, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), other.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	artifact := NewArtifact("Artifact", "", WithAbility(Activate(Costs(TapCost(This)), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(artifact, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	creature := NewCreature("Creature", "", 1, 1, WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(creature, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), creature.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 4 {
		t.Fatalf("stack = %d, want 4", got)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if b.Life() != 23 {
		t.Fatalf("life = %d, want 23", b.Life())
	}
	if a.Life() != 21 {
		t.Fatalf("life = %d, want 21", a.Life())
	}
}

func TestActivatedAbilityArtifactOpponentFilterDefinition(t *testing.T) {
	good := NewEnchantment("Good", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().FromArtifact().ByOpponent(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []EventPattern{
		ActivatedAbilityEvent().FromArtifact().FromArtifact(),
		ActivatedAbilityEvent().ByOpponent().ByOpponent(),
	} {
		bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
			t.Fatal("duplicate filter passed registration")
		}
	}
}

func TestActivatedAbilityArtifactOpponentSourceLKI(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := BindDamageSource("source")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().FromArtifact().ByOpponent().WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Artifact", "", WithKeyword(Lifelink), WithAbility(
		Activate(Costs(SacrificeSourceCost()), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(artifact, b.PlayerID())
	old := g.FindPermanent(artifact.ID()).incarnationID
	if err := g.ActivateAbilityByIndex(b.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(artifact, a.PlayerID())
	if g.FindPermanent(artifact.ID()).incarnationID == old {
		t.Fatal("return kept old incarnation")
	}
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 2 {
		t.Fatalf("stack = %d, want 2", got)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 23 {
		t.Fatalf("life = %d/%d, want 18/23", a.Life(), b.Life())
	}
}

// --- from composable_ability_activation_no_tap_test.go ---
func TestActivatedAbilityWithoutTapCostFilter(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	actor := BindPlayer("actor")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().FromArtifact().ByOpponent().WithoutTapCost().WithActivator(actor),
		GainLifeFor(FromPlayer(actor), Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	plain := NewArtifact("Plain", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	tapped := NewArtifact("Tapped", "", WithAbility(Activate(Costs(TapCost(This)), GainLifeFor(You, Fixed(1)))))
	creature := NewCreature("Creature", "", 1, 1, WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	failed := NewArtifact("Failed", "", WithAbility(Activate(Costs(LifePayCost(30)), GainLifeFor(You, Fixed(1)))))
	for _, card := range []Card{plain, tapped, creature, failed} {
		g.PutOnBattlefield(card, b.PlayerID())
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), plain.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), tapped.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), creature.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), failed.ID(), 0, nil); err == nil {
		t.Fatal("unpaid activation succeeded")
	}
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 4 {
		t.Fatalf("stack = %d, want 4", got)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 25 {
		t.Fatalf("life = %d/%d, want 20/25", a.Life(), b.Life())
	}
}

func TestActivatedAbilityWithoutTapCostSourceAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := BindDamageSource("source")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().FromArtifact().ByOpponent().WithoutTapCost().WithSource(source),
		DealDamageFrom(FromDamageSource(source), You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	artifact := NewArtifact("Artifact", "", WithKeyword(Lifelink), WithAbility(
		Activate(Costs(SacrificeSourceCost()), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(artifact, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), artifact.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(artifact, a.PlayerID())
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 22 {
		t.Fatalf("life = %d/%d, want 19/22", a.Life(), b.Life())
	}
	g.PhaseOut(watcher.ID())
	other := NewArtifact("Other", "", WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(other, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), other.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 1 {
		t.Fatalf("phased watcher created trigger: stack = %d", got)
	}
}

func TestActivatedAbilityWithoutTapCostManaAbilities(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		ActivatedAbilityEvent().WithoutTapCost(), GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	plain := NewArtifact("Plain mana", "", WithAbility(Activate(Costs(), AddManaAmount(Blue, Fixed(1)))))
	tapped := NewArtifact("Tap mana", "", WithAbility(NewManaAbility(Blue)))
	g.PutOnBattlefield(plain, b.PlayerID())
	g.PutOnBattlefield(tapped, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), plain.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), tapped.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if got := g.StackSize(); got != 1 {
		t.Fatalf("stack = %d, want 1", got)
	}
	g.ResolveStack()
	if got := a.Life(); got != 21 {
		t.Fatalf("life = %d, want 21", got)
	}
}

func TestActivatedAbilityWithoutTapCostDefinition(t *testing.T) {
	pattern := ActivatedAbilityEvent().FromArtifact().ByOpponent().WithoutTapCost()
	if got := InspectDefinition(pattern); got.Role != "event-pattern" {
		t.Fatalf("role = %q", got.Role)
	}
	good := NewEnchantment("Good", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(good.Name(), good); err != nil {
		t.Fatal(err)
	}
	bad := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern.WithoutTapCost(), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(bad.Name(), bad); err == nil {
		t.Fatal("repeated filter passed registration")
	}
}

// --- from composable_activation_capture_test.go ---
func TestActivationNumberCaptureSacrificeAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("credits")
	body := Sequence(GainLifeFor(You, FromNumber(n)), SnapshotNumber(Fixed(99), n))
	card := NewCreature("Source", "", 1, 1, WithAbility(Activate(Costs(SacrificeSourceCost()), body,
		CaptureNumberBeforePayment(CountersOn(This, Credit), n))))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID()).AddCounter(Credit, 4)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil || g.StackSize() != 1 || a.Life() != 20 {
		t.Fatal("cost or timing is incorrect")
	}
	g.PutOnBattlefield(card, a.PlayerID()).AddCounter(Credit, 10)
	clone := g.Clone()
	for _, game := range []*Game{clone, g} {
		game.CopyStackObjectDirect(game.StackObjects()[0], b.PlayerID(), false)
		game.ResolveStack()
		if game.GetPlayer(a.PlayerID()).Life() != 24 || game.GetPlayer(b.PlayerID()).Life() != 24 {
			t.Fatal("capture changed after payment, return, or copy")
		}
		if FromNumber(n).resolveValue(&EffectContext{Game: game}) != 0 {
			t.Fatal("capture escaped resolution")
		}
		if game == clone && a.Life() != 20 {
			t.Fatal("clone changed original")
		}
	}
}

func TestActivationNumberCaptureRepeatedAndFailedPayment(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("charge")
	card := NewArtifact("Source", "", WithAbility(Activate(Costs(RemoveCountersCost(Charge, 2)), GainLifeFor(You, FromNumber(n)),
		CaptureNumberBeforePayment(CountersOn(This, Charge), n))))
	g.PutOnBattlefield(card, a.PlayerID()).AddCounter(Charge, 5)
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil); err == nil {
		t.Fatal("unpaid activation succeeded")
	}
	if g.StackSize() != 2 || g.FindPermanent(card.ID()).Counters[Charge] != 1 {
		t.Fatal("failed payment changed state")
	}
	g.ResolveStack()
	if a.Life() != 28 {
		t.Fatalf("life = %d", a.Life())
	}
}

func TestActivationNumberCaptureValidation(t *testing.T) {
	n := BindNumber("number")
	for _, tc := range []struct {
		name     string
		captures []ActivationOption
		body     ResolvingEffect
		want     string
	}{
		{"valid", []ActivationOption{CaptureNumberBeforePayment(Fixed(2), n)}, GainLifeFor(You, FromNumber(n)), ""},
		{"empty name", []ActivationOption{CaptureNumberBeforePayment(Fixed(2), BindNumber(""))}, GainLifeFor(You, Fixed(1)), "name"},
		{"duplicate name", []ActivationOption{CaptureNumberBeforePayment(Fixed(2), n), CaptureNumberBeforePayment(Fixed(3), BindNumber("number"))}, GainLifeFor(You, Fixed(1)), "duplicate"},
		{"overwrite", []ActivationOption{CaptureNumberBeforePayment(Fixed(2), n), CaptureNumberBeforePayment(SubtractValues(FromNumber(n), Fixed(1)), n)}, GainLifeFor(You, FromNumber(n)), ""},
		{"self read", []ActivationOption{CaptureNumberBeforePayment(FromNumber(n), n)}, GainLifeFor(You, Fixed(1)), "producer"},
		{"nil value", []ActivationOption{CaptureNumberBeforePayment(nil, n)}, GainLifeFor(You, Fixed(1)), "non-nil"},
		{"nil binding", []ActivationOption{CaptureNumberBeforePayment(Fixed(1), nil)}, GainLifeFor(You, Fixed(1)), "non-nil"},
		{"body too late", []ActivationOption{CaptureNumberBeforePayment(FromNumber(n), BindNumber("other"))}, SnapshotNumber(Fixed(2), n), "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewArtifact("Capture", "", WithAbility(Activate(Costs(), tc.body, tc.captures...)))
			err := ValidateCardDefinition(c.Name(), c)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestActivationNumberCaptureTargetsAndOrder(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n, total := BindNumber("power"), BindNumber("total")
	target := DeclarePermanentTarget("creature", IsCreature)
	card := NewArtifact("Source", "", WithAbility(Activate(Costs(RemoveCountersCost(Charge, 1)),
		Sequence(ReturnToHand(target), GainLifeFor(You, FromNumber(total))),
		CaptureNumberBeforePayment(PowerOf(target), n),
		CaptureNumberBeforePayment(SubtractValues(FromNumber(n), CountersOn(This, Charge)), total),
		WithTargets(target))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID()).AddCounter(Charge, 2)
	creature := NewCreature("Creature", "", 3, 3)
	creature.SetOwner(b.PlayerID())
	g.PutOnBattlefield(creature, b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, []uuid.UUID{card.ID()}); err == nil {
		t.Fatal("invalid target succeeded")
	}
	if g.FindPermanent(card.ID()).Counters[Charge] != 2 || g.StackSize() != 0 {
		t.Fatal("invalid target changed state")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, []uuid.UUID{creature.ID()}); err != nil {
		t.Fatal(err)
	}
	g.MutablePermanent(creature.ID()).AddCounter(P1P1, 5)
	g.ResolveStack()
	if a.Life() != 21 || g.FindPermanent(creature.ID()) != nil {
		t.Fatal("target capture or ordering failed")
	}
}

func TestActivationNumberCaptureGrantedModalAndDelayed(t *testing.T) {
	g, a, b := newModalGame()
	n := BindNumber("credits")
	rule := GrantEachActivatedModes(MatchingPermanents(IsArtifact), Costs(SacrificeSourceCost()),
		CaptureNumberBeforePayment(CountersOn(This, Credit), n),
		ModeOf("now", GainLifeFor(You, FromNumber(n))),
		ModeOf("later", AtNext(EachEndStep, GainLifeFor(You, FromNumber(n)))))
	donor := NewEnchantment("Donor", "", WithAbility(Static(rule)))
	if err := ValidateCardDefinition(donor.Name(), donor); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(donor, a.PlayerID())
	for _, count := range []int{2, 5} {
		recipient := NewArtifact("Recipient", "")
		recipient.SetOwner(b.PlayerID())
		g.PutOnBattlefield(recipient, b.PlayerID()).AddCounter(Credit, count)
		g.ApplyContinuousEffects()
		b.modes = []int{1}
		if err := g.ActivateAbilityByIndex(b.PlayerID(), recipient.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	if b.Life() != 20 {
		t.Fatal("delayed body resolved early")
	}
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 27 {
		t.Fatal("grant or delayed capture used wrong values or controller")
	}
}

func TestActivationNumberCaptureMana(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	n := BindNumber("amount")
	source := NewArtifact("Source", "", WithAbility(Activate(Costs(RemoveCountersCost(Charge, 2)),
		AddManaAmount(Blue, FromNumber(n)), CaptureNumberBeforePayment(CountersOn(This, Charge), n))))
	g.PutOnBattlefield(source, a.PlayerID()).AddCounter(Charge, 4)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 0 || a.ManaPool().TotalMana() != 4 {
		t.Fatal("mana capture did not resolve immediately")
	}
}

func TestActivationNumberCaptureSignedValuesAndFrameIsolation(t *testing.T) {
	for _, amount := range []int{-3, 0, 4} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		n := BindNumber("number")
		g.resolution.numberBindings = map[*NumberBinding]int{n: 50}
		body := GainLifeFor(You, SubtractValues(Fixed(5), FromNumber(n)))
		c := NewArtifact("Source", "", WithAbility(Activate(Costs(), body, CaptureNumberBeforePayment(Fixed(amount), n))))
		g.PutOnBattlefield(c, a.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), c.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		if g.resolution.numberBindings[n] != 50 {
			t.Fatal("activation changed outer frame")
		}
		g.ResolveStack()
		if a.Life() != 25-amount || len(g.resolution.numberBindings) != 0 {
			t.Fatal("signed value or resolution cleanup is incorrect")
		}
	}
}

func TestActivationNumberCaptureScope(t *testing.T) {
	n := BindNumber("number")
	target := DeclarePermanentTarget("target", IsCreature)
	cards := []Card{
		NewArtifact("Other ability", "",
			WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), CaptureNumberBeforePayment(Fixed(2), n))),
			WithAbility(Activate(Costs(), GainLifeFor(You, FromNumber(n))))),
		NewArtifact("Mode target", "", WithAbility(ActivateModes(Costs(),
			CaptureNumberBeforePayment(PowerOf(target), n),
			ModeOf("one", GainLifeFor(You, FromNumber(n)), WithTargets(target)),
			ModeOf("two", GainLifeFor(You, FromNumber(n)))))),
	}
	for _, c := range cards {
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatalf("invalid capture scope accepted for %s", c.Name())
		}
	}
}

// --- from composable_activation_test.go ---
func TestComposableLimitedPumpPayment(t *testing.T) {
	for _, limit := range []int{1, 2} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.turns.SetActivePlayerIndex(1)
			g.turns.SetStep(Upkeep)
			card := NewCreature("Source", "", 1, 1, WithAbility(Activate(
				Costs(ManaPayment("{R}")),
				Until(EndOfTurn, ModifyPT(This, Fixed(1), Fixed(0))), ActivationLimit(limit))))
			g.PutOnBattlefield(card, a.PlayerID())
			activate := func(game *Game) error {
				return game.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, nil)
			}
			if err := activate(g); err == nil || g.StackSize() != 0 {
				t.Fatal("unpaid activation changed the stack", err)
			}
			a.ManaPool().Add(Red, limit+1)
			for range limit {
				if err := activate(g); err != nil {
					t.Fatal("failed payment consumed an activation", err)
				}
			}
			if err := activate(g); err == nil || g.StackSize() != limit || a.ManaPool().TotalMana() != 1 {
				t.Fatal("limit did not prevent payment before resolution", err)
			}
			clone := g.Clone()
			clone.ResolveStack()
			clone.ApplyContinuousEffects()
			if got := clone.FindPermanent(card.ID()).CurrentPower(clone); got != 1+limit {
				t.Fatalf("power = %d, want %d", got, 1+limit)
			}
			if err := activate(clone); err == nil {
				t.Fatal("resolution reset the activation limit")
			}
			clone.doCleanupActions()
			clone.turns.SetStep(Upkeep)
			clone.ApplyContinuousEffects()
			if got := clone.FindPermanent(card.ID()).CurrentPower(clone); got != 1 {
				t.Fatalf("cleanup retained the bonus: power = %d", got)
			}
			clone.GetPlayer(a.PlayerID()).ManaPool().Add(Red, 1)
			if err := activate(clone); err != nil {
				t.Fatal("cleanup retained the activation limit", err)
			}
			if err := activate(g); err == nil || g.StackSize() != limit || a.ManaPool().TotalMana() != 1 {
				t.Fatal("clone changed the original activation state", err)
			}
		})
	}
}

func TestComposableActivationRestrictions(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.turns.SetStep(Upkeep)
	untapOnly := Activate(Costs(), GainLifeFor(You, Fixed(1)), OnlyDuring(Untap))
	if untapOnly.CanActivate(a.PlayerID(), g) {
		t.Fatal("zero-valued untap restriction allowed upkeep activation")
	}
	ability := Activate(Costs(), GainLifeFor(You, Fixed(1)), OnlyDuring(Upkeep), ActivationLimit(2), ActivationIf(Controls(IsCreature)))
	card := NewCreature("Source", "", 1, 1, WithAbility(ability))
	card.SetOwner(a.PlayerID())
	g.PutOnBattlefield(card, a.PlayerID())
	ability.SetSource(card.ID())
	g.turns.SetStep(Upkeep)
	if !ability.CanActivate(a.PlayerID(), g) {
		t.Fatal("valid activation rejected")
	}
	g.turns.SetStep(EndStep)
	if ability.CanActivate(a.PlayerID(), g) {
		t.Fatal("step restriction ignored")
	}
	g.turns.SetStep(Upkeep)
	ability.activationsThisTurn = 2
	if ability.CanActivate(a.PlayerID(), g) {
		t.Fatal("limit ignored")
	}
	ability.activationsThisTurn = 0
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	if ability.CanActivate(a.PlayerID(), g) {
		t.Fatal("state condition ignored")
	}
}

func TestComposableActivationOptionValidation(t *testing.T) {
	for _, options := range [][]ActivationOption{
		{AnyPlayerMayActivate(), AnyPlayerMayActivate()},
		{OnlyAsSorcery(), OnlyDuring(Upkeep)},
		{OnlyDuring(Upkeep), OnlyDuring(Upkeep)},
		{OnlyOnYourTurn(), OnlyOnYourTurn()},
		{OnlyDuring(Upkeep), OnlyOnYourTurn(), OnlyDuring(EndStep)},
		{OnlyOnYourTurn(), OnlyDuring(Upkeep), OnlyOnYourTurn()},
		{OnlyOnYourTurn(), OnlyAsSorcery()},
		{ActivationLimit(1), ActivationLimit(2)},
		{ActivationLimit(0)},
		{OnlyDuring(PhaseStep(-1))},
		{ActivationIf(nil)},
	} {
		if err := validateFactory("Invalid activation", func() Card {
			return NewCreature("Invalid activation", "", 1, 1, WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), options...)))
		}); err == nil {
			t.Fatal("invalid options accepted")
		}
	}
	ability := Activate(Costs(), GainLifeFor(You, Fixed(1)), ActivationIf(AllConditions()))
	if err := ValidateCardDefinition("Invalid condition", NewCreature("Invalid condition", "", 1, 1, WithAbility(ability))); err == nil {
		t.Fatal("activation condition was not inspected")
	}
}

// --- from composable_attached_tap_cost_test.go ---
func TestTapAttachedPermanentCostPayment(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	land := NewLand("Host")
	host := g.PutOnBattlefield(land, a.PlayerID())
	aura := NewAura("Source", "", WithCastTarget(TargetLand()))
	source := g.PutOnBattlefield(aura, a.PlayerID())
	g.Attach(source.ID(), host.ID())
	cost := TapAttachedPermanentCost()
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(host.ID()).Tapped || g.FindPermanent(source.ID()).Tapped {
		t.Fatal("payment did not tap only the attached permanent")
	}
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}}); err == nil {
		t.Fatal("already tapped host paid again")
	}
}

func TestTapAttachedPermanentCostStaleAttachmentAndRollback(t *testing.T) {
	for _, change := range []string{"detach", "reattach", "phase", "return", "tap", "source return"} {
		t.Run(change, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			host := g.PutOnBattlefield(NewLand("Host"), a.PlayerID())
			other := g.PutOnBattlefield(NewLand("Other"), a.PlayerID())
			auraCard := NewAura("Source", "", WithCastTarget(TargetLand()))
			source := g.PutOnBattlefield(auraCard, a.PlayerID())
			g.Attach(source.ID(), host.ID())
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{LifePayCost(2), TapAttachedPermanentCost()}})
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "detach":
				g.MutablePermanent(source.ID()).AttachedTo = other.ID()
			case "reattach":
				g.Attach(source.ID(), other.ID())
			case "phase":
				g.PhaseOut(host.ID())
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
				g.PutOnBattlefield(host.Card, a.PlayerID())
			case "tap":
				g.TapPermanent(g.FindPermanent(host.ID()))
			case "source return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(auraCard, a.PlayerID())
			}
			clone := g.Clone()
			if err := tx.Commit(); err == nil {
				t.Fatal("stale attached cost paid")
			}
			if a.Life() != 20 || g.FindPermanent(other.ID()).Tapped || clone.FindPermanent(other.ID()).Tapped {
				t.Fatal("failed payment changed live state or clone")
			}
		})
	}
}

func TestTapAttachedPermanentCostDeclaration(t *testing.T) {
	valid := NewAura("Valid", "", WithAbility(Activate(Costs(TapAttachedPermanentCost()), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	invalid := NewInstant("Invalid", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)))), WithAdditionalCost(TapAttachedPermanentCost()))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil || !strings.Contains(err.Error(), "battlefield") {
		t.Fatalf("invalid cost validation = %v", err)
	}
}

func TestTapAttachedPermanentCostAbsentAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	host := g.PutOnBattlefield(NewLand("Host"), a.PlayerID())
	source := g.PutOnBattlefield(NewAura("Source", "", WithCastTarget(TargetLand())), a.PlayerID())
	cost := TapAttachedPermanentCost()
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}}); err == nil {
		t.Fatal("unattached source paid")
	}
	g.Attach(source.ID(), host.ID())
	g.PhaseOut(host.ID())
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}}); err == nil {
		t.Fatal("phased host paid")
	}
	g.PhaseIn([]uuid.UUID{host.ID()})
	source = g.PutOnBattlefield(NewAura("Replacement source", "", WithCastTarget(TargetLand())), a.PlayerID())
	g.Attach(source.ID(), host.ID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(host.ID()).Tapped || clone.FindPermanent(host.ID()).Tapped {
		t.Fatal("payment changed cloned host")
	}
}

// --- from composable_captured_untap_test.go ---
func TestCapturedUntapControlAndExpiry(t *testing.T) {
	for _, change := range []string{"none", "control", "phase", "return", "untapped", "other restriction"} {
		t.Run(change, func(t *testing.T) {
			g, a, b, p := controllerTestGame(t)
			id := p.ID()
			p.Tapped = true
			ctx := &EffectContext{Game: g, Controller: a.PlayerID(), SourceID: id}
			for range 2 {
				if err := SkipNextUntapOf(This, You).Apply(ctx); err != nil {
					t.Fatal(err)
				}
			}
			switch change {
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: id, ControllerID: b.PlayerID(), Duration: Indefinite})
			case "phase":
				g.PhaseOut(id)
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(p.Card, a.PlayerID()).Tapped = true
			case "untapped":
				g.UntapPermanent(p)
			case "other restriction":
				g.MutablePermanent(id).GrantBaseAttr(AttrDoesNotUntap)
			}
			g.turns.SetActivePlayerIndex(1)
			g.doUntap()
			if change == "control" && g.FindPermanent(id).Tapped {
				t.Fatal("captured A restriction blocked B's untap")
			}
			g.turns.SetActivePlayerIndex(0)
			g.doUntap()
			if change == "none" && !g.FindPermanent(id).Tapped {
				t.Fatal("A's next untap was not blocked")
			}
			if change == "return" && g.FindPermanent(id).Tapped {
				t.Fatal("restriction followed returned card")
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{id})
			}
			if change == "other restriction" {
				g.MutablePermanent(id).RevokeBaseAttr(AttrDoesNotUntap)
			}
			g.MutablePermanent(id).Tapped = true
			if change == "control" {
				g.turns.SetActivePlayerIndex(1)
			}
			g.doUntap()
			if g.FindPermanent(id).Tapped {
				t.Fatal("restriction did not expire during A's step")
			}
		})
	}
}

func TestCapturedUntapCloneAndIndependentPlayers(t *testing.T) {
	g, a, b, p := controllerTestGame(t)
	p.Tapped = true
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID()}
	if err := SkipNextUntapOf(This, You).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	ctx.Controller = b.PlayerID()
	if err := SkipNextUntapOf(This, You).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.doUntap()
	clone.doUntap()
	if clone.FindPermanent(p.ID()).Tapped {
		t.Fatal("clone restriction did not expire")
	}
	g.doUntap()
	if !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("clone consumed original restriction")
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.turns.SetActivePlayerIndex(1)
	g.doUntap()
	if !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("A's step consumed B's restriction")
	}
	g.doUntap()
	if g.FindPermanent(p.ID()).Tapped {
		t.Fatal("B's restriction did not expire")
	}
}

func TestCapturedUntapAbsentAndValidation(t *testing.T) {
	g, a, _, p := controllerTestGame(t)
	ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: uuid.New()}
	if err := SkipNextUntapOf(This, You).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	ctx.Controller = a.PlayerID()
	saved := BindPermanent("missing")
	if err := SkipNextUntapOf(From(saved), You).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	p.Tapped = true
	g.doUntap()
	if g.FindPermanent(p.ID()).Tapped {
		t.Fatal("absent reference installed restriction")
	}
	player := BindPlayer("player")
	for _, body := range []ResolvingEffect{
		SkipNextUntapOf(nil, You), SkipNextUntapOf(This, nil),
		SkipNextUntapOf(This, FromPlayer(player)),
		SkipNextUntapOf(DeclarePermanentTarget("target", IsCreature), You),
	} {
		card := NewInstant("Invalid", "", WithAbility(Spell(body)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid definition")
		}
	}
}

func TestCapturedUntapTargetSourceExitAndExistingRestriction(t *testing.T) {
	g, a, b, p := controllerTestGame(t)
	p.Tapped = true
	target := DeclarePermanentTarget("creature", IsCreature)
	body := SkipNextUntapOf(target, You)
	ability := Activate(Costs(), body, WithTargets(target))
	card := NewArtifact("Source", "", WithAbility(ability))
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: source.ID(), Card: card, IsAbility: true, Controller: a.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	g.RemoveFromBattlefield(source)
	g.ResolveStack()
	g.SkipNextUntap(p.ID())
	g.doUntap()
	if !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("source departure removed restriction")
	}
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: Indefinite})
	g.turns.SetActivePlayerIndex(1)
	g.doUntap()
	if g.FindPermanent(p.ID()).Tapped {
		t.Fatal("overlapping restrictions did not both expire")
	}
	if body.Properties().Outcome != OutcomeDetriment {
		t.Fatal("missing detriment metadata")
	}
}

func TestCapturedUntapSkippedStep(t *testing.T) {
	g, a, _, p := controllerTestGame(t)
	p.Tapped = true
	if err := SkipNextUntapOf(This, You).Apply(&EffectContext{Game: g, Controller: a.PlayerID(), SourceID: p.ID()}); err != nil {
		t.Fatal(err)
	}
	g.turns.BuildNextTurn()
	g.SkipNextOccurrenceOfStep(Untap)
	step, ok := g.turns.PopNextStep()
	if !ok || step != Upkeep {
		t.Fatalf("next step = %v, %v", step, ok)
	}
	g.doUpkeepActions()
	g.turns.BuildNextTurn()
	step, ok = g.turns.PopNextStep()
	if !ok || step != Untap {
		t.Fatalf("next step = %v, %v", step, ok)
	}
	g.doUntap()
	if !g.FindPermanent(p.ID()).Tapped {
		t.Fatal("skipped step consumed restriction")
	}
	g.doUntap()
	if g.FindPermanent(p.ID()).Tapped {
		t.Fatal("restriction survived actual untap step")
	}
}

// --- from composable_granted_activation_test.go ---
func TestStaticGrantedActivation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	rule := GrantEachActivatedAbility(WithoutPermanent(MatchingPermanents(HasSubType("Zombie")), This), Costs(ManaPayment("{B}")), Regenerate(This))
	source := NewCreature("Grant source", "", 2, 3, WithSubTypes("Zombie"), WithAbility(Static(rule)))
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	recipient := NewArtifact("Recipient", "", WithSubTypes("Zombie"))
	recipient.SetOwner(b.PlayerID())
	g.PutOnBattlefield(recipient, b.PlayerID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(source.ID()).RuntimeAbilities) != 1 {
		t.Fatal("source received excluded ability")
	}
	abilities := g.FindPermanent(recipient.ID()).RuntimeAbilities
	if len(abilities) != 1 {
		t.Fatalf("noncreature recipient has %d abilities", len(abilities))
	}
	ability := UnwrapAbility(abilities[0]).(ActivatedAbility)
	if ability.Source() != recipient.ID() || ability.Controller() != b.PlayerID() {
		t.Fatal("grant used donor context")
	}
	if !IsRegenerationEffect(ability.Effects()[0]) {
		t.Fatal("missing regeneration metadata")
	}
	clone := g.Clone()
	clone.RemoveFromBattlefield(clone.FindPermanent(source.ID()))
	clone.ApplyContinuousEffects()
	if len(clone.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 || len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 1 {
		t.Fatal("grant clone isolation failed")
	}
	g.PhaseOut(source.ID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(recipient.ID()).RuntimeAbilities) != 0 {
		t.Fatal("phased source retained grant")
	}
}

func TestStaticGrantedActivationValidation(t *testing.T) {
	saved := BindPermanent("outside")
	for _, body := range []ResolvingEffect{Regenerate(From(saved)), Regenerate(DeclarePermanentTarget("undeclared", IsCreature)), nil} {
		card := NewEnchantment("Invalid grant", "", WithAbility(Static(GrantEachActivatedAbility(MatchingPermanents(IsCreature), Costs(), body))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "abilities[0]") {
			t.Fatalf("missing grant diagnostic: %v", err)
		}
	}
	card := NewEnchantment("Valid grant", "", WithAbility(Static(GrantEachActivatedAbility(WithoutPermanent(MatchingPermanents(IsCreature), This), Costs(), Regenerate(This)))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

func TestStaticGrantedManaActivation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewEnchantment("Mana grant", "", WithAbility(Static(GrantEachActivatedAbility(ControlledPermanents(IsLand), Costs(TapCost(This)), AddManaAmount(Blue, Fixed(1))))))
	g.PutOnBattlefield(source, a.PlayerID())
	first, second, opponent := NewLand("First"), NewLand("Second"), NewLand("Opponent")
	for _, card := range []Card{first, second} {
		g.PutOnBattlefield(card, a.PlayerID())
	}
	g.PutOnBattlefield(opponent, b.PlayerID())
	g.ApplyContinuousEffects()
	if len(g.FindPermanent(opponent.ID()).RuntimeAbilities) != 0 {
		t.Fatal("controlled set included opponent")
	}
	one := UnwrapAbility(g.FindPermanent(first.ID()).RuntimeAbilities[0]).(ActivatedAbility)
	two := UnwrapAbility(g.FindPermanent(second.ID()).RuntimeAbilities[0]).(ActivatedAbility)
	if one == two || one.AbilityID() == two.AbilityID() || one.Source() != first.ID() || two.Source() != second.ID() {
		t.Fatal("recipients share runtime ability state")
	}
	if !g.CanAfford(a.PlayerID(), ManaCost{Blue: 2}, nil) {
		t.Fatal("planner cannot use granted mana")
	}
	clone := g.Clone()
	if err := clone.AutoTapForCost(a.PlayerID(), ManaCost{Blue: 2}); err != nil {
		t.Fatal(err)
	}
	if clone.GetStack().Size() != 0 || clone.GetPlayer(a.PlayerID()).ManaPool().Count(Blue) != 2 || a.ManaPool().Count(Blue) != 0 || g.FindPermanent(first.ID()).Tapped {
		t.Fatal("granted mana timing or clone isolation failed")
	}
}

func TestWithoutPermanentAbsentAndIdentity(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Subject", "", 1, 1)
	g.PutOnBattlefield(card, a.PlayerID())
	saved := BindPermanent("excluded")
	set := WithoutPermanent(MatchingPermanents(IsCreature), From(saved))
	ctx := &EffectContext{Game: g, SourceID: card.ID(), Controller: a.PlayerID()}
	if len(set.resolvePermanents(ctx)) != 1 {
		t.Fatal("absent reference removed a member")
	}
	if err := Snapshot(This, saved).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if len(set.resolvePermanents(ctx)) != 0 {
		t.Fatal("saved reference did not exclude member")
	}
	g.RemoveFromBattlefield(g.FindPermanent(card.ID()))
	g.PutOnBattlefield(card, a.PlayerID())
	if len(set.resolvePermanents(ctx)) != 1 {
		t.Fatal("old reference excluded returned object")
	}
}

// --- from composable_legacy_payment_test.go ---
func TestLegacyPaymentDefinitionInspection(t *testing.T) {
	paid := DrawFor(You, Fixed(2))
	unpaid := GainLifeFor(You, Fixed(3))
	effect := IfPlayerPays(SelectController(), ManaCostOf("{1}"), "Pay?", paid, unpaid)
	info := InspectDefinition(effect)
	if info.Role != "legacy-payment-branch" || info.Text != "Pay?" || len(info.Children) != 4 {
		t.Fatalf("payment metadata = %+v", info)
	}
	if info.Children[2].Properties.DrawCount != 2 || info.Children[3].Properties.LifeGain != 3 {
		t.Fatalf("branch metadata = %+v", info.Children)
	}
	if info.Properties.DrawCount != 2 || info.Properties.LifeGain != 3 {
		t.Fatalf("summary = %+v", info.Properties)
	}
	unless := InspectDefinition(UnlessTargetPays(SelectController(), ManaCostOf("{1}"), "Pay?", paid))
	if len(unless.Children) != 3 || unless.Children[2].Properties.DrawCount != 2 {
		t.Fatalf("unless metadata = %+v", unless)
	}
}

func TestLegacyPaymentDefinitionValidation(t *testing.T) {
	target := DeclarePermanentTarget("creature", IsCreature)
	saved := BindPermanent("saved")
	var missing *sequenceEffect
	payment := func(paid, unpaid Effect) Effect {
		return IfPlayerPays(SelectController(), ManaCostOf("{1}"), "Pay?", paid, unpaid)
	}
	tests := []struct {
		name    string
		effect  Effect
		path    string
		message string
	}{
		{"paid target", payment(Destroy(target), nil), ".paid.subject", "target binding requires"},
		{"unpaid target", UnlessTargetPays(SelectController(), ManaCostOf("{1}"), "Pay?", Destroy(target)), ".unpaid.subject", "target binding requires"},
		{"paid saved read", payment(Destroy(From(saved)), nil), ".paid.subject", "producer before this read"},
		{"unpaid saved read", payment(nil, Destroy(From(saved))), ".unpaid.subject", "producer before this read"},
		{"sibling assignment", payment(Snapshot(This, saved), Destroy(From(saved))), ".unpaid.subject", "producer before this read"},
		{"escaped assignment", CompositeEffects("sequence", payment(Snapshot(This, saved), nil), Destroy(From(saved))), ".effects[1].subject", "producer before this read"},
		{"nil payer", IfPlayerPays(nil, ManaCostOf("{1}"), "Pay?", nil, nil), ".payer", "non-nil expression"},
		{"nil cost", IfPlayerPays(SelectController(), nil, "Pay?", nil, nil), ".cost", "non-nil expression"},
		{"typed nil paid", payment(missing, nil), ".paid", "non-nil expression"},
		{"typed nil unpaid", payment(nil, missing), ".unpaid", "non-nil expression"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			card := NewArtifact("Payment", "", WithAbility(NewActivatedAbility(test.effect, ManaCostOf("{0}"))))
			err := ValidateCardDefinition("Payment", card)
			if err == nil || !strings.Contains(err.Error(), "Payment: abilities[0].") || !strings.Contains(err.Error(), test.path) || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("diagnostic = %v, want path %s and %s", err, test.path, test.message)
			}
		})
	}
}

func TestLegacyPaymentDefinitionValidScopes(t *testing.T) {
	saved := BindPermanent("saved")
	payment := func(paid, unpaid Effect) Effect {
		return IfPlayerPays(SelectController(), ManaCostOf("{1}"), "Pay?", paid, unpaid)
	}
	for _, effect := range []Effect{
		payment(nil, nil),
		payment(Sequence(Snapshot(This, saved), Destroy(From(saved))), nil),
		payment(nil, Sequence(Snapshot(This, saved), Destroy(From(saved)))),
		CompositeEffects("outer snapshot", Snapshot(This, saved), payment(Destroy(From(saved)), Destroy(From(saved)))),
	} {
		card := NewArtifact("Payment", "", WithAbility(NewActivatedAbility(effect, ManaCostOf("{0}"))))
		if err := ValidateCardDefinition("Payment", card); err != nil {
			t.Fatal(err)
		}
	}
	target := DeclarePermanentTarget("creature", IsCreature)
	card := NewInstant("Payment", "", WithAbility(Spell(payment(Destroy(target), nil), WithTargets(target))))
	if err := ValidateCardDefinition("Payment", card); err != nil {
		t.Fatal(err)
	}
	bad := NewInstant("Payment", "", WithAbility(Spell(payment(Sacrifice(This), nil))))
	if err := ValidateCardDefinition("Payment", bad); err == nil || !strings.Contains(err.Error(), ".paid.subject: battlefield source") {
		t.Fatalf("spell source diagnostic = %v", err)
	}
}

// --- from composable_mana_choice_test.go ---
func TestAddManaOfAnyOneColorChoiceAndEvent(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	addTappedForManaWatcher(g, a)
	ability := Activate(Costs(TapCost(This)), AddManaOfAnyOneColor(Fixed(2)))
	source := g.PutOnBattlefield(NewLand("Choice source", WithAbility(ability)), a.PlayerID())
	if err := ValidateCardDefinition("Choice source", source.Card); err != nil {
		t.Fatal(err)
	}
	productions := ManaProductionsForAbility(ability)
	if len(productions) != 1 || productions[0].Color != AnyColor || productions[0].Amount != 2 || productions[0].AnyCombination {
		t.Fatalf("planner metadata = %+v", productions)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(White) != 2 || g.StackSize() != 0 {
		t.Fatal("choice did not add both mana immediately")
	}
	resolvePendingTriggers(g)
	if a.Life() != 21 {
		t.Fatal("tap-for-mana event was not emitted once")
	}
}

func TestAddManaOfAnyOneColorPlanningAndRollback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := Activate(Costs(TapCost(This)), AddManaOfAnyOneColor(Fixed(2)))
	source := g.PutOnBattlefield(NewLand("Choice source", WithAbility(ability)), a.PlayerID())
	if !g.CanAfford(a.PlayerID(), ManaCost{Green: 2}, nil) {
		t.Fatal("planner missed chosen-color production")
	}
	if g.CanAfford(a.PlayerID(), ManaCost{Green: 1, Blue: 1}, nil) {
		t.Fatal("planner split one color choice")
	}
	clone := g.Clone()
	if err := clone.AutoTapForCost(a.PlayerID(), ManaCost{Green: 2}); err != nil {
		t.Fatal(err)
	}
	if !clone.FindPermanent(source.ID()).Tapped || source.Tapped || a.ManaPool().Count(Green) != 0 {
		t.Fatal("planned mana leaked across clones")
	}
	if err := g.AutoTapForCost(a.PlayerID(), ManaCost{Green: 1, Blue: 1}); err == nil {
		t.Fatal("split-color payment succeeded")
	}
	if source.Tapped || a.ManaPool().Count(Green) != 0 || a.ManaPool().Count(Blue) != 0 {
		t.Fatal("failed payment changed source or pool")
	}
}

func TestAddManaOfAnyOneColorValidation(t *testing.T) {
	for _, amount := range []ResolutionValue{nil, FromNumber(BindNumber("missing"))} {
		card := NewSorcery("Invalid", "", Spell(AddManaOfAnyOneColor(amount)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid amount accepted")
		}
	}
	ability := Activate(Costs(TapCost(This)), AddManaOfAnyOneColor(PowerOf(This)))
	if !IsActivatedManaAbility(ability) || ManaProductionsForAbility(ability) != nil {
		t.Fatal("dynamic amount was misclassified or advertised to planner")
	}
}

func TestAddManaOfAnyOneColorDynamicAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	ability := Activate(Costs(TapCost(This)), AddManaOfAnyOneColor(PowerOf(This)))
	source := g.PutOnBattlefield(NewCreature("Dynamic source", "", 3, 3, WithAbility(ability), WithKeyword(Haste)), a.PlayerID())
	clone := g.Clone()
	if err := clone.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if clone.GetPlayer(a.PlayerID()).ManaPool().Count(White) != 3 || a.ManaPool().Count(White) != 0 || source.Tapped {
		t.Fatal("dynamic choice leaked across clones")
	}
	spell := NewInstant("Choice spell", "", Spell(AddManaOfAnyOneColor(Fixed(2))))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.ResolveStack()
	if a.ManaPool().Count(White) != 2 || b.ManaPool().Count(White) != 2 {
		t.Fatal("copied mana instruction lost its resolving controller")
	}
}

// --- from composable_mana_recipient_test.go ---
func TestAddManaToPlayerUsesSavedRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("producer")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithPlayer(player), AddManaToPlayer(FromPlayer(player), Green, Fixed(2)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Blue, Fixed(1)))))
	g.PutOnBattlefield(land, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if got := b.ManaPool().Count(Green); got != 2 {
		t.Fatalf("immediate recipient green mana = %d, want 2", got)
	}
	g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
	g.PutOnBattlefield(land, a.PlayerID())
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if got := clone.GetPlayer(b.PlayerID()).ManaPool().Count(Green); got != 2 {
		t.Fatalf("clone recipient green mana = %d, want 2", got)
	}
	g.ResolveStack()
	if got := b.ManaPool().Count(Green); got != 2 {
		t.Fatalf("recipient green mana = %d, want 2", got)
	}
	if got := a.ManaPool().Count(Green); got != 0 {
		t.Fatalf("controller green mana = %d, want 0", got)
	}
}

func TestAddManaToPlayerAbsenceAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if err := ApplyEffect(g, AddManaToPlayer(FromPlayer(BindPlayer("missing")), Green, Fixed(2)),
		uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := a.ManaPool().Count(Green); got != 0 {
		t.Fatalf("absent recipient added %d mana", got)
	}
	for _, effect := range []ResolvingEffect{
		AddManaToPlayer(nil, Green, Fixed(1)),
		AddManaToPlayer(You, AnyColor, Fixed(1)),
		AddManaToPlayer(You, Green, nil),
	} {
		card := NewSorcery("Bad", "", Spell(effect))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid mana recipient action")
		}
	}
	card := NewSorcery("Unbound", "", Spell(AddManaToPlayer(FromPlayer(BindPlayer("missing")), Green, Fixed(1))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("accepted unbound player")
	}
	if got := b.ManaPool().Count(Green); got != 0 {
		t.Fatalf("other player received %d mana", got)
	}
}

func TestAddManaToPlayerActivatedManaClassification(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Shared", WithAbility(Activate(Costs(TapCost(This)),
		AddManaToPlayer(ActivePlayer, Red, Fixed(1)))))
	g.PutOnBattlefield(land, a.PlayerID())
	if !IsActivatedManaAbility(land.Abilities()[0]) {
		t.Fatal("targetless player mana action was not a mana ability")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 0 || a.ManaPool().Count(Red) != 1 || b.ManaPool().Count(Red) != 0 {
		t.Fatal("mana activation did not resolve immediately for its recipient")
	}
}

// --- from composable_mana_types_test.go ---
type producedTypeChoicePlayer struct {
	*BasePlayer
	choice Color
	calls  int
}

func (p *producedTypeChoicePlayer) ChooseManaColor(string) Color {
	p.calls++
	return p.choice
}

func TestLandManaTypesCapturedForTriggeredMana(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("producer")
	types := BindManaTypes("produced types")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithPlayer(player).WithManaTypes(types),
		AddManaOfProducedTypeToPlayer(FromPlayer(player), types, Fixed(1)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(Blue, Fixed(2)))))
	g.PutOnBattlefield(land, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if got := b.ManaPool().Count(Blue); got != 3 || g.StackSize() != 0 {
		t.Fatalf("blue mana = %d, stack = %d; want 3 and 0", got, g.StackSize())
	}
}

func TestLandManaTypesDeclaration(t *testing.T) {
	types := BindManaTypes("types")
	for _, pattern := range []landTappedForManaPattern{
		LandTappedForMana().WithManaTypes(nil),
		LandTappedForMana().WithManaTypes(BindManaTypes("")),
		LandTappedForMana().WithManaTypes(types).WithManaTypes(types),
	} {
		card := NewEnchantment("Bad", "", WithAbility(WhenEvent(pattern,
			AddManaOfProducedTypeToPlayer(You, types, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid type binding")
		}
	}
	card := NewEnchantment("Unbound", "", WithAbility(WhenEvent(LandTappedForMana(),
		AddManaOfProducedTypeToPlayer(You, types, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("accepted unbound type set")
	}
	for _, effect := range []ResolvingEffect{
		AddManaOfProducedTypeToPlayer(nil, types, Fixed(1)),
		AddManaOfProducedTypeToPlayer(You, nil, Fixed(1)),
		AddManaOfProducedTypeToPlayer(You, types, nil),
	} {
		card := NewEnchantment("Bad action", "", WithAbility(WhenEvent(
			LandTappedForMana().WithManaTypes(types), effect)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid mana action")
		}
	}
}

func TestLandManaTypesMultiTypeChoiceAndClone(t *testing.T) {
	a := NewBasePlayer("A")
	b := &producedTypeChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: Red}
	g := NewGame(a, b)
	player := BindPlayer("producer")
	types := BindManaTypes("produced types")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithPlayer(player).WithManaTypes(types),
		AddManaOfProducedTypeToPlayer(FromPlayer(player), types, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), Sequence(
		AddManaAmount(Blue, Fixed(1)), AddManaAmount(Red, Fixed(1))))))
	g.PutOnBattlefield(land, b.PlayerID())
	clone := g.Clone()
	if err := clone.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(land.ID()).Tapped || b.ManaPool().Count(Red) != 0 || clone.GetPlayer(b.PlayerID()).ManaPool().Count(Blue) != 2 {
		t.Fatal("clone mana choice changed original or missed produced type")
	}
	if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if b.ManaPool().Count(Red) != 2 || b.ManaPool().Count(Blue) != 1 || b.calls != 1 {
		t.Fatal("mana type choice did not use the produced set")
	}
}

func TestLandManaTypesInvalidChoiceAndSourcePhasing(t *testing.T) {
	a := &producedTypeChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: Green}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	types := BindManaTypes("types")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithManaTypes(types), AddManaOfProducedTypeToPlayer(You, types, Fixed(1)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), Sequence(
		AddManaAmount(Blue, Fixed(1)), AddManaAmount(Red, Fixed(1))))))
	g.PutOnBattlefield(land, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(Blue) != 1 || a.ManaPool().Count(Green) != 0 {
		t.Fatal("invalid color choice produced a type absent from the event")
	}
	g.FindPermanent(watcher.ID()).PhasedOut = true
	g.FindPermanent(land.ID()).Tapped = false
	if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(Blue) != 1 {
		t.Fatal("phased trigger source produced mana")
	}
}

// --- from composable_mill_cost_test.go ---
func TestMillCostOptionalPayment(t *testing.T) {
	for _, count := range []int{0, 1, 2, 3} {
		a := &acceptingMayPlayer{NewBasePlayer("A")}
		g := NewGame(a, NewBasePlayer("B"))
		for range count {
			card := NewLand("Library card")
			card.SetOwner(a.PlayerID())
			a.AddToLibrary(card)
		}
		if err := ApplyEffect(g, UnlessPaid(You, Costs(MillCost(2)), GainLifeFor(You, Fixed(3))), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if want := max(0, count-2); count >= 2 && len(a.Library()) != want {
			t.Fatalf("%d cards: library = %d, want %d", count, len(a.Library()), want)
		}
		if count < 2 && len(a.Library()) != count {
			t.Fatalf("%d cards: failed cost changed library", count)
		}
		wantLife := 20
		if count < 2 {
			wantLife = 23
		}
		if a.Life() != wantLife {
			t.Fatalf("%d cards: life = %d, want %d", count, a.Life(), wantLife)
		}
	}
}

func TestMillCostDeclinedPayment(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	for range 2 {
		card := NewLand("Library card")
		card.SetOwner(a.PlayerID())
		a.AddToLibrary(card)
	}
	if err := ApplyEffect(g, UnlessPaid(You, Costs(MillCost(2)), GainLifeFor(You, Fixed(3))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 2 || a.Life() != 23 {
		t.Fatal("declined cost changed library or missed unpaid branch")
	}
}

func TestMillCostReplacementAndRollback(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	for range 3 {
		card := NewLand("Library card")
		card.SetOwner(a.PlayerID())
		a.AddToLibrary(card)
	}
	source := NewCreature("Modifier", "", 1, 1)
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddMillReplacement(source.ID(), func(_ *Game, _ uuid.UUID, n int) int { return n * 2 })
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), MillCost(2)}}); err == nil {
		t.Fatal("replacement increased cost beyond available library")
	}
	if a.Life() != 20 || len(a.Library()) != 3 {
		t.Fatal("failed preparation changed live state")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), MillCost(2)}})
	if err != nil {
		t.Fatal(err)
	}
	a.SetLibrary(a.Library()[:1])
	if err := tx.Commit(); err == nil {
		t.Fatal("stale library paid cost")
	}
	if a.Life() != 20 || len(a.Library()) != 1 {
		t.Fatal("failed commit changed live state")
	}
}

func TestMillCostReplacementChangeAndClone(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	for range 4 {
		card := NewLand("Library card")
		card.SetOwner(a.PlayerID())
		a.AddToLibrary(card)
	}
	source := NewCreature("Modifier", "", 1, 1)
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddMillReplacement(source.ID(), func(_ *Game, _ uuid.UUID, n int) int { return n * 2 })
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{MillCost(2)}})
	if err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.GetPlayer(a.PlayerID()).SetLibrary(clone.GetPlayer(a.PlayerID()).Library()[:2])
	if len(a.Library()) != 4 {
		t.Fatal("clone changed original library")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	if err := tx.Commit(); err == nil {
		t.Fatal("changed replacement paid stale locked cost")
	}
	if len(a.Library()) != 4 {
		t.Fatal("failed commit milled cards")
	}
	if err := ApplyEffect(g, UnlessPaid(You, Costs(MillCost(2)), GainLifeFor(You, Fixed(3))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 2 || a.Life() != 20 {
		t.Fatal("new payment did not use current mill rule")
	}
}

func TestMillCostZeroReplacement(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	source := NewCreature("Modifier", "", 1, 1)
	source.SetOwner(a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddMillReplacement(source.ID(), func(_ *Game, _ uuid.UUID, _ int) int { return 0 })
	if err := ApplyEffect(g, UnlessPaid(You, Costs(MillCost(2)), GainLifeFor(You, Fixed(3))), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 {
		t.Fatal("replacement made zero-card payment impossible")
	}
}

func TestMillCostMultipleComponentsRecheck(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	for range 4 {
		card := NewLand("Library card")
		card.SetOwner(a.PlayerID())
		a.AddToLibrary(card)
	}
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), MillCost(2), MillCost(2)}})
	if err != nil {
		t.Fatal(err)
	}
	a.SetLibrary(a.Library()[:3])
	if err := tx.Commit(); err == nil {
		t.Fatal("multiple mill components paid with too few cards")
	}
	if a.Life() != 20 || len(a.Library()) != 3 {
		t.Fatal("failed payment changed resources")
	}
}

func TestMillCostDeclaration(t *testing.T) {
	valid := NewCreature("Valid", "", 1, 1, WithAbility(At(YourUpkeep, UnlessPaid(You, Costs(MillCost(2)), Sacrifice(This)))))
	if err := ValidateCardDefinition("Valid", valid); err != nil {
		t.Fatal(err)
	}
	for _, n := range []int{-1, 0} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(At(YourUpkeep, UnlessPaid(You, Costs(MillCost(n)), Sacrifice(This)))))
		if err := ValidateCardDefinition("Invalid", card); err == nil || !strings.Contains(err.Error(), "positive") {
			t.Fatalf("count %d: validation = %v", n, err)
		}
	}
}

// --- from composable_optional_payment_availability_test.go ---
type countedMayPlayer struct {
	*BasePlayer
	accept bool
	offers int
}

func (p *countedMayPlayer) ChooseMayAbility(string) bool {
	p.offers++
	return p.accept
}

func TestUnlessPaidOffersOnlyPossibleSacrifice(t *testing.T) {
	for _, tc := range []struct {
		name       string
		cards      int
		permanents int
		other      bool
		phased     bool
		accept     bool
		wantOffers int
		wantHand   int
		wantBoard  int
	}{
		{"no permanent", 2, 0, false, false, true, 0, 0, 0},
		{"opponent permanent", 2, 1, true, false, true, 0, 0, 1},
		{"phased permanent", 2, 1, false, true, true, 0, 0, 0},
		{"pay once then discard", 2, 1, false, false, true, 1, 1, 0},
		{"decline each time", 2, 1, false, false, false, 2, 0, 1},
		{"no drawn cards", 0, 1, false, false, true, 0, 0, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: tc.accept}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			for range tc.cards {
				a.AddToLibrary(NewArtifact("Drawn", ""))
			}
			for range tc.permanents {
				controller := a.PlayerID()
				if tc.other {
					controller = b.PlayerID()
				}
				permanent := g.PutOnBattlefield(NewLand("Land"), controller)
				permanent.PhasedOut = tc.phased
			}
			cards, member := BindCards("drawn"), BindCard("member")
			effect := Sequence(
				DrawFor(You, Fixed(tc.cards), DrawCardsInto(cards)),
				ForEachSavedCard(cards, member, UnlessPaid(You,
					Costs(SacrificeMatchingIncludingSourceCost(AnyPermanent, "Sacrifice a permanent")),
					Discard(You, Fixed(1)))),
			)
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			board := len(g.FilterBattlefield(AnyPermanent))
			if a.offers != tc.wantOffers || len(a.Hand()) != tc.wantHand || board != tc.wantBoard {
				t.Fatalf("offers = %d, hand = %d, battlefield = %d", a.offers, len(a.Hand()), board)
			}
		})
	}
}

func TestUnlessPaidPerDrawnCardDeclaration(t *testing.T) {
	cards, member := BindCards("drawn"), BindCard("member")
	body := Sequence(DrawFor(You, Fixed(2), DrawCardsInto(cards)),
		ForEachSavedCard(cards, member, UnlessPaid(You,
			Costs(SacrificeMatchingIncludingSourceCost(AnyPermanent, "Sacrifice a permanent")),
			Discard(You, Fixed(1)))))
	for _, card := range []*BaseCard{
		NewInstant("Draw spell", "", Spell(body)),
		NewArtifact("Draw artifact", "", WithAbility(Activate(Costs(), body))),
		NewArtifact("Draw trigger", "", WithAbility(WhenEvent(Enters(This), body))),
	} {
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

// --- from composable_optional_payment_result_test.go ---
type optionalPaymentPlayer struct {
	*BasePlayer
	accept bool
}

func (p *optionalPaymentPlayer) ChooseMayAbility(string) bool { return p.accept }

func TestOptionalPaymentResultInstances(t *testing.T) {
	a, b := &acceptingMayPlayer{NewBasePlayer("A")}, NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("paid")
	source := NewArtifact("Source", "", WithAbility(Activate(Costs(
		OptionalCostInto(LifePayCost(2), "Pay two life?", paid)),
		GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 18 {
		t.Fatalf("payment life = %d", a.Life())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 19 || clone.GetPlayer(b.PlayerID()).Life() != 21 || a.Life() != 18 {
		t.Fatal("optional result did not stay in its stack and game instances")
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 21 {
		t.Fatal("optional result escaped its payment instance")
	}
}

func TestOptionalPaymentResultDeclinedAndFailure(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	source := NewArtifact("Source", "", WithAbility(Activate(Costs(
		OptionalCostInto(LifePayCost(25), "Pay life?", paid)),
		GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatalf("declined or impossible payment changed life: %d", a.Life())
	}
	if err := ValidateCardDefinition("Nil", NewArtifact("Nil", "", WithAbility(Activate(Costs(
		OptionalCostInto(LifePayCost(1), "Pay?", nil)), GainLifeFor(You, Fixed(1)))))); err == nil || !strings.Contains(err.Error(), "non-nil") {
		t.Fatalf("nil binding validation = %v", err)
	}
	if err := ValidateCardDefinition("Wrong", NewArtifact("Wrong", "", WithAbility(Activate(Costs(
		OptionalCostInto(LifePayCost(1), "Pay?", paid)), IfCardHasType(FromCard(BindCard("card")), TypeCreature, GainLifeFor(You, Fixed(1))))))); err == nil {
		t.Fatal("unbound card was accepted")
	}
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{OptionalCostInto(LifePayCost(1), "", paid), "prompt"},
		{OptionalCostInto(LifePayCost(1), "Pay?", BindNumber("")), "name"},
		{OptionalCostInto(RevealFromHandCostWithCard(IsCreatureCard, "Reveal", BindCard("card")), "Reveal?", paid), "captured"},
		{OptionalCostInto(EitherCost(LifePayCost(1), LifePayCost(2)), "Pay?", paid), "direct"},
	} {
		card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("validation = %v; want %s", err, tc.want)
		}
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(
		OptionalCostInto(LifePayCost(1), "First?", paid),
		OptionalCostInto(LifePayCost(1), "Second?", paid)), GainLifeFor(You, FromNumber(paid)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate binding = %v", err)
	}
}

func TestOptionalPaymentResultPendingAndRollback(t *testing.T) {
	a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	source := NewArtifact("Source", "", WithAbility(Activate(Costs(
		OptionalCostInto(LifePayCost(2), "Pay life?", paid)),
		GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	a.accept = false
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 19 {
		t.Fatalf("pending optional outcomes = %d", a.Life())
	}
	a.accept = true
	failed := NewArtifact("Failed", "", WithAbility(Activate(Costs(
		OptionalCostInto(LifePayCost(2), "Pay life?", paid), LifePayCost(25)),
		GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(failed, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), failed.ID(), 0, nil); err == nil {
		t.Fatal("unpayable total cost was accepted")
	}
	if a.Life() != 19 || len(g.StackObjects()) != 0 {
		t.Fatal("failed total cost changed live payment state")
	}
}

func TestOptionalPaymentResultSpellAdditionalCost(t *testing.T) {
	a := &acceptingMayPlayer{NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	spell := NewInstant("Optional", "", WithAbility(Spell(GainLifeFor(You, FromNumber(paid)))),
		WithAdditionalCost(OptionalCostInto(LifePayCost(2), "Pay life?", paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 19 {
		t.Fatalf("spell payment result = %d", a.Life())
	}
}

// --- from composable_paid_card_test.go ---
func TestPaidSacrificeCardSpellAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("paid")
	spell := NewInstant("Offering", "", WithAbility(Spell(IfCardHasType(FromCard(paid), TypeCreature, GainLifeFor(You, Fixed(2))))),
		WithAdditionalCost(SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	creature := NewCreature("Paid", "", 1, 1)
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("paid card escaped a stack instance")
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("paid card was not available to both stack copies")
	}
}

func TestPaidSacrificeCardVisitAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("paid")
	spell := NewInstant("Visit", "", WithAbility(Spell(Sequence(
		ReturnCardToHand(FromCard(paid)),
		IfCardHasType(FromCard(paid), TypeCreature, GainLifeFor(You, Fixed(2))),
	))), WithAdditionalCost(SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	creature := NewCreature("Paid", "", 1, 1)
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 20 || len(a.Hand()) != 1 || a.Hand()[0].ID() != creature.ID() {
		t.Fatal("saved card followed a later zone visit")
	}

	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid), ""},
		{SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", nil), "non-nil"},
		{SacrificeMatchingCostWithCard(IsCreature, "", paid), "text"},
	} {
		card := NewInstant("Offering", "", WithAbility(Spell(ReturnCardToHand(FromCard(paid)))), WithAdditionalCost(tc.cost))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	missing := NewInstant("Missing", "", WithAbility(Spell(ReturnCardToHand(FromCard(paid)))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing payment producer: %v", err)
	}
	duplicate := NewInstant("Duplicate", "", WithAbility(Spell(ReturnCardToHand(FromCard(paid)))),
		WithAdditionalCost(SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid)),
		WithAdditionalCost(SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid)))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate payment producer: %v", err)
	}
}

func TestPaidSacrificeCardFailedPayment(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("paid")
	spell := NewInstant("Offering", "", WithAbility(Spell(ReturnCardToHand(FromCard(paid)))),
		WithAdditionalCost(SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err == nil {
		t.Fatal("cast without a creature succeeded")
	}
	if len(a.Hand()) != 1 || !g.stack.IsEmpty() || len(g.resolution.cardBindings) != 0 {
		t.Fatal("failed payment changed the game or saved a card")
	}
}

func TestPaidSacrificeTokenHasNoGraveyardCard(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("paid")
	source := NewArtifact("Offering", "", WithAbility(Activate(
		Costs(SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid)),
		IfCardHasType(FromCard(paid), TypeCreature, GainLifeFor(You, Fixed(2))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	token := NewToken("Token", 1, 1, []CardType{TypeCreature}, nil)
	g.PutOnBattlefield(token, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("token produced a graveyard card")
	}
}

func TestPaidSacrificeCardActivationAndStaleChoice(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("paid")
	source := NewArtifact("Offering", "", WithAbility(Activate(
		Costs(SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid)),
		IfCardHasType(FromCard(paid), TypeCreature, GainLifeFor(You, Fixed(2))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	creature := NewCreature("Paid", "", 1, 1)
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("activation lost its paid card")
	}

	for _, change := range []func(*Game, Card){
		func(g *Game, p Card) { g.PhaseOut(p.ID()) },
		func(g *Game, p Card) {
			g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
			g.PutOnBattlefield(p, a.PlayerID())
		},
	} {
		other := NewCreature("Other", "", 1, 1)
		g.PutOnBattlefield(other, a.PlayerID())
		tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
			Controller: a.PlayerID(), SourceID: source.ID(),
			Costs: []Cost{SacrificeMatchingCostWithCard(IsCreature, "Sacrifice a creature", paid)},
		})
		if err != nil {
			t.Fatal(err)
		}
		change(g, other)
		if err := tx.Commit(); err == nil || len(tx.paidCards) != 0 {
			t.Fatal("stale chosen incarnation produced a paid card")
		}
		g.RemoveFromBattlefield(g.FindPermanent(other.ID()))
	}
}

// --- from composable_paid_counters_test.go ---
type counterChoicePlayer struct {
	*BasePlayer
	choice, calls int
}

func (p *counterChoicePlayer) ChooseNumber(minimum, maximum int, _ string) int {
	p.calls++
	return p.choice
}

func TestPaidCounterChoiceActivation(t *testing.T) {
	a := &counterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: 2}
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid counters")
	source := NewArtifact("Battery", "", WithAbility(Activate(
		Costs(RemoveAnyCountersCost(Charge, paid)),
		GainLifeFor(You, FromNumber(paid)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.calls != 1 || g.FindPermanent(source.ID()).Counters[Charge] != 1 {
		t.Fatal("payment did not use one locked choice")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 || a.Life() != 20 {
		t.Fatal("copy or clone lost payment value")
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatalf("life = %d", a.Life())
	}
}

func TestPaidCounterChoiceRollbackAndValidation(t *testing.T) {
	a := &counterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: 2}
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	source := NewArtifact("Battery", "")
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{RemoveAnyCountersCost(Charge, paid)}})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || g.FindPermanent(source.ID()).Counters[Charge] != 3 {
		t.Fatal("returned source paid old cost")
	}
	phasedSource := NewArtifact("Phasing Battery", "")
	g.PutOnBattlefield(phasedSource, a.PlayerID())
	g.MutablePermanent(phasedSource.ID()).AddCounter(Charge, 3)
	phasedTx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: phasedSource.ID(), Controller: a.PlayerID(), Costs: []Cost{RemoveAnyCountersCost(Charge, paid)}})
	if err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(phasedSource.ID())
	if err := phasedTx.Commit(); err == nil || len(phasedTx.paidNumbers) != 0 {
		t.Fatal("phased-out source paid cost")
	}

	for _, tc := range []struct {
		name string
		cost Cost
		want string
	}{
		{"nil", RemoveAnyCountersCost(Charge, nil), "non-nil"},
		{"unnamed", RemoveAnyCountersCost(Charge, BindNumber("")), "name"},
		{"valid", RemoveAnyCountersCost(Charge, paid), ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Test", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, FromNumber(paid)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}

func TestPaidCounterChoiceZeroAndInvalid(t *testing.T) {
	for _, tc := range []struct {
		name         string
		choice       int
		wantLife     int
		wantCounters int
		wantError    bool
	}{
		{"zero", 0, 20, 3, false},
		{"negative", -1, 20, 3, true},
		{"too many", 4, 20, 3, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &counterChoicePlayer{BasePlayer: NewBasePlayer("A"), choice: tc.choice}
			g := NewGame(a, NewBasePlayer("B"))
			paid := BindNumber("paid")
			source := NewArtifact("Battery", "", WithAbility(Activate(Costs(RemoveAnyCountersCost(Charge, paid)), GainLifeFor(You, FromNumber(paid)))))
			g.PutOnBattlefield(source, a.PlayerID())
			g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			if (err != nil) != tc.wantError {
				t.Fatalf("activation error = %v", err)
			}
			g.ResolveStack()
			if a.Life() != tc.wantLife || int(g.FindPermanent(source.ID()).Counters[Charge]) != tc.wantCounters {
				t.Fatal("invalid or zero payment changed state")
			}
		})
	}
}

// --- from composable_paid_discard_card_test.go ---
type paidLandChoicePlayer struct{ *BasePlayer }

func (p *paidLandChoicePlayer) ChooseCardsFromHand(_ int, _ string, _ GameReader) []Card {
	for _, card := range p.Hand() {
		if card.HasType(TypeLand) {
			return []Card{card}
		}
	}
	return nil
}

func TestPaidDiscardCardActivationAndClone(t *testing.T) {
	a, b := &paidLandChoicePlayer{NewBasePlayer("A")}, NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("discarded")
	source := NewArtifact("Edge", "", WithAbility(Activate(Costs(DiscardCardCostInto(paid)),
		IfCardHasType(FromCard(paid), TypeLand, GainLifeFor(You, Fixed(2))))))
	g.PutOnBattlefield(source, a.PlayerID())
	a.AddToHand(NewLand("Paid land"))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("paid discard capture escaped cloned resolution")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("paid land type was not available to the activation")
	}
}

func TestPaidDiscardCardDefinition(t *testing.T) {
	paid := BindCard("discarded")
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{DiscardCardCostInto(paid), ""},
		{DiscardCardCostInto(nil), "non-nil"},
	} {
		card := NewArtifact("Edge", "", WithAbility(Activate(Costs(tc.cost), IfCardHasType(FromCard(paid), TypeLand, GainLifeFor(You, Fixed(2))))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	missing := NewArtifact("Missing", "", WithAbility(Activate(Costs(), IfCardHasType(FromCard(paid), TypeLand, GainLifeFor(You, Fixed(2))))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing producer: %v", err)
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(DiscardCardCostInto(paid), DiscardCardCostInto(paid)), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate producer: %v", err)
	}
}

func TestPaidDiscardCardReplacementAndRollback(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindCard("discarded")
	land := NewLand("Paid land")
	a.AddToHand(land)
	g.effects.AddReplacement(&allDiscardToLibraryReplacement{})
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{DiscardCardCostInto(paid)}})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 1 || tx.paidCards[paid].visit != uuid.Nil {
		t.Fatal("hidden replacement exposed the discarded card")
	}

	b := NewBasePlayer("C")
	other := NewGame(b, NewBasePlayer("D"))
	b.AddToHand(NewLand("Paid land"))
	other.effects.AddReplacement(&preventCostDiscardReplacement{})
	if _, err := other.prepareActionPaymentTransaction(actionPaymentSpec{Controller: b.PlayerID(), Costs: []Cost{DiscardCardCostInto(paid)}}); err == nil {
		t.Fatal("prevented discard paid the cost")
	}
	if len(b.Hand()) != 1 || len(b.Graveyard()) != 0 {
		t.Fatal("failed payment changed live zones")
	}
}

func TestPaidDiscardCardSpellCopyAndVisit(t *testing.T) {
	a, b := &paidLandChoicePlayer{NewBasePlayer("A")}, NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("discarded")
	spell := NewInstant("Discard spell", "", WithAbility(Spell(
		IfCardHasType(FromCard(paid), TypeLand, GainLifeFor(You, Fixed(2))))),
		WithAdditionalCost(DiscardCardCostInto(paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	land := NewLand("Paid land")
	a.AddToHand(land)
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("stack copy did not keep its paid card")
	}

	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.resolution.cardBindings = make(map[*CardBinding]capturedCard)
	g.resolution.cardBindings[paid] = capturedCard{id: land.ID(), visit: g.graveyardVisit(land.ID()), zone: ZoneGraveyard}
	if err := ReturnCardToHand(FromCard(paid)).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if _, ok := FromCard(paid).resolveCard(ctx); ok {
		t.Fatal("paid card followed a later zone visit")
	}
}

// --- from composable_paid_exile_card_test.go ---
func TestPaidExileCardActivationAndVisit(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindCard("exiled card")
	source := NewArtifact("Synthetic", "", WithAbility(Activate(
		Costs(ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile a creature card", paid)),
		IfCardHasType(FromCard(paid), TypeCreature, GainLifeFor(You, Fixed(2))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	card := NewCreature("Payment", "{2}", 1, 1)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 || a.Life() != 20 {
		t.Fatal("clone did not keep the paid card binding")
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatal("paid card was absent at resolution")
	}
}

func TestPaidExileCardSpellAndFailedCost(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindCard("exiled card")
	spell := NewInstant("Synthetic", "", WithAbility(Spell(
		IfCardHasType(FromCard(paid), TypeCreature, GainLifeFor(You, Fixed(2))))),
		WithAdditionalCost(ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile a creature card", paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	card := NewCreature("Payment", "", 1, 1)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("spell did not read paid card")
	}
	source := NewArtifact("Failure", "", WithAbility(Activate(Costs(
		ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile", paid), LifePayCost(25)),
		GainLifeFor(You, Fixed(1)))))
	g.PutOnBattlefield(source, a.PlayerID())
	second := NewCreature("Second", "", 1, 1)
	second.SetOwner(a.PlayerID())
	a.AddToGraveyard(second)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("unpayable cost committed")
	}
	if g.FindExiledCard(second.ID()) != nil || a.Life() != 22 || len(g.StackObjects()) != 0 {
		t.Fatal("failed payment changed live state")
	}
}

func TestPaidExileCardVisitAndRollback(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindCard("exiled card")
	card := NewCreature("Payment", "", 1, 1)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{
		ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile", paid),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := g.MoveFromGraveyard(a.PlayerID(), card.ID(), ZoneHand); !ok {
		t.Fatal("card did not leave graveyard")
	}
	a.AddToGraveyard(card)
	if err := tx.Commit(); err == nil || len(tx.paidCards) != 0 || len(g.GetExile()) != 0 {
		t.Fatalf("stale cost committed: %v", err)
	}
	tx, err = g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{
		ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile", paid),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	ctx := &EffectContext{Game: g}
	g.resolution.cardBindings = tx.paidCards
	if got, ok := FromCard(paid).resolveCard(ctx); !ok || got.ID() != card.ID() {
		t.Fatal("paid exile visit was absent")
	}
	if _, ok := g.RemoveFromExile(card.ID()); !ok {
		t.Fatal("exiled card was absent")
	}
	g.ExileCard(card, card.ID())
	if _, ok := FromCard(paid).resolveCard(ctx); ok {
		t.Fatal("paid binding followed a new exile visit")
	}
}

func TestPaidExileCardDeclaration(t *testing.T) {
	paid := BindCard("paid")
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile", paid), ""},
		{ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile", nil), "non-nil"},
		{ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Exile", BindCard("")), "name"},
		{ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "", paid), "text"},
	} {
		card := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(tc.cost), IfCardHasType(FromCard(paid), TypeCreature, GainLifeFor(You, Fixed(1))))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(
		ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "One", paid),
		ExileMatchingCardFromGraveyardCostWithCard(IsCreatureCard, "Two", paid)),
		IfCardHasType(FromCard(paid), TypeCreature, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate binding: %v", err)
	}
}

// --- from composable_paid_exile_cards_test.go ---
func TestPaidGraveyardExileCardsActivationAndVisits(t *testing.T) {
	a := &graveyardExileChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewLand("First"), NewLand("Second")
	for _, card := range []Card{first, second} {
		card.SetOwner(a.PlayerID())
		a.AddToGraveyard(card)
	}
	a.choices = []uuid.UUID{second.ID(), first.ID()}
	paid := BindCards("exiled cards")
	source := NewArtifact("Synthetic", "", WithAbility(Activate(
		Costs(ExileFromGraveyardCostWithCards(2, paid)),
		GainLifeFor(You, CountCards(FromCards(paid))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	frame := cloneCardsBindings(g.StackObjects()[0].capturedCardSets)
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	g.resolution.cardsBindings = frame
	if got := CountCards(FromCards(paid)).resolveValue(ctx); got != 2 {
		t.Fatalf("paid cards = %d", got)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("clone lost paid exile visits")
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatal("activation lost paid exile visits")
	}
	if _, ok := g.RemoveFromExile(first.ID()); !ok {
		t.Fatal("first card not exiled")
	}
	g.ExileCard(first, source.ID())
	g.resolution.cardsBindings = frame
	if got := CountCards(FromCards(paid)).resolveValue(ctx); got != 1 {
		t.Fatalf("returned visit counted: %d", got)
	}
}

func TestPaidGraveyardExileCardsDeclaration(t *testing.T) {
	paid := BindCards("paid")
	valid := NewArtifact("Valid", "", WithAbility(Activate(Costs(ExileFromGraveyardCostWithCards(2, paid)), GainLifeFor(You, CountCards(FromCards(paid))))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{ExileFromGraveyardCostWithCards(1, nil), "non-nil"},
		{ExileFromGraveyardCostWithCards(1, BindCards("")), "name"},
		{ExileFromGraveyardCostWithCards(-1, paid), "negative"},
		{EitherCost(ExileFromGraveyardCostWithCards(1, paid), LifePayCost(1)), "direct cost"},
	} {
		card := NewArtifact("Bad", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(
		ExileFromGraveyardCostWithCards(0, paid), ExileFromGraveyardCostWithCards(0, paid)),
		GainLifeFor(You, CountCards(FromCards(paid))))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate validation = %v", err)
	}
	unbound := NewInstant("Unbound", "", Spell(GainLifeFor(You, CountCards(FromCards(paid)))))
	if err := ValidateCardDefinition(unbound.Name(), unbound); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("unbound validation = %v", err)
	}
}

func TestPaidGraveyardExileCardsSpellCopyAndRollback(t *testing.T) {
	a := &graveyardExileChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	first, second := NewLand("First"), NewLand("Second")
	for _, card := range []Card{first, second} {
		card.SetOwner(a.PlayerID())
		a.AddToGraveyard(card)
	}
	paid := BindCards("paid")
	spell := NewInstant("Synthetic", "", WithAbility(Spell(GainLifeFor(You, CountCards(FromCards(paid))))),
		WithAdditionalCost(ExileFromGraveyardCostWithCards(2, paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.choices = []uuid.UUID{first.ID(), second.ID()}
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	copiedSpell := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
	if copiedSpell == nil {
		t.Fatal("copy absent")
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatalf("copied spell life = %d", a.Life())
	}

	third := NewLand("Third")
	third.SetOwner(a.PlayerID())
	a.AddToGraveyard(third)
	a.choices = []uuid.UUID{third.ID()}
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{
		ExileFromGraveyardCostWithCards(1, paid), LifePayCost(2),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := g.MoveFromGraveyard(a.PlayerID(), third.ID(), ZoneHand); !ok {
		t.Fatal("card did not leave")
	}
	a.RemoveFromHand(third.ID())
	a.AddToGraveyard(third)
	if err := tx.Commit(); err == nil || len(tx.paidCardSets) != 0 || a.Life() != 24 || g.FindExiledCard(third.ID()) != nil {
		t.Fatalf("stale payment changed state: %v", err)
	}
}

// --- from composable_paid_exile_mana_value_test.go ---
func TestExiledGraveyardCostManaValueActivation(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("exiled mana value")
	source := NewArtifact("Synthetic", "", WithAbility(Activate(
		Costs(ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile a creature card", paid)),
		GainLifeFor(You, FromNumber(paid)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	for _, cost := range []string{"{2}", "{5}"} {
		card := NewCreature("Payment", cost, 1, 1)
		card.SetOwner(a.PlayerID())
		a.AddToGraveyard(card)
	}
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 29 || a.Life() != 20 {
		t.Fatalf("clone life = %d, original = %d", clone.GetPlayer(a.PlayerID()).Life(), a.Life())
	}
	g.ResolveStack()
	if a.Life() != 29 {
		t.Fatalf("life = %d", a.Life())
	}
}

func TestExiledGraveyardCostManaValueDeclaration(t *testing.T) {
	paid := BindNumber("paid")
	for _, tc := range []struct {
		name string
		cost Cost
		want string
	}{
		{"valid", ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile a creature card", paid), ""},
		{"nil", ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile a creature card", nil), "non-nil"},
		{"unnamed", ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile a creature card", BindNumber("")), "name"},
		{"text", ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "", paid), "text"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, FromNumber(paid)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(
		ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile one", paid),
		ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile two", paid)), GainLifeFor(You, FromNumber(paid)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate validation = %v", err)
	}
	missing := NewArtifact("Missing", "", WithAbility(Activate(Costs(), GainLifeFor(You, FromNumber(paid)))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing producer validation = %v", err)
	}
	branch := NewArtifact("Branch", "", WithAbility(Activate(Costs(), IfPaid(You,
		Costs(ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile", paid)), GainLifeFor(You, FromNumber(paid))))))
	if err := ValidateCardDefinition(branch.Name(), branch); err == nil || !strings.Contains(err.Error(), "activated ability") {
		t.Fatalf("branch validation = %v", err)
	}
}

func TestExiledGraveyardCostManaValueStaleChoiceAndFailedPayment(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	card := NewCreature("Payment", "{4}", 1, 1)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{
		ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile", paid),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := g.MoveFromGraveyard(a.PlayerID(), card.ID(), ZoneHand); !ok {
		t.Fatal("card did not leave graveyard")
	}
	a.AddToGraveyard(card)
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || len(a.Graveyard()) != 1 {
		t.Fatalf("stale payment = %v, captures = %v", err, tx.paidNumbers)
	}
	source := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(
		ExileMatchingCardFromGraveyardCostInto(IsCreatureCard, "Exile", paid), LifePayCost(25)),
		GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("unpayable activation succeeded")
	}
	if len(a.Graveyard()) != 1 || a.Life() != 20 || len(g.StackObjects()) != 0 {
		t.Fatal("failed payment changed live state")
	}
}

// --- from composable_paid_fixed_counters_test.go ---
func TestFixedCounterPaymentCaptureAndCopies(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	source := NewArtifact("Battery", "", WithAbility(Activate(
		Costs(RemoveCountersCostInto(Charge, 2, paid)),
		GainLifeFor(You, FromNumber(paid)),
	)))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if got := g.FindPermanent(source.ID()).Counters[Charge]; got != 1 {
		t.Fatalf("counters = %d", got)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 24 {
		t.Fatalf("clone life = %d", got)
	}
	if got := a.Life(); got != 20 {
		t.Fatalf("original life before resolution = %d", got)
	}
	g.ResolveStack()
	if got := a.Life(); got != 24 {
		t.Fatalf("original life = %d", got)
	}
}

func TestFixedCounterPaymentRejectsChangedSource(t *testing.T) {
	for _, change := range []string{"return", "phase", "short"} {
		t.Run(change, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			paid := BindNumber("paid")
			source := NewArtifact("Battery", "")
			g.PutOnBattlefield(source, a.PlayerID())
			g.MutablePermanent(source.ID()).AddCounter(Charge, 2)
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), RemoveCountersCostInto(Charge, 2, paid)}})
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(source, a.PlayerID())
				g.MutablePermanent(source.ID()).AddCounter(Charge, 2)
			case "phase":
				g.PhaseOut(source.ID())
			case "short":
				g.MutablePermanent(source.ID()).RemoveCounter(Charge, 1)
			}
			if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || a.Life() != 20 {
				t.Fatalf("commit = %v, captures = %v", err, tx.paidNumbers)
			}
		})
	}
}

func TestFixedCounterPaymentDeclaration(t *testing.T) {
	for _, tc := range []struct {
		name string
		cost Cost
		want string
	}{
		{"valid", RemoveCountersCostInto(Charge, 1, BindNumber("paid")), ""},
		{"nil", RemoveCountersCostInto(Charge, 1, nil), "non-nil"},
		{"negative", RemoveCountersCostInto(Charge, -1, BindNumber("paid")), "negative"},
		{"unnamed", RemoveCountersCostInto(Charge, 1, BindNumber("")), "name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Test", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	paid := BindNumber("paid")
	card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(RemoveCountersCostInto(Charge, 1, paid), RemoveCountersCostInto(Charge, 1, paid)), GainLifeFor(You, FromNumber(paid)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("duplicate producer accepted")
	}
	missing := NewArtifact("Missing", "", WithAbility(Activate(Costs(), GainLifeFor(You, FromNumber(paid)))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil {
		t.Fatal("unproduced binding accepted")
	}
	spell := NewInstant("Spell", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)))), WithAdditionalCost(RemoveCountersCostInto(Charge, 1, paid)))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil {
		t.Fatal("battlefield counter payment accepted for a spell")
	}
}

func TestFixedCounterPaymentZeroAndInterleavedActivations(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	firstPaid := BindNumber("first")
	secondPaid := BindNumber("second")
	first := NewArtifact("First", "", WithAbility(Activate(Costs(RemoveCountersCostInto(Charge, 0, firstPaid)), GainLifeFor(You, FromNumber(firstPaid)))))
	second := NewArtifact("Second", "", WithAbility(Activate(Costs(RemoveCountersCostInto(Charge, 2, secondPaid)), GainLifeFor(You, FromNumber(secondPaid)))))
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	g.MutablePermanent(second.ID()).AddCounter(Charge, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), first.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), second.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if got := a.Life(); got != 22 {
		t.Fatalf("life = %d", got)
	}
}

// --- from composable_paid_life_test.go ---
func TestPaidLifeAmountActivationCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("life paid")
	source := NewArtifact("Life source", "", WithAbility(Activate(
		Costs(LifePayCostInto(3, paid)), GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 17 {
		t.Fatalf("life after payment = %d", a.Life())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 23 || a.Life() != 17 {
		t.Fatal("payment result escaped or was lost by clone")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 23 {
		t.Fatal("payment result escaped its stack instance")
	}
}

func TestPaidLifeAmountBoundaryAndFailure(t *testing.T) {
	for _, tc := range []struct {
		amount, starting, want int
		fails                  bool
	}{
		{0, 0, 0, false},
		{3, 3, 3, false},
		{4, 3, 3, true},
	} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		g.PlayerLoseLife(a, 20-tc.starting)
		paid := BindNumber("paid")
		source := NewArtifact("Source", "", WithAbility(Activate(
			Costs(LifePayCostInto(tc.amount, paid)), GainLifeFor(You, FromNumber(paid)))))
		g.PutOnBattlefield(source, a.PlayerID())
		err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
		if (err != nil) != tc.fails {
			t.Fatalf("amount %d: activation = %v", tc.amount, err)
		}
		g.ResolveStack()
		if a.Life() != tc.want {
			t.Fatalf("amount %d: life = %d", tc.amount, a.Life())
		}
	}
}

func TestPaidLifeAmountDefinition(t *testing.T) {
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{LifePayCostInto(2, BindNumber("paid")), ""},
		{LifePayCostInto(2, nil), "non-nil"},
		{LifePayCostInto(-1, BindNumber("paid")), "negative"},
		{LifePayCostInto(2, BindNumber("")), "name"},
	} {
		paid := BindNumber("paid")
		if c, ok := tc.cost.(*lifePayCost); ok && c.paid != nil {
			paid = c.paid
		}
		card := NewArtifact("Source", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, FromNumber(paid)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
}

func TestPaidLifeAmountPendingActivationsAndSpell(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("life paid")
	target := DeclarePlayerTarget("recipient")
	source := NewArtifact("Source", "", WithAbility(Activate(
		Costs(LifePayCostInto(2, paid)), GainLifeFor(target, FromNumber(paid)), WithTargets(target))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 22 {
		t.Fatal("pending activations shared a payment frame")
	}
	spellPaid := BindNumber("spell life")
	spell := NewInstant("Spell", "", WithAbility(Spell(GainLifeFor(You, FromNumber(spellPaid)))), WithAdditionalCost(LifePayCostInto(3, spellPaid)))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	a.AddToHand(spell)
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 18 {
		t.Fatalf("spell did not read its life payment: %d", a.Life())
	}
}

func TestPaidLifeAmountInvalidDeclarations(t *testing.T) {
	b := BindNumber("paid")
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"duplicate", NewArtifact("Duplicate", "", WithAbility(Activate(Costs(LifePayCostInto(1, b), LifePayCostInto(1, b)), GainLifeFor(You, FromNumber(b))))), "duplicate"},
		{"branch", NewArtifact("Branch", "", WithAbility(Activate(Costs(), IfPaid(You, Costs(LifePayCostInto(1, b)), GainLifeFor(You, FromNumber(b)))))), "requires an activated ability"},
		{"unbound", NewArtifact("Unbound", "", WithAbility(Activate(Costs(), GainLifeFor(You, FromNumber(b))))), "producer"},
	} {
		err := ValidateCardDefinition(tc.card.Name(), tc.card)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: validation = %v", tc.name, err)
		}
	}
}

func TestPaidLifeAmountTotalCostFailure(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	source := NewArtifact("Source", "", WithAbility(Activate(
		Costs(LifePayCostInto(3, paid), LifePayCost(100)), GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
		t.Fatal("unpayable total cost was accepted")
	}
	if a.Life() != 20 || len(g.StackObjects()) != 0 {
		t.Fatal("failed total cost changed life or stack")
	}
}

// --- from composable_paid_mana_value_test.go ---
func TestPaidSacrificeManaValuePendingActivations(t *testing.T) {
	a := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("paid mana value")
	target := DeclarePlayerTarget("recipient")
	source := NewArtifact("Offering", "", WithAbility(Activate(
		Costs(SacrificeMatchingCostWithManaValue(IsArtifact, "Sacrifice an artifact", paid)),
		GainLifeFor(target, FromNumber(paid)), WithTargets(target),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Small", "{2}"), a.PlayerID())
	g.PutOnBattlefield(NewArtifact("Large", "{5}"), a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 30 || clone.GetPlayer(b.PlayerID()).Life() != 22 {
		t.Fatalf("clone life: A=%d B=%d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life())
	}
	g.ResolveStack()
	if a.Life() != 30 || b.Life() != 22 || a.choices != 2 {
		t.Fatalf("live life: A=%d B=%d, choices=%d", a.Life(), b.Life(), a.choices)
	}
}

func TestPaidSacrificeManaValueValidationAndRollback(t *testing.T) {
	paid := BindNumber("paid")
	for _, tc := range []struct {
		name string
		cost Cost
		want string
	}{
		{"valid", SacrificeMatchingCostWithManaValue(IsArtifact, "Sacrifice an artifact", paid), ""},
		{"nil", SacrificeMatchingCostWithManaValue(IsArtifact, "Sacrifice an artifact", nil), "non-nil"},
		{"empty", SacrificeMatchingCostWithManaValue(IsArtifact, "Sacrifice an artifact", BindNumber("")), "name"},
		{"no text", SacrificeMatchingCostWithManaValue(IsArtifact, "", paid), "text"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, FromNumber(paid)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	artifact := NewArtifact("Payment", "{3}")
	g.PutOnBattlefield(artifact, a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: a.PlayerID(), Costs: []Cost{SacrificeMatchingCostWithManaValue(IsArtifact, "Sacrifice an artifact", paid)},
	})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(artifact.ID()))
	g.PutOnBattlefield(artifact, a.PlayerID())
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || g.FindPermanent(artifact.ID()) == nil {
		t.Fatal("returned artifact produced a payment result")
	}
}

func TestPaidSacrificeManaValueSpell(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("paid mana value")
	card := NewInstant("Offering", "", WithAbility(Spell(GainLifeFor(You, FromNumber(paid)))),
		WithAdditionalCost(SacrificeMatchingCostWithManaValue(IsArtifact, "Sacrifice an artifact", paid)))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err == nil {
		t.Fatal("spell cast without payment")
	}
	if len(a.Hand()) != 1 || !g.stack.IsEmpty() || a.Life() != 20 {
		t.Fatal("failed cast changed state")
	}
	g.PutOnBattlefield(NewArtifact("Payment", "{4}"), a.PlayerID())
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatalf("life = %d, want 24", a.Life())
	}
}

// --- from composable_paid_mill_test.go ---
func TestPaidMillCountUsesCommittedAmountAndFrames(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	for range 6 {
		card := NewLand("Library card")
		card.SetOwner(a.PlayerID())
		a.AddToLibrary(card)
	}
	paid := BindNumber("milled")
	source := NewArtifact("Mill source", "", WithAbility(Activate(Costs(MillCostInto(2, paid)), GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(source, a.PlayerID())
	modifier := NewCreature("Modifier", "", 1, 1)
	g.PutOnBattlefield(modifier, a.PlayerID())
	g.AddMillReplacement(modifier.ID(), func(_ *Game, _ uuid.UUID, n int) int { return n + 1 })
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(modifier.ID()))
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Library()) != 1 {
		t.Fatalf("library = %d", len(a.Library()))
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 28 {
		t.Fatalf("clone life = %d", got)
	}
	if a.Life() != 20 {
		t.Fatal("clone changed original life")
	}
	g.ResolveStack()
	if a.Life() != 28 {
		t.Fatalf("life = %d", a.Life())
	}
}

func TestPaidMillCountRollbackAndDeclaration(t *testing.T) {
	paid := BindNumber("milled")
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{MillCostInto(2, paid), ""},
		{MillCostInto(0, paid), "positive"},
		{MillCostInto(2, nil), "non-nil"},
		{MillCostInto(2, BindNumber("")), "name"},
		{EitherCost(MillCostInto(2, paid), LifePayCost(1)), "direct cost"},
	} {
		card := NewArtifact("Test", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(MillCostInto(1, paid), MillCostInto(1, paid)), GainLifeFor(You, FromNumber(paid)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil {
		t.Fatal("duplicate producer accepted")
	}
	optional := NewArtifact("Optional", "", WithAbility(Activate(Costs(), IfPaid(You, Costs(MillCostInto(1, paid)), GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(optional.Name(), optional); err == nil {
		t.Fatal("resolution-time capture accepted")
	}
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	for range 2 {
		card := NewLand("Library card")
		card.SetOwner(a.PlayerID())
		a.AddToLibrary(card)
	}
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{LifePayCost(2), MillCostInto(2, paid)}})
	if err != nil {
		t.Fatal(err)
	}
	a.SetLibrary(a.Library()[:1])
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || a.Life() != 20 || len(a.Library()) != 1 {
		t.Fatalf("rollback: %v, %v", err, tx.paidNumbers)
	}
}

func TestPaidMillCountZeroAndSpellCopy(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("milled")
	spell := NewInstant("Mill spell", "", WithAbility(Spell(GainLifeFor(You, FromNumber(paid)))), WithAdditionalCost(MillCostInto(2, paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	modifier := NewCreature("Modifier", "", 1, 1)
	g.PutOnBattlefield(modifier, a.PlayerID())
	g.AddMillReplacement(modifier.ID(), func(_ *Game, _ uuid.UUID, _ int) int { return 0 })
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	if n := g.StackObjects()[0].capturedNumbers[paid]; n != 0 {
		t.Fatalf("captured count = %d", n)
	}
	g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatalf("zero payment changed life: %d", a.Life())
	}
}

// --- from composable_paid_permanent_test.go ---
func TestPaidPermanentSpellIdentityAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindPermanent("paid")
	spell := NewInstant("Offering", "", WithAbility(Spell(
		DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(2)))),
		WithAdditionalCost(SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	creature := NewCreature("Paid", "", 2, 2, WithKeyword(Lifelink))
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(creature.ID()) != nil {
		t.Fatal("cost did not sacrifice chosen creature")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 18 || a.Life() != 20 {
		t.Fatalf("copy or clone lost paid source controller and lifelink: clone A=%d B=%d, live A=%d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life(), a.Life())
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 18 {
		t.Fatal("paid source did not survive stack copy")
	}
}

func TestPaidPermanentFailedCommitAndDefinition(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindPermanent("paid")
	for _, change := range []func(*Game, Card){
		func(g *Game, p Card) { g.PhaseOut(p.ID()) },
		func(g *Game, p Card) {
			g.RemoveFromBattlefield(g.FindPermanent(p.ID()))
			g.PutOnBattlefield(p, a.PlayerID())
		},
	} {
		creature := NewCreature("Paid", "", 2, 2)
		g.PutOnBattlefield(creature, a.PlayerID())
		tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{
			SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid),
		}})
		if err != nil {
			t.Fatal(err)
		}
		change(g, creature)
		if err := tx.Commit(); err == nil || len(tx.paidPermanents) != 0 {
			t.Fatal("stale choice produced a paid permanent")
		}
		if p := g.FindPermanent(creature.ID()); p != nil {
			g.RemoveFromBattlefield(p)
		}
	}
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid), ""},
		{SacrificeMatchingCostWithPermanent(IsCreature, "", paid), "text"},
		{SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", nil), "non-nil"},
	} {
		card := NewInstant("Test", "", WithAbility(Spell(DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(1)))), WithAdditionalCost(tc.cost))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	missing := NewInstant("Missing", "", WithAbility(Spell(DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(1)))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing binding validation = %v", err)
	}
	duplicate := NewInstant("Duplicate", "", WithAbility(Spell(DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(1)))),
		WithAdditionalCost(SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid)),
		WithAdditionalCost(SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid)))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate binding validation = %v", err)
	}
	resolutionCost := NewArtifact("Invalid", "", WithAbility(Activate(Costs(),
		IfPaid(You, Costs(SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid)),
			GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(resolutionCost.Name(), resolutionCost); err == nil || !strings.Contains(err.Error(), "activated ability cost") {
		t.Fatalf("resolution payment validation = %v", err)
	}
}

func TestPaidPermanentActivationRetainsFormerIncarnation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindPermanent("paid")
	source := NewArtifact("Offering", "", WithAbility(Activate(
		Costs(SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid)),
		DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(3)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	creature := NewCreature("Paid", "", 2, 2, WithKeyword(Lifelink))
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(creature, b.PlayerID())
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatalf("life A=%d B=%d, want former controller's lifelink", a.Life(), b.Life())
	}
}

func TestPaidPermanentAlternateCostCast(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindPermanent("paid")
	spell := NewInstant("Offering", "", WithAbility(Spell(
		DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(2)))),
		WithAdditionalCost(SacrificeMatchingCostWithPermanent(IsCreature, "Sacrifice a creature", paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	g.PutOnBattlefield(NewCreature("Paid", "", 1, 1, WithKeyword(Lifelink)), a.PlayerID())
	if err := g.CastCardFromZoneWithAlternateCost(a.PlayerID(), spell.ID(), ZoneHand, ManaCost{}, nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatalf("life = %d, want damage and lifelink from paid creature", a.Life())
	}
}

// --- from composable_paid_reveal_test.go ---
func TestPaidRevealCardKeepsOneHandVisitPerActivation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindCard("revealed")
	source := NewArtifact("Watcher", "", WithAbility(Activate(
		Costs(RevealFromHandCostWithCard(IsCreatureCard, "Reveal a creature", paid)),
		IfCardHasType(FromCard(paid), TypeCreature, GainLifeFor(You, Fixed(2))))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	land := NewCreature("Creature", "", 1, 1)
	a.AddToHand(land)
	for range 2 {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	if len(g.PublicReveals()) != 2 {
		t.Fatal("reveal costs were not made public on commit")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 || clone.GetPlayer(b.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatal("reveal binding did not stay in each stack frame")
	}
	if _, ok := a.RemoveFromHand(land.ID()); !ok {
		t.Fatal("revealed card left hand")
	}
	a.AddToHand(land)
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("a returned hand card kept an old reveal binding")
	}
}

func TestPaidRevealCardRegistrationAndRollback(t *testing.T) {
	paid := BindCard("revealed")
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{RevealFromHandCostWithCard(IsCreatureCard, "Reveal a creature", paid), ""},
		{RevealFromHandCostWithCard(IsCreatureCard, "Reveal a creature", nil), "non-nil"},
		{RevealFromHandCostWithCard(IsCreatureCard, "Reveal a creature", BindCard("")), "name"},
		{RevealFromHandCostWithCard(IsCreatureCard, "", paid), "text"},
	} {
		card := NewArtifact("Watcher", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	land := NewCreature("Creature", "", 1, 1)
	a.AddToHand(land)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{
		RevealFromHandCostWithCard(IsCreatureCard, "Reveal a creature", paid), LifePayCost(25),
	}})
	if err == nil || tx != nil || a.Life() != 20 || len(a.Hand()) != 1 || len(g.PublicReveals()) != 0 {
		t.Fatalf("failed total payment changed game: %v", err)
	}
	missing := NewArtifact("Missing", "", WithAbility(Activate(Costs(), IfCardHasType(FromCard(paid), TypeCreature, GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing binding: %v", err)
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(
		RevealFromHandCostWithCard(IsCreatureCard, "Reveal one", paid),
		RevealFromHandCostWithCard(IsCreatureCard, "Reveal two", paid)), GainLifeFor(You, Fixed(1)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate binding: %v", err)
	}
}

func TestPaidRevealCardRejectsStalePayment(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindCard("revealed")
	card := NewCreature("Creature", "", 1, 1)
	a.AddToHand(card)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{
		RevealFromHandCostWithCard(IsCreatureCard, "Reveal a creature", paid),
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := a.RemoveFromHand(card.ID()); !ok {
		t.Fatal("card did not leave hand")
	}
	a.AddToHand(card)
	if err := tx.Commit(); err == nil || len(tx.paidCards) != 0 || a.handVisit(card.ID()) == uuid.Nil {
		t.Fatalf("stale payment committed: %v", err)
	}
}

// --- from composable_paid_sacrifice_subtype_test.go ---
func TestPaidSacrificeSubtypeAndStackFrames(t *testing.T) {
	a := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	thrull := BindNumber("was Thrull")
	source := NewCreature("Praetor", "", 5, 5, WithAbility(Activate(
		Costs(SacrificeCreatureCostWithSubtype("Thrull", thrull)),
		IfPositive(FromNumber(thrull), GainLifeFor(You, Fixed(1))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(NewCreature("Thrull", "", 1, 1, WithSubTypes("Thrull")), a.PlayerID())
	g.PutOnBattlefield(NewCreature("Other", "", 1, 1), a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	clone.ResolveStack()
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 21 {
		t.Fatalf("clone life = %d, want 21", got)
	}
	g.ResolveStack()
	if got := a.Life(); got != 21 {
		t.Fatalf("life = %d, want 21", got)
	}
}

func TestPaidSacrificeSubtypeRollbackAndDeclaration(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("paid")
	creature := NewCreature("Thrull", "", 1, 1, WithSubTypes("Thrull"))
	g.PutOnBattlefield(creature, a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{SacrificeCreatureCostWithSubtype("Thrull", paid)}})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 {
		t.Fatal("returned creature produced a payment value")
	}
	for _, tc := range []struct {
		subtype string
		binding *NumberBinding
		want    string
	}{
		{"Thrull", paid, ""}, {"", paid, "subtype"}, {"Island", paid, "creature subtype"}, {"Thrull", nil, "non-nil"},
	} {
		card := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(SacrificeCreatureCostWithSubtype(tc.subtype, tc.binding)), GainLifeFor(You, Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("subtype %q: validation = %v", tc.subtype, err)
		}
	}
}

func TestPaidSacrificeSubtypeSourceAndStackCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	matched := BindNumber("matched")
	source := NewCreature("Self", "", 2, 2, WithSubTypes("Thrull"), WithAbility(Activate(
		Costs(SacrificeCreatureCostWithSubtype("Thrull", matched)),
		IfPositive(FromNumber(matched), GainLifeFor(You, Fixed(2))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("source was not sacrificed")
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 {
		t.Fatal("source payment subtype was not kept in both stack frames")
	}
}

func TestPaidSacrificeSubtypeSpellAndResolutionCost(t *testing.T) {
	paid := BindNumber("paid subtype")
	spell := NewInstant("Offering", "", WithAbility(Spell(
		IfPositive(FromNumber(paid), GainLifeFor(You, Fixed(1))))),
		WithAdditionalCost(SacrificeCreatureCostWithSubtype("Thrull", paid)))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatalf("spell additional cost: %v", err)
	}
	invalid := NewInstant("Invalid", "", WithAbility(Spell(
		IfPaid(You, Costs(SacrificeCreatureCostWithSubtype("Thrull", paid)), GainLifeFor(You, Fixed(1))))))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil || !strings.Contains(err.Error(), "activated ability cost or spell additional cost") {
		t.Fatalf("resolution payment validation = %v", err)
	}
}

// --- from composable_paid_sacrifice_test.go ---
func TestPaidSacrificeStatsActivation(t *testing.T) {
	a := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
	g := NewGame(a, NewBasePlayer("B"))
	power, toughness := BindNumber("paid power"), BindNumber("paid toughness")
	source := NewArtifact("Offering", "", WithAbility(Activate(
		Costs(SacrificeCreatureCostWithStats(power, toughness)),
		GainLifeFor(You, FromNumber(toughness)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(NewCreature("Small", "", 1, 1), a.PlayerID())
	chosen := NewCreature("Large", "", 3, 5)
	g.PutOnBattlefield(chosen, a.PlayerID())
	g.MutablePermanent(chosen.ID()).AddCounter(P1P1, 2)
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.choices != 1 || g.FindPermanent(chosen.ID()) != nil {
		t.Fatal("payment did not sacrifice the chosen creature once")
	}
	g.ResolveStack()
	if a.Life() != 27 {
		t.Fatalf("life = %d, want paid toughness 7", a.Life())
	}
}

func TestPaidSacrificeStatsCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power, toughness := BindNumber("paid power"), BindNumber("paid toughness")
	source := NewArtifact("Offering", "", WithAbility(Activate(
		Costs(SacrificeCreatureCostWithStats(power, toughness)),
		GainLifeFor(You, FromNumber(power)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	creature := NewCreature("Creature", "", 4, 2)
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 || clone.GetPlayer(b.PlayerID()).Life() != 24 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone or stack copy lost its payment value")
	}
	g.ResolveStack()
	if a.Life() != 24 || b.Life() != 24 {
		t.Fatal("payment value escaped its stack object")
	}
}

func TestPaidSacrificeStatsPendingActivations(t *testing.T) {
	a := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	power, toughness := BindNumber("paid power"), BindNumber("paid toughness")
	target := DeclarePlayerTarget("recipient")
	source := NewArtifact("Offering", "", WithAbility(Activate(
		Costs(SacrificeCreatureCostWithStats(power, toughness)),
		GainLifeFor(target, FromNumber(power)), WithTargets(target),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(NewCreature("First", "", 2, 1), a.PlayerID())
	g.PutOnBattlefield(NewCreature("Second", "", 5, 1), a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.choices != 2 || a.Life() != 25 || b.Life() != 22 {
		t.Fatal("pending activations shared a payment result")
	}
}

func TestPaidSacrificeStatsModalActivation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power, toughness := BindNumber("paid power"), BindNumber("paid toughness")
	source := NewArtifact("Offering", "", WithAbility(ActivateModes(
		Costs(SacrificeCreatureCostWithStats(power, toughness)),
		ModeOf("gain", GainLifeFor(You, FromNumber(toughness))),
		ModeOf("lose", LoseLifeFor(You, FromNumber(power))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(NewCreature("Creature", "", 2, 4), a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatalf("life = %d, want paid toughness 4", a.Life())
	}
}

func TestPaidSacrificeStatsFailedCommit(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power, toughness := BindNumber("paid power"), BindNumber("paid toughness")
	creature := NewCreature("Creature", "", 4, 2)
	g.PutOnBattlefield(creature, a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: a.PlayerID(), Costs: []Cost{SacrificeCreatureCostWithStats(power, toughness)},
	})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || g.FindPermanent(creature.ID()) == nil {
		t.Fatal("returned creature produced a payment result")
	}
}

func TestPaidSacrificeStatsDefinition(t *testing.T) {
	for _, tc := range []struct {
		name string
		cost Cost
		body ResolvingEffect
		want string
	}{
		{"valid", SacrificeCreatureCostWithStats(BindNumber("power"), BindNumber("toughness")), GainLifeFor(You, Fixed(1)), ""},
		{"nil power", SacrificeCreatureCostWithStats(nil, BindNumber("toughness")), GainLifeFor(You, Fixed(1)), "non-nil"},
		{"nil both", SacrificeCreatureCostWithStats(nil, nil), GainLifeFor(You, Fixed(1)), "non-nil"},
		{"duplicate", SacrificeCreatureCostWithStats(BindNumber("same"), BindNumber("same")), GainLifeFor(You, Fixed(1)), "duplicate"},
		{"read before payment", SacrificeCreatureCost(), GainLifeFor(You, FromNumber(BindNumber("missing"))), "producer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(tc.cost), tc.body)))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}

func TestPaidSacrificeStatsSpellAndResolutionCost(t *testing.T) {
	power, toughness := BindNumber("power"), BindNumber("toughness")
	cost := SacrificeCreatureCostWithStats(power, toughness)
	card := NewInstant("Invalid", "", WithAbility(Spell(GainLifeFor(You, FromNumber(toughness)))), WithAdditionalCost(cost))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatalf("additional cost validation = %v", err)
	}
	card = NewInstant("Invalid", "", WithAbility(NewAction(ActionSpell, WithCost(cost), WithEffect(GainLifeFor(You, FromNumber(toughness))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatalf("spell action cost validation = %v", err)
	}
	card = NewArtifact("Invalid", "", WithAbility(Activate(Costs(), IfPaid(You, Costs(cost), GainLifeFor(You, FromNumber(toughness))))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "activated ability cost") {
		t.Fatalf("resolution payment validation = %v", err)
	}
}

// --- from composable_paid_sacrifice_total_power_test.go ---
func TestPaidSacrificeTotalPowerActivationCopyAndClone(t *testing.T) {
	a := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	total := BindNumber("total power")
	source := NewArtifact("Offering", "", WithAbility(Activate(Costs(SacrificeNMatchingCostWithTotalPower(2, IsCreature, "Sacrifice two creatures", total)), GainLifeFor(You, FromNumber(total)))))
	g.PutOnBattlefield(source, a.PlayerID())
	small := NewCreature("Small", "", 1, 1)
	large := NewCreature("Large", "", 3, 3)
	g.PutOnBattlefield(small, a.PlayerID())
	g.PutOnBattlefield(large, a.PlayerID())
	g.MutablePermanent(large.ID()).AddCounter(P1P1, 1)
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.choices != 2 {
		t.Fatalf("choices = %d", a.choices)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 25 || clone.GetPlayer(b.PlayerID()).Life() != 25 {
		t.Fatal("clone lost paid total")
	}
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	if a.Life() != 25 || b.Life() != 25 {
		t.Fatal("stack copy lost paid total")
	}
}

func TestPaidSacrificeTotalPowerRollback(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	first := NewCreature("First", "", 2, 2)
	second := NewCreature("Second", "", 3, 3)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	total := BindNumber("total")
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), Costs: []Cost{SacrificeNMatchingCostWithTotalPower(2, IsCreature, "Sacrifice two", total)}})
	if err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(second.ID())
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || g.FindPermanent(first.ID()) == nil {
		t.Fatal("phased choice paid or captured")
	}
	g.PhaseIn([]uuid.UUID{second.ID()})
	g.RemoveFromBattlefield(g.FindPermanent(second.ID()))
	g.PutOnBattlefield(second, a.PlayerID())
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || g.FindPermanent(first.ID()) == nil {
		t.Fatal("returned choice paid or captured")
	}
}

func TestPaidSacrificeTotalPowerDefinition(t *testing.T) {
	for _, tc := range []struct {
		name string
		cost Cost
		want string
	}{
		{"valid", SacrificeNMatchingCostWithTotalPower(2, IsCreature, "Sacrifice two", BindNumber("total")), ""},
		{"nil", SacrificeNMatchingCostWithTotalPower(2, IsCreature, "Sacrifice two", nil), "non-nil"},
		{"unnamed", SacrificeNMatchingCostWithTotalPower(2, IsCreature, "Sacrifice two", BindNumber("")), "name"},
		{"count", SacrificeNMatchingCostWithTotalPower(0, IsCreature, "Sacrifice", BindNumber("total")), "positive"},
		{"text", SacrificeNMatchingCostWithTotalPower(2, IsCreature, "", BindNumber("total")), "text"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Synthetic", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}

func TestPaidSacrificeTotalPowerPendingActivations(t *testing.T) {
	a := &lastSacrificeChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	total := BindNumber("total")
	target := DeclarePlayerTarget("recipient")
	source := NewArtifact("Offering", "", WithAbility(Activate(Costs(SacrificeNMatchingCostWithTotalPower(2, IsCreature, "Sacrifice two", total)), GainLifeFor(target, FromNumber(total)), WithTargets(target))))
	g.PutOnBattlefield(source, a.PlayerID())
	for i := 1; i <= 4; i++ {
		g.PutOnBattlefield(NewCreature("Creature", "", i, 1), a.PlayerID())
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 27 || b.Life() != 23 {
		t.Fatalf("life totals = %d, %d", a.Life(), b.Life())
	}
}

func TestPaidSacrificeTotalPowerSpellAndScope(t *testing.T) {
	total := BindNumber("total")
	spell := NewInstant("Offering", "", WithAbility(Spell(GainLifeFor(You, FromNumber(total)))), WithAdditionalCost(SacrificeNMatchingCostWithTotalPower(2, IsCreature, "Sacrifice two", total)))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(
		SacrificeNMatchingCostWithTotalPower(1, IsCreature, "Sacrifice one", total),
		SacrificeNMatchingCostWithTotalPower(1, IsCreature, "Sacrifice another", total),
	), GainLifeFor(You, FromNumber(total)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate validation = %v", err)
	}
	branch := NewArtifact("Branch", "", WithAbility(Activate(Costs(), IfPaid(You,
		Costs(SacrificeNMatchingCostWithTotalPower(1, IsCreature, "Sacrifice one", total)),
		GainLifeFor(You, FromNumber(total))))))
	if err := ValidateCardDefinition(branch.Name(), branch); err == nil || !strings.Contains(err.Error(), "activated ability cost") {
		t.Fatalf("branch validation = %v", err)
	}
}

// --- from composable_paid_source_mana_value_test.go ---
func TestPaidSourceSacrificeManaValue(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("paid mana value")
	source := NewCreature("Source", "{3}{U}", 1, 1, WithAbility(Activate(
		Costs(SacrificeMatchingIncludingSourceCostWithManaValue(And(IsCreature, HasColorFilter(Blue)), "Sacrifice a blue creature", paid)),
		GainLifeFor(You, FromNumber(paid)),
	)))
	g.PutOnBattlefield(NewCreature("Red creature", "{R}", 1, 1), a.PlayerID())
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("source was not sacrificed")
	}
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 24 {
		t.Fatalf("clone life = %d", clone.GetPlayer(a.PlayerID()).Life())
	}
	g.ResolveStack()
	if a.Life() != 24 {
		t.Fatalf("life = %d", a.Life())
	}
}

func TestPaidSourceSacrificeManaValueValidationAndRollback(t *testing.T) {
	paid := BindNumber("paid")
	for _, tc := range []struct {
		name string
		cost Cost
		want string
	}{
		{"valid", SacrificeMatchingIncludingSourceCostWithManaValue(IsCreature, "Sacrifice a creature", paid), ""},
		{"nil", SacrificeMatchingIncludingSourceCostWithManaValue(IsCreature, "Sacrifice a creature", nil), "non-nil"},
		{"empty", SacrificeMatchingIncludingSourceCostWithManaValue(IsCreature, "Sacrifice a creature", BindNumber("")), "name"},
		{"no text", SacrificeMatchingIncludingSourceCostWithManaValue(IsCreature, "", paid), "text"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewCreature("Synthetic", "{2}", 1, 1, WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, FromNumber(paid)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "{2}", 1, 1)
	g.PutOnBattlefield(source, a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{SacrificeMatchingIncludingSourceCostWithManaValue(IsCreature, "Sacrifice a creature", paid)},
	})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || g.FindPermanent(source.ID()) == nil {
		t.Fatal("returned source produced a payment result")
	}
}

// --- from composable_paid_source_permanent_test.go ---
func TestPaidSourcePermanentIdentityAndCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindPermanent("paid source")
	source := NewCreature("Offering", "", 2, 2, WithKeyword(Lifelink), WithAbility(Activate(
		Costs(SacrificeMatchingIncludingSourceCostWithPermanent(IsCreature, "Sacrifice a creature", paid)),
		DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(2)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("source was not sacrificed")
	}
	g.PutOnBattlefield(source, b.PlayerID())
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 18 || a.Life() != 20 {
		t.Fatalf("copy or clone lost old source: clone A=%d B=%d live A=%d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life(), a.Life())
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 18 {
		t.Fatalf("old source result A=%d B=%d", a.Life(), b.Life())
	}
}

func TestPaidSourcePermanentRollbackAndDeclaration(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindPermanent("paid source")
	source := NewCreature("Offering", "", 2, 2)
	g.PutOnBattlefield(source, a.PlayerID())
	cost := SacrificeMatchingIncludingSourceCostWithPermanent(IsCreature, "Sacrifice a creature", paid)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}})
	if err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(source.ID())
	if err := tx.Commit(); err == nil || len(tx.paidPermanents) != 0 {
		t.Fatal("phased source paid a stale cost")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	second, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{
		SacrificeMatchingIncludingSourceCostWithPermanent(IsCreature, "Sacrifice a creature", paid),
	}})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := second.Commit(); err == nil || len(second.paidPermanents) != 0 {
		t.Fatal("returned source paid a stale cost")
	}
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{cost, ""},
		{SacrificeMatchingIncludingSourceCostWithPermanent(IsCreature, "", paid), "text"},
		{SacrificeMatchingIncludingSourceCostWithPermanent(IsCreature, "Sacrifice a creature", nil), "non-nil"},
	} {
		card := NewCreature("Definition", "", 1, 1, WithAbility(Activate(Costs(tc.cost), DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	missing := NewCreature("Missing", "", 1, 1, WithAbility(Activate(Costs(), DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(1)))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing producer validation = %v", err)
	}
	resolutionCost := NewCreature("Resolution", "", 1, 1, WithAbility(Activate(Costs(),
		IfPaid(You, Costs(SacrificeMatchingIncludingSourceCostWithPermanent(IsCreature, "Sacrifice a creature", paid)),
			DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(1))))))
	if err := ValidateCardDefinition(resolutionCost.Name(), resolutionCost); err == nil || !strings.Contains(err.Error(), "activated ability cost") {
		t.Fatalf("resolution cost validation = %v", err)
	}
}

// --- from composable_paid_source_sacrifice_test.go ---
func TestPaidSourceSacrificeStats(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	power, toughness := BindNumber("power"), BindNumber("toughness")
	source := NewCreature("Offering", "", 2, 3, WithAbility(Activate(
		Costs(SacrificeCreatureIncludingSourceCostWithStats(power, toughness)),
		GainLifeFor(You, FromNumber(toughness)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(P1P1, 2)
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(source.ID()) != nil {
		t.Fatal("the source was not sacrificed")
	}
	g.ResolveStack()
	if a.Life() != 25 {
		t.Fatalf("life = %d, want 25", a.Life())
	}
}

func TestPaidSourceSacrificeStatsRollbackAndDefinition(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	power, toughness := BindNumber("power"), BindNumber("toughness")
	source := NewCreature("Offering", "", 2, 3)
	g.PutOnBattlefield(source, a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{
		Controller: a.PlayerID(), SourceID: source.ID(),
		Costs: []Cost{SacrificeCreatureIncludingSourceCostWithStats(power, toughness)},
	})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || g.FindPermanent(source.ID()) == nil {
		t.Fatal("returned source paid a stale cost")
	}
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{SacrificeCreatureIncludingSourceCostWithStats(power, toughness), ""},
		{SacrificeCreatureIncludingSourceCostWithStats(nil, toughness), "non-nil"},
		{SacrificeCreatureIncludingSourceCostWithStats(power, power), "duplicate"},
	} {
		card := NewCreature("Definition", "", 1, 1, WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("validation = %v, want %q", err, tc.want)
		}
	}
	missing := BindNumber("missing")
	invalid := NewCreature("Missing", "", 1, 1, WithAbility(Activate(
		Costs(SacrificeCreatureIncludingSourceCostWithStats(power, toughness)),
		GainLifeFor(You, FromNumber(missing)),
	)))
	if err := ValidateCardDefinition(invalid.Name(), invalid); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("unbound paid value validation = %v", err)
	}
	spell := NewInstant("Additional", "", WithAbility(Spell(GainLifeFor(You, FromNumber(power)))),
		WithAdditionalCost(SacrificeCreatureIncludingSourceCostWithStats(power, toughness)))
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatalf("spell additional cost validation = %v", err)
	}
}

func TestPaidSourceSacrificeStatsCopyAndClone(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power, toughness := BindNumber("power"), BindNumber("toughness")
	source := NewCreature("Offering", "", 3, 2, WithAbility(Activate(
		Costs(SacrificeCreatureIncludingSourceCostWithStats(power, toughness)),
		GainLifeFor(You, FromNumber(power)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 23 || clone.GetPlayer(b.PlayerID()).Life() != 23 || a.Life() != 20 {
		t.Fatal("clone lost the paid source value")
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 23 {
		t.Fatal("stack copy lost the paid source value")
	}
}

// --- from composable_paid_tap_permanent_test.go ---
func TestPaidTapPermanentKeepsPaymentSourceThroughCopies(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindPermanent("paid")
	source := NewArtifact("Source", "", WithAbility(Activate(
		Costs(TapNOtherMatchingCostWithPermanent(IsCreature, "Tap another creature", paid)),
		DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(2)),
	)))
	creature := NewCreature("Paid", "", 1, 1, WithKeyword(Lifelink))
	g.PutOnBattlefield(source, a.PlayerID())
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(creature.ID()).Tapped {
		t.Fatal("payment did not tap the chosen creature")
	}
	g.RemoveFromBattlefield(g.FindPermanent(creature.ID()))
	g.PutOnBattlefield(creature, b.PlayerID())
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || a.Life() != 20 {
		t.Fatalf("clone payment source: clone A=%d, live A=%d", clone.GetPlayer(a.PlayerID()).Life(), a.Life())
	}
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 18 {
		t.Fatalf("paid source identity: A=%d B=%d", a.Life(), b.Life())
	}
}

func TestPaidTapPermanentRejectsStalePayment(t *testing.T) {
	a := NewBasePlayer("A")
	for _, change := range []func(*Game, Card){
		func(g *Game, c Card) { g.PhaseOut(c.ID()) },
		func(g *Game, c Card) {
			g.RemoveFromBattlefield(g.FindPermanent(c.ID()))
			g.PutOnBattlefield(c, a.PlayerID())
		},
	} {
		g := NewGame(a, NewBasePlayer("B"))
		paid := BindPermanent("paid")
		source := NewArtifact("Source", "")
		creature := NewCreature("Paid", "", 1, 1)
		g.PutOnBattlefield(source, a.PlayerID())
		g.PutOnBattlefield(creature, a.PlayerID())
		tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: []Cost{
			TapNOtherMatchingCostWithPermanent(IsCreature, "Tap another creature", paid), LifePayCost(2),
		}})
		if err != nil {
			t.Fatal(err)
		}
		change(g, creature)
		if err := tx.Commit(); err == nil || len(tx.paidPermanents) != 0 || a.Life() != 20 {
			t.Fatalf("stale payment changed live state: err=%v, paid=%v, life=%d", err, tx.paidPermanents, a.Life())
		}
	}
}

func TestPaidTapPermanentSpellAdditionalCost(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindPermanent("paid")
	spell := NewInstant("Spell", "", WithAbility(Spell(
		DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(2)))),
		WithAdditionalCost(TapNOtherMatchingCostWithPermanent(IsCreature, "Tap a creature", paid)))
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	creature := NewCreature("Paid", "", 1, 1, WithKeyword(Lifelink))
	g.PutOnBattlefield(creature, a.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(creature.ID()).Tapped {
		t.Fatal("additional cost did not tap the chosen creature")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatalf("paid source characteristics: A=%d B=%d", a.Life(), b.Life())
	}
}

func TestPaidTapPermanentDeclaration(t *testing.T) {
	paid := BindPermanent("paid")
	for _, tc := range []struct {
		cost Cost
		want string
	}{
		{TapNOtherMatchingCostWithPermanent(IsCreature, "Tap another creature", paid), ""},
		{TapNOtherMatchingCostWithPermanent(IsCreature, "", paid), "text"},
		{TapNOtherMatchingCostWithPermanent(IsCreature, "Tap another creature", nil), "non-nil"},
		{TapNOtherMatchingCostWithPermanent(IsCreature, "Tap another creature", BindPermanent("")), "name"},
	} {
		card := NewArtifact("Source", "", WithAbility(Activate(Costs(tc.cost), TapSubject(From(paid)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("definition = %v, want %q", err, tc.want)
		}
	}
	missing := NewArtifact("Missing", "", WithAbility(Activate(Costs(), TapSubject(From(paid)))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil || !strings.Contains(err.Error(), "producer") {
		t.Fatalf("missing producer = %v", err)
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(
		TapNOtherMatchingCostWithPermanent(IsCreature, "Tap another creature", paid),
		TapNOtherMatchingCostWithPermanent(IsCreature, "Tap another creature", paid),
	), TapSubject(From(paid)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate producer = %v", err)
	}
	resolutionCost := NewArtifact("Resolution", "", WithAbility(Activate(Costs(),
		IfPaid(You, Costs(TapNOtherMatchingCostWithPermanent(IsCreature, "Tap another creature", paid)),
			TapSubject(From(paid))))))
	if err := ValidateCardDefinition(resolutionCost.Name(), resolutionCost); err == nil || !strings.Contains(err.Error(), "activated ability cost") {
		t.Fatalf("resolution payment binding = %v", err)
	}
}

// --- from composable_paid_x_counters_test.go ---
func TestPaidXCountersActivationFrames(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	paid := BindNumber("removed")
	source := NewArtifact("Counter source", "", WithAbility(Activate(
		Costs(RemoveXCountersFromSourceCostInto(Charge, paid)),
		GainLifeFor(You, FromNumber(paid)),
	)))
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 5)
	for _, x := range []int{0, 2, 3} {
		g.SetXValue(x)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	if got := g.FindPermanent(source.ID()).Counters[Charge]; got != 0 {
		t.Fatalf("remaining counters = %d", got)
	}
	g.CopyStackObjectDirect(g.StackObjects()[1], a.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if got := clone.GetPlayer(a.PlayerID()).Life(); got != 27 {
		t.Fatalf("clone life = %d", got)
	}
	if got := a.Life(); got != 20 {
		t.Fatalf("original life = %d", got)
	}
	g.ResolveStack()
	if got := a.Life(); got != 27 {
		t.Fatalf("life = %d", got)
	}
}

func TestPaidXCountersRollbackAndDeclaration(t *testing.T) {
	paid := BindNumber("removed")
	for _, tc := range []struct {
		name string
		cost Cost
		want string
	}{
		{"valid", RemoveXCountersFromSourceCostInto(Charge, paid), ""},
		{"nil", RemoveXCountersFromSourceCostInto(Charge, nil), "non-nil"},
		{"unnamed", RemoveXCountersFromSourceCostInto(Charge, BindNumber("")), "name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewArtifact("Test", "", WithAbility(Activate(Costs(tc.cost), GainLifeFor(You, Fixed(1)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	duplicate := NewArtifact("Duplicate", "", WithAbility(Activate(Costs(RemoveXCountersFromSourceCostInto(Charge, paid), RemoveXCountersFromSourceCostInto(Charge, paid)), GainLifeFor(You, FromNumber(paid)))))
	if err := ValidateCardDefinition(duplicate.Name(), duplicate); err == nil {
		t.Fatal("duplicate binding accepted")
	}
	spell := NewInstant("Spell", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)))), WithAdditionalCost(RemoveXCountersFromSourceCostInto(Charge, paid)))
	if err := ValidateCardDefinition(spell.Name(), spell); err == nil {
		t.Fatal("battlefield cost accepted on spell")
	}
	missing := NewArtifact("Missing", "", WithAbility(Activate(Costs(), GainLifeFor(You, FromNumber(paid)))))
	if err := ValidateCardDefinition(missing.Name(), missing); err == nil {
		t.Fatal("unproduced binding accepted")
	}
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	source := NewArtifact("Source", "")
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 2)
	resolved := g.prepareActionCosts([]Cost{RemoveXCountersFromSourceCostInto(Charge, paid)}, nil, 2)
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: append([]Cost{LifePayCost(2)}, resolved...)})
	if err != nil {
		t.Fatal(err)
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	g.PutOnBattlefield(source, a.PlayerID())
	g.MutablePermanent(source.ID()).AddCounter(Charge, 2)
	if err := tx.Commit(); err == nil || len(tx.paidNumbers) != 0 || a.Life() != 20 {
		t.Fatalf("commit = %v, capture = %v", err, tx.paidNumbers)
	}
}

func TestPaidXCountersLockedAmountAndPhasing(t *testing.T) {
	for _, phase := range []bool{false, true} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		paid := BindNumber("paid")
		source := NewArtifact("Source", "")
		g.PutOnBattlefield(source, a.PlayerID())
		g.MutablePermanent(source.ID()).AddCounter(Charge, 2)
		costs := g.prepareActionCosts([]Cost{RemoveXCountersFromSourceCostInto(Charge, paid)}, nil, 2)
		tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{SourceID: source.ID(), Controller: a.PlayerID(), Costs: costs})
		if err != nil {
			t.Fatal(err)
		}
		g.SetXValue(0)
		if phase {
			g.PhaseOut(source.ID())
		}
		err = tx.Commit()
		if phase {
			if err == nil || len(tx.paidNumbers) != 0 {
				t.Fatalf("phased payment = %v, capture = %v", err, tx.paidNumbers)
			}
		} else if err != nil || tx.paidNumbers[paid] != 2 || g.FindPermanent(source.ID()).Counters[Charge] != 0 {
			t.Fatalf("locked payment = %v, capture = %v", err, tx.paidNumbers)
		}
	}
}

func TestPaidXCountersRejectsUnpayableX(t *testing.T) {
	for _, x := range []int{-1, 3} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		paid := BindNumber("paid")
		source := NewArtifact("Source", "", WithAbility(Activate(
			Costs(RemoveXCountersFromSourceCostInto(Charge, paid)),
			GainLifeFor(You, FromNumber(paid)),
		)))
		g.PutOnBattlefield(source, a.PlayerID())
		g.MutablePermanent(source.ID()).AddCounter(Charge, 2)
		g.SetXValue(x)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil {
			t.Fatalf("X=%d was payable", x)
		}
		if len(g.StackObjects()) != 0 || a.Life() != 20 || g.FindPermanent(source.ID()).Counters[Charge] != 2 {
			t.Fatalf("X=%d changed game state", x)
		}
	}
}

// --- from composable_paid_x_life_test.go ---
func TestPaidXLifeSpellBoundaryAndRollback(t *testing.T) {
	for _, tc := range []struct {
		x, life, want int
		fails         bool
	}{
		{0, 0, 0, false},
		{3, 3, 3, false},
		{4, 3, 3, true},
	} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		g.PlayerLoseLife(a, 20-tc.life)
		paid := BindNumber("paid")
		spell := NewInstant("Variable life", "", WithAbility(Spell(GainLifeFor(You, FromNumber(paid)))), WithAdditionalCost(LifePayXCostInto(paid)))
		if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
			t.Fatal(err)
		}
		a.AddToHand(spell)
		err := g.CastSpellByName(a.PlayerID(), spell.Name(), nil, tc.x)
		if (err != nil) != tc.fails {
			t.Fatalf("X=%d: cast = %v", tc.x, err)
		}
		if tc.fails && (len(g.StackObjects()) != 0 || len(a.Hand()) != 1) {
			t.Fatal("failed payment changed stack or hand")
		}
		g.ResolveStack()
		if a.Life() != tc.want {
			t.Fatalf("X=%d: life = %d, want %d", tc.x, a.Life(), tc.want)
		}
	}
}

func TestPaidXLifeSeparateInstances(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindNumber("paid")
	source := NewArtifact("Source", "", WithAbility(Activate(Costs(LifePayXCostInto(paid)), GainLifeFor(You, FromNumber(paid)))))
	g.PutOnBattlefield(source, a.PlayerID())
	if err := ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.SetXValue(2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.SetXValue(3)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 15 {
		t.Fatalf("life after payment = %d", a.Life())
	}
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 22 || a.Life() != 15 {
		t.Fatal("clone or copy lost instance payment")
	}
	g.ResolveStack()
	if a.Life() != 20 || b.Life() != 22 {
		t.Fatal("pending payments shared state")
	}
}

func TestPaidXLifeRegistration(t *testing.T) {
	paid := BindNumber("paid")
	for _, tc := range []struct {
		card Card
		want string
	}{
		{NewInstant("Nil", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)))), WithAdditionalCost(LifePayXCostInto(nil))), "non-nil"},
		{NewInstant("Unnamed", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)))), WithAdditionalCost(LifePayXCostInto(BindNumber("")))), "name"},
		{NewInstant("Duplicate", "", WithAbility(Spell(GainLifeFor(You, FromNumber(paid)))), WithAdditionalCost(LifePayXCostInto(paid)), WithAdditionalCost(LifePayXCostInto(paid))), "duplicate"},
		{NewInstant("Unbound", "", WithAbility(Spell(GainLifeFor(You, FromNumber(paid))))), "producer"},
	} {
		err := ValidateCardDefinition(tc.card.Name(), tc.card)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: %v", tc.card.Name(), err)
		}
	}
}

// --- from composable_payment_test.go ---
func TestComposablePaymentAtomicity(t *testing.T) {
	for _, accept := range []bool{false, true} {
		for _, total := range []int{3, 25} {
			t.Run(map[bool]string{false: "decline", true: "accept"}[accept]+Fixed(total).Text(), func(t *testing.T) {
				base := NewBasePlayer("A")
				var a Player = base
				if accept {
					a = &acceptingMayPlayer{base}
				}
				b := NewBasePlayer("B")
				g := NewGame(a, b)
				effect := UnlessPaid(You, Costs(LifePayCost(1), LifePayCost(total-1)), GainLifeFor(ActivePlayer, Fixed(4)))
				if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want := 24
				if accept && total == 3 {
					want = 17
				}
				if got := g.GetPlayer(a.PlayerID()).Life(); got != want {
					t.Fatalf("life = %d, want %d", got, want)
				}
			})
		}
	}
}

func TestComposablePaymentSavedSubject(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Land")
	land.SetOwner(b.PlayerID())
	g.PutOnBattlefield(land, b.PlayerID())
	saved := BindPermanent("land")
	effect := Sequence(Snapshot(This, saved), UnlessPaid(ControllerOf(From(saved)), Costs(ManaPayment("{1}")), Destroy(From(saved))))
	if err := ApplyEffect(g, effect, land.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(land.ID()) != nil {
		t.Fatal("unpaid branch lost its saved subject")
	}
	if err := ValidateCardDefinition("Payment", NewCreature("Payment", "", 1, 1, WithAbility(Activate(Costs(), effect)))); err != nil {
		t.Fatal(err)
	}
	bad := UnlessPaid(You, Costs(), Snapshot(This, saved))
	if err := ValidateCardDefinition("Scope", NewCreature("Scope", "", 1, 1, WithAbility(Activate(Costs(), Sequence(bad, Destroy(From(saved))))))); err == nil {
		t.Fatal("payment branch exported an assignment")
	}
}

func TestComposableIfPaid(t *testing.T) {
	for _, accept := range []bool{false, true} {
		base := NewBasePlayer("A")
		var player Player = base
		if accept {
			player = &acceptingMayPlayer{base}
		}
		g := NewGame(player, NewBasePlayer("B"))
		effect := IfPaid(You, Costs(LifePayCost(3)), GainLifeFor(You, Fixed(4)))
		if err := ApplyEffect(g, effect, uuid.Nil, player.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := 20
		if accept {
			want = 21
		}
		if got := g.GetPlayer(player.PlayerID()).Life(); got != want {
			t.Fatalf("life = %d, want %d", got, want)
		}
	}
}

// --- from composable_permanent_capture_test.go ---
func TestActivationPermanentCaptureLifetime(t *testing.T) {
	for _, change := range []string{"return source", "return host", "phase host", "phase round trip", "control", "absent"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			host := BindPermanent("host")
			body := Sequence(AddCountersTo(From(host), Charge, Fixed(1)), Snapshot(This, host), GainLifeFor(You, Fixed(2)))
			aura := NewAura("Source", "", WithAbility(Activate(Costs(SacrificeSourceCost()), body, CapturePermanentBeforePayment(AttachedPermanent, host))))
			first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
			for _, c := range []Card{first, second, aura} {
				c.SetOwner(a.PlayerID())
				g.PutOnBattlefield(c, a.PlayerID())
			}
			if change != "absent" {
				g.Attach(aura.ID(), first.ID())
			}
			if err := ValidateCardDefinition(aura.Name(), aura); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), aura.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(aura.ID()) != nil || g.StackSize() != 1 {
				t.Fatal("sacrifice was not paid")
			}
			want := uint8(2)
			switch change {
			case "return source":
				g.PutOnBattlefield(aura, a.PlayerID())
				g.Attach(aura.ID(), second.ID())
			case "return host":
				g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
				g.PutOnBattlefield(first, a.PlayerID())
				want = 0
			case "phase host":
				g.PhaseOut(first.ID())
				want = 0
			case "phase round trip":
				g.PhaseOut(first.ID())
				g.PhaseIn([]uuid.UUID{first.ID()})
			case "control":
				if err := ApplyEffect(g, GainControl().Until(EndOfTurn), b.PlayerID(), b.PlayerID(), []uuid.UUID{first.ID()}); err != nil {
					t.Fatal(err)
				}
			case "absent":
				want = 0
			}
			clone := g.Clone()
			for _, game := range []*Game{clone, g} {
				game.CopyStackObjectDirect(game.StackPeek(), b.PlayerID(), false)
				game.ResolveStack()
				if game.FindPermanentIncludingPhased(first.ID()).Counters[Charge] != want || game.FindPermanent(second.ID()).Counters[Charge] != 0 {
					t.Fatal("capture lost incarnation or copy isolation")
				}
				if game.GetPlayer(a.PlayerID()).Life() != 22 || game.GetPlayer(b.PlayerID()).Life() != 22 {
					t.Fatal("missing subject stopped sequence or changed controller")
				}
				if game == clone && a.Life() != 20 {
					t.Fatal("clone changed original")
				}
			}
		})
	}
}

func TestActivationPermanentCaptureOrderAndValidation(t *testing.T) {
	host := BindPermanent("host")
	n := BindNumber("power")
	valid := []ActivationOption{CapturePermanentBeforePayment(This, host), CaptureNumberBeforePayment(PowerOf(From(host)), n)}
	for _, tc := range []struct {
		name    string
		options []ActivationOption
		want    string
	}{
		{"valid", valid, ""},
		{"read before write", []ActivationOption{valid[1], valid[0]}, "producer"},
		{"self read", []ActivationOption{CapturePermanentBeforePayment(From(host), host)}, "producer"},
		{"nil subject", []ActivationOption{CapturePermanentBeforePayment(nil, host)}, "non-nil"},
		{"nil binding", []ActivationOption{CapturePermanentBeforePayment(This, nil)}, "non-nil"},
		{"empty name", []ActivationOption{CapturePermanentBeforePayment(This, BindPermanent(""))}, "name"},
		{"duplicate", []ActivationOption{valid[0], CapturePermanentBeforePayment(This, BindPermanent("host"))}, "duplicate"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCreature("Source", "", 3, 3, WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), tc.options...)))
			err := ValidateCardDefinition(c.Name(), c)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v", err)
			}
		})
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	c := NewCreature("Source", "", 3, 3, WithAbility(Activate(Costs(SacrificeSourceCost()), GainLifeFor(You, FromNumber(n)), valid...)))
	c.SetOwner(a.PlayerID())
	g.PutOnBattlefield(c, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), c.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 23 {
		t.Fatal("mixed captures ran out of order")
	}
}

func TestActivationPermanentCaptureTargetsAndFailure(t *testing.T) {
	g, a, b := newSpellCopyGame()
	host := BindPermanent("host")
	target := DeclarePermanentTarget("creature", IsCreature)
	source := NewArtifact("Source", "", WithAbility(Activate(Costs(RemoveCountersCost(Charge, 1)),
		AddCountersTo(From(host), Charge, Fixed(1)), CapturePermanentBeforePayment(target, host), WithTargets(target))))
	g.PutOnBattlefield(source, a.PlayerID()).AddCounter(Charge, 2)
	first, second := NewCreature("First", "", 2, 2), NewCreature("Second", "", 2, 2)
	g.PutOnBattlefield(first, a.PlayerID())
	g.PutOnBattlefield(second, b.PlayerID())
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{host: {id: source.ID()}}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{source.ID()}); err == nil {
		t.Fatal("illegal target accepted")
	}
	for _, c := range []Card{first, second} {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{c.ID()}); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID()}); err == nil {
		t.Fatal("unpaid activation accepted")
	}
	if g.StackSize() != 2 || g.resolution.permanentBindings[host].id != source.ID() {
		t.Fatal("capture leaked into outer frame")
	}
	a.chooseQueue = [][]uuid.UUID{{first.ID()}}
	copied := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true)
	if copied.Targets[0] != first.ID() {
		t.Fatal("copy did not select the new target")
	}
	g.ResolveStack()
	if g.FindPermanent(first.ID()).Counters[Charge] != 1 || g.FindPermanent(second.ID()).Counters[Charge] != 2 {
		t.Fatalf("counters first=%d second=%d", g.FindPermanent(first.ID()).Counters[Charge], g.FindPermanent(second.ID()).Counters[Charge])
	}
}

func TestActivationPermanentCaptureGrantedModalDelayed(t *testing.T) {
	g, a, b := newModalGame()
	host := BindPermanent("host")
	rule := GrantEachActivatedModes(MatchingPermanents(IsArtifact), Costs(),
		CapturePermanentBeforePayment(This, host),
		ModeOf("now", AddCountersTo(From(host), Charge, Fixed(1))),
		ModeOf("later", AtNext(EachEndStep, AddCountersTo(From(host), Charge, Fixed(1)))))
	donor := NewEnchantment("Donor", "", WithAbility(Static(rule)))
	if err := ValidateCardDefinition(donor.Name(), donor); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(donor, a.PlayerID())
	first, second := NewArtifact("First", ""), NewArtifact("Second", "")
	for _, c := range []Card{first, second} {
		g.PutOnBattlefield(c, b.PlayerID())
		g.ApplyContinuousEffects()
		b.modes = []int{1}
		if err := g.ActivateAbilityByIndex(b.PlayerID(), c.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
	g.FireEvent(GameEvent{Type: EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	clone := g.Clone()
	for _, game := range []*Game{clone, g} {
		game.ResolveStack()
		for _, c := range []Card{first, second} {
			if game.FindPermanent(c.ID()).Counters[Charge] != 1 {
				t.Fatal("grant or delayed capture lost recipient")
			}
		}
	}
}

func TestActivationPermanentCaptureScope(t *testing.T) {
	host := BindPermanent("host")
	target := DeclarePermanentTarget("creature", IsCreature)
	for _, c := range []Card{
		NewArtifact("Other ability", "",
			WithAbility(Activate(Costs(), GainLifeFor(You, Fixed(1)), CapturePermanentBeforePayment(This, host))),
			WithAbility(Activate(Costs(), Regenerate(From(host))))),
		NewArtifact("Mode target", "", WithAbility(ActivateModes(Costs(),
			CapturePermanentBeforePayment(target, host),
			ModeOf("one", Regenerate(From(host)), WithTargets(target)),
			ModeOf("two", Regenerate(From(host)))))),
	} {
		if err := ValidateCardDefinition(c.Name(), c); err == nil {
			t.Fatal("invalid capture scope accepted")
		}
	}
}

func TestActivationPermanentCaptureAbsentOverwrite(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	host := BindPermanent("host")
	c := NewArtifact("Source", "", WithAbility(Activate(Costs(),
		Sequence(AddCountersTo(From(host), Charge, Fixed(1)), GainLifeFor(You, Fixed(1))),
		CapturePermanentBeforePayment(This, host),
		CapturePermanentBeforePayment(AttachedPermanent, host))))
	if err := ValidateCardDefinition(c.Name(), c); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(c, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), c.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(c.ID()).Counters[Charge] != 0 || a.Life() != 21 {
		t.Fatal("absent capture did not clear the earlier binding")
	}
}

// --- from composable_sacrifice_capture_test.go ---
func TestCreatureYouSacrificePermanentCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindPermanent("sacrificed creature")
	source := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(
		CreatureYouSacrifice().WithPermanent(paid),
		Sequence(DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(2)),
			AddCountersTo(From(paid), P1P1, Fixed(1))),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	victim := NewCreature("Victim", "", 3, 3)
	g.PutOnBattlefield(victim, a.PlayerID())
	g.DoSacrifice(g.FindPermanent(victim.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("sacrifice did not create one trigger")
	}
	g.PutOnBattlefield(victim, b.PlayerID())
	g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	clone := g.Clone()
	clone.ResolveStack()
	if a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed original")
	}
	g.ResolveStack()
	for _, game := range []*Game{g, clone} {
		if game.GetPlayer(a.PlayerID()).Life() != 18 || game.GetPlayer(b.PlayerID()).Life() != 18 {
			t.Fatal("captured source did not survive return, copy, or clone")
		}
		if game.FindPermanent(victim.ID()).CurrentPower(game) != 3 {
			t.Fatal("old permanent reference modified the returned incarnation")
		}
	}
}

func TestCreatureYouSacrificePermanentCaptureSelfAndPhasing(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	paid := BindPermanent("paid")
	source := NewCreature("Self", "", 1, 1, WithAbility(WhenEvent(
		CreatureYouSacrifice().WithPermanent(paid),
		DealDamageFrom(DamageSourceOf(From(paid)), You, Fixed(1)),
	)))
	g.PutOnBattlefield(source, a.PlayerID())
	other := NewCreature("Other", "", 1, 1)
	g.PutOnBattlefield(other, a.PlayerID())
	g.PhaseOut(source.ID())
	g.DoSacrifice(g.FindPermanent(other.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("phased source captured a sacrifice")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	g.DoSacrifice(g.FindPermanent(source.ID()))
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("self sacrifice missed its event")
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 20 {
		t.Fatal("self sacrifice lost departure source")
	}
}

func TestCreatureYouSacrificePermanentCaptureValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		make func() EventPattern
		want string
	}{
		{"valid", func() EventPattern { return CreatureYouSacrifice().WithPermanent(BindPermanent("paid")) }, ""},
		{"nil", func() EventPattern { return CreatureYouSacrifice().WithPermanent(nil) }, "non-nil"},
		{"unnamed", func() EventPattern { return CreatureYouSacrifice().WithPermanent(BindPermanent("")) }, "name"},
		{"duplicate", func() EventPattern {
			return CreatureYouSacrifice().WithPermanent(BindPermanent("one")).WithPermanent(BindPermanent("two"))
		}, "once"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewCreature("Synthetic", "", 1, 1, WithAbility(WhenEvent(tc.make(), GainLifeFor(You, Fixed(1)))))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
	missing := BindPermanent("missing")
	card := NewCreature("Unbound", "", 1, 1, WithAbility(WhenEvent(CreatureYouSacrifice(),
		DealDamageFrom(DamageSourceOf(From(missing)), You, Fixed(1)))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("unbound event participant passed registration")
	}
}

// --- from composable_spell_cast_x_capture_test.go ---
func TestSpellCastAnnouncedXCapture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	x := BindNumber("cast X")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().WithXInManaCost().WithAnnouncedX(x), GainLifeFor(You, FromNumber(x)))))
	g.PutOnBattlefield(watcher, a.PlayerID())
	for _, value := range []int{0, 3, 5} {
		spell := NewInstant("Spell", "{X}{R}")
		obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, XValue: value, Controller: b.PlayerID()}
		g.stack.Push(obj)
		g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
		obj.XValue = 99
		g.stack.RemoveByID(obj.ID)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 3 {
		t.Fatalf("stack = %d, want 3", g.StackSize())
	}
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 28 {
			t.Fatalf("captured X: life = %d, want 28", got)
		}
	}
}

func TestSpellCastAnnouncedXDeclaration(t *testing.T) {
	x := BindNumber("cast X")
	pattern := SpellCast().WithAnnouncedX(x)
	if got := InspectDefinition(pattern); got.Role != "event-pattern" || len(got.Children) != 1 {
		t.Fatalf("inspection = %#v", got)
	}
	for _, tc := range []struct {
		name string
		card Card
		want string
	}{
		{"valid", NewArtifact("Valid", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, FromNumber(x))))), ""},
		{"nil", NewArtifact("Nil", "", WithAbility(WhenEvent(SpellCast().WithAnnouncedX(nil), GainLifeFor(You, Fixed(1))))), "binding"},
		{"unnamed", NewArtifact("Unnamed", "", WithAbility(WhenEvent(SpellCast().WithAnnouncedX(BindNumber("")), GainLifeFor(You, Fixed(1))))), "name"},
		{"duplicate", NewArtifact("Duplicate", "", WithAbility(WhenEvent(pattern.WithAnnouncedX(BindNumber("other")), GainLifeFor(You, Fixed(1))))), "one"},
		{"unbound", NewArtifact("Unbound", "", WithAbility(WhenEvent(SpellCast(), GainLifeFor(You, FromNumber(x))))), "producer"},
		{"wrong zone", NewInstant("Wrong", "", WithAbility(WhenEvent(pattern, GainLifeFor(You, FromNumber(x))))), "battlefield"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCardDefinition(tc.card.Name(), tc.card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}

func TestSpellCastAnnouncedXSourceLifecycleAndCopy(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	x := BindNumber("cast X")
	watcher := NewArtifact("Watcher", "", WithAbility(WhenEvent(
		SpellCast().ByOpponent().WithAnnouncedX(x), GainLifeFor(You, FromNumber(x)))))
	permanent := g.PutOnBattlefield(watcher, a.PlayerID())
	spell := NewInstant("Spell", "{X}{R}")
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, XValue: 4, Controller: b.PlayerID()}
	g.stack.Push(obj)
	permanent.PhasedOut = true
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	permanent.PhasedOut = false
	g.FireEvent(GameEvent{Type: EvtSpellCast, SourceID: spell.ID(), TargetID: obj.ID, PlayerID: b.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("stack = %d, want spell and one trigger", g.StackSize())
	}
	g.RemoveFromBattlefield(permanent)
	g.PutOnBattlefield(watcher, b.PlayerID())
	g.stack.RemoveByID(obj.ID)
	g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
	clone := g.Clone()
	for _, game := range []*Game{g, clone} {
		game.ResolveStack()
		if got := game.GetPlayer(a.PlayerID()).Life(); got != 24 {
			t.Fatalf("original trigger life = %d, want 24", got)
		}
		if got := game.GetPlayer(b.PlayerID()).Life(); got != 24 {
			t.Fatalf("copied trigger life = %d, want 24", got)
		}
	}
}

// --- from composable_spell_paid_sacrifice_test.go ---
func TestSpellPaidSacrificeStats(t *testing.T) {
	for _, alternate := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		power, toughness := BindNumber("power"), BindNumber("toughness")
		card := NewInstant("Offering", "", WithAbility(Spell(Sequence(
			GainLifeFor(You, FromNumber(power)), GainLifeFor(You, FromNumber(toughness)),
		))), WithAdditionalCost(SacrificeCreatureCostWithStats(power, toughness)))
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		creature := NewCreature("Creature", "", 2, 3)
		g.PutOnBattlefield(creature, a.PlayerID())
		g.MutablePermanent(creature.ID()).AddCounter(P1P1, 1)
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
		var err error
		if alternate {
			err = g.CastCardFromZoneWithAlternateCost(a.PlayerID(), card.ID(), ZoneHand, ManaCost{}, nil, 0)
		} else {
			err = g.CastSpellByName(a.PlayerID(), card.Name(), nil)
		}
		if err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(creature.ID()) != nil {
			t.Fatal("payment did not sacrifice creature")
		}
		g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false)
		clone := g.Clone()
		clone.ResolveStack()
		if clone.GetPlayer(a.PlayerID()).Life() != 27 || clone.GetPlayer(b.PlayerID()).Life() != 27 {
			t.Fatalf("clone or copy lost paid values: A=%d B=%d", clone.GetPlayer(a.PlayerID()).Life(), clone.GetPlayer(b.PlayerID()).Life())
		}
		g.ResolveStack()
		if a.Life() != 27 || b.Life() != 27 {
			t.Fatalf("spell stack frames shared paid values: A=%d B=%d", a.Life(), b.Life())
		}
	}
}

func TestSpellPaidSacrificeDefinition(t *testing.T) {
	for _, tc := range []struct {
		name string
		cost Cost
		want string
	}{
		{"valid", SacrificeCreatureCostWithStats(BindNumber("power"), BindNumber("toughness")), ""},
		{"nil binding", SacrificeCreatureCostWithStats(nil, BindNumber("toughness")), "non-nil"},
		{"duplicate binding", SacrificeCreatureCostWithStats(BindNumber("same"), BindNumber("same")), "duplicate"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			card := NewInstant("Offering", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)))), WithAdditionalCost(tc.cost))
			err := ValidateCardDefinition(card.Name(), card)
			if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
				t.Fatalf("validation = %v", err)
			}
		})
	}
}

func TestSpellPaidSacrificeFailureKeepsResources(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power, toughness := BindNumber("power"), BindNumber("toughness")
	card := NewInstant("Offering", "", WithAbility(Spell(GainLifeFor(You, FromNumber(power)))), WithAdditionalCost(SacrificeCreatureCostWithStats(power, toughness)))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err == nil {
		t.Fatal("cast without creature succeeded")
	}
	if len(a.Hand()) != 1 || !g.stack.IsEmpty() || a.Life() != 20 {
		t.Fatal("failed cast changed resources")
	}
}

func TestSpellPaidSacrificePendingSpells(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power, toughness := BindNumber("power"), BindNumber("toughness")
	for _, tc := range []struct {
		name  string
		power int
	}{
		{"First", 2}, {"Second", 5},
	} {
		card := NewInstant(tc.name, "", WithAbility(Spell(GainLifeFor(You, FromNumber(power)))), WithAdditionalCost(SacrificeCreatureCostWithStats(power, toughness)))
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		g.PutOnBattlefield(NewCreature(tc.name+" creature", "", tc.power, 1), a.PlayerID())
		if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	if a.Life() != 27 {
		t.Fatalf("life = %d, want separate paid powers", a.Life())
	}
}

func TestSpellPaidSacrificeModalBody(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	power, toughness := BindNumber("power"), BindNumber("toughness")
	card := NewInstant("Modal offering", "", WithAbility(SpellModes(
		ModeOf("power", GainLifeFor(You, FromNumber(power))),
		ModeOf("toughness", GainLifeFor(You, FromNumber(toughness))),
	)), WithAdditionalCost(SacrificeCreatureCostWithStats(power, toughness)))
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	g.PutOnBattlefield(NewCreature("Creature", "", 2, 4), a.PlayerID())
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByName(a.PlayerID(), card.Name(), nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("life = %d, want chosen mode's paid power", a.Life())
	}
}

// --- from composable_tap_matching_cost_test.go ---
type matchingTapChoicePlayer struct {
	*BasePlayer
	choices int
}

func (p *matchingTapChoicePlayer) ChoosePermanent(candidates []*Permanent, _ string, _ GameReader) *Permanent {
	p.choices++
	return candidates[len(candidates)-1]
}

func TestTapNOtherMatchingCostPaysDistinctWhiteCreatures(t *testing.T) {
	a := &matchingTapChoicePlayer{BasePlayer: NewBasePlayer("A")}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	source := NewCreature("Source", "{W}", 2, 6, WithKeyword(Haste))
	g.PutOnBattlefield(source, a.PlayerID())
	chosen := make([]Card, 3)
	for i := range chosen {
		chosen[i] = NewCreature("White", "{W}", 1, 1)
		g.PutOnBattlefield(chosen[i], a.PlayerID())
	}
	blue := NewCreature("Blue", "{U}", 1, 1)
	g.PutOnBattlefield(blue, a.PlayerID())
	opponent := NewCreature("Opponent", "{W}", 1, 1)
	g.PutOnBattlefield(opponent, b.PlayerID())
	cost := TapNOtherMatchingCost(3, And(IsCreature, HasColorFilter(White)), "Tap three white creatures")
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{Tap(), cost}})
	if err != nil {
		t.Fatal(err)
	}
	if a.choices != 3 {
		t.Fatalf("choices = %d, want 3", a.choices)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if !g.FindPermanent(source.ID()).Tapped {
		t.Fatal("source tap cost was not paid")
	}
	for _, card := range chosen {
		if !g.FindPermanent(card.ID()).Tapped {
			t.Fatalf("%s was not tapped", card.Name())
		}
	}
	if g.FindPermanent(blue.ID()).Tapped || g.FindPermanent(opponent.ID()).Tapped || a.choices != 3 {
		t.Fatal("payment used an ineligible object or chose again at commit")
	}
}

func TestTapNOtherMatchingCostRejectsChangedChoiceWithoutPartialPayment(t *testing.T) {
	for _, change := range []string{"tap", "phase", "return", "color"} {
		t.Run(change, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			source := NewArtifact("Source", "")
			g.PutOnBattlefield(source, a.PlayerID())
			cards := make([]Card, 3)
			for i := range cards {
				cards[i] = NewCreature("White", "{W}", 1, 1)
				g.PutOnBattlefield(cards[i], a.PlayerID())
			}
			cost := TapNOtherMatchingCost(3, And(IsCreature, HasColorFilter(White)), "Tap three white creatures")
			tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{cost}})
			if err != nil {
				t.Fatal(err)
			}
			changed := cards[0]
			switch change {
			case "tap":
				g.TapPermanent(g.FindPermanent(changed.ID()))
			case "phase":
				g.PhaseOut(changed.ID())
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(changed.ID()))
				g.PutOnBattlefield(changed, a.PlayerID())
			case "color":
				colors := []Color{Blue}
				g.MutablePermanent(changed.ID()).ColorOverride = &colors
			}
			clone := g.Clone()
			if err := tx.Commit(); err == nil {
				t.Fatal("stale payment succeeded")
			}
			for _, card := range cards[1:] {
				if g.FindPermanent(card.ID()).Tapped || clone.FindPermanent(card.ID()).Tapped {
					t.Fatal("failed payment tapped another creature or changed clone")
				}
			}
		})
	}
}

func TestTapNOtherMatchingCostRegistration(t *testing.T) {
	for _, tc := range []struct {
		count      int
		text, want string
	}{
		{3, "Tap three white creatures", ""},
		{0, "Tap", "positive"},
		{-1, "Tap", "positive"},
		{3, "", "text"},
	} {
		card := NewArtifact("Synthetic", "", WithAbility(Activate(
			Costs(TapNOtherMatchingCost(tc.count, IsCreature, tc.text)),
			GainLifeFor(You, Fixed(1)))))
		err := ValidateCardDefinition(card.Name(), card)
		if tc.want == "" && err != nil || tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)) {
			t.Fatalf("count %d, text %q: validation = %v, want %q", tc.count, tc.text, err, tc.want)
		}
	}
}

func TestTapNOtherMatchingCostEventsSeeCompleteTap(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	source := NewArtifact("Source", "")
	g.PutOnBattlefield(source, a.PlayerID())
	cards := make([]Card, 3)
	for i := range cards {
		cards[i] = NewCreature("White", "{W}", 1, 1)
		g.PutOnBattlefield(cards[i], a.PlayerID())
	}
	events, incomplete := 0, false
	watcher := NewEnchantment("Watcher", "", WithAbility(NewTriggered(EvtTapped, false,
		GainLifeFor(You, Fixed(1))).SetCondition(func(_ *GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		events++
		for _, card := range cards {
			if !reader.FindPermanent(card.ID()).Tapped {
				incomplete = true
			}
		}
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{
		TapNOtherMatchingCost(3, And(IsCreature, HasColorFilter(White)), "Tap three white creatures"),
	}})
	if err != nil {
		t.Fatal(err)
	}
	events, incomplete = 0, false
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if events != 3 || incomplete {
		t.Fatalf("events = %d, incomplete = %t", events, incomplete)
	}
}

func TestTapNOtherMatchingCostInsufficientAndRollback(t *testing.T) {
	a := NewBasePlayer("A")
	g := NewGame(a, NewBasePlayer("B"))
	source := NewArtifact("Source", "")
	g.PutOnBattlefield(source, a.PlayerID())
	cards := make([]Card, 3)
	for i := range cards {
		cards[i] = NewCreature("White", "{W}", 1, 1)
		g.PutOnBattlefield(cards[i], a.PlayerID())
	}
	cost := TapNOtherMatchingCost(3, And(IsCreature, HasColorFilter(White)), "Tap three white creatures")
	g.TapPermanent(g.FindPermanent(cards[0].ID()))
	if _, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{LifePayCost(2), cost}}); err == nil {
		t.Fatal("payment with only two eligible creatures succeeded")
	}
	if a.Life() != 20 || g.FindPermanent(cards[1].ID()).Tapped || g.FindPermanent(cards[2].ID()).Tapped {
		t.Fatal("failed preparation changed live payment state")
	}
	g.MutablePermanent(cards[0].ID()).Tapped = false
	tx, err := g.prepareActionPaymentTransaction(actionPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Costs: []Cost{LifePayCost(2), cost}})
	if err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(cards[0].ID())
	if err := tx.Commit(); err == nil || a.Life() != 20 {
		t.Fatalf("changed choice paid life: %v", err)
	}
}

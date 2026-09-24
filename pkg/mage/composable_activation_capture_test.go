package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

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

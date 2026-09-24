package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

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

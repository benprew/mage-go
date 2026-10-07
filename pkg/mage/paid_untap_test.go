package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type paidUntapChooser struct {
	*BasePlayer
	remaining int
	invalid   *Permanent
	check     func([]*Permanent)
}

func (p *paidUntapChooser) ChooseMayAbility(string) bool {
	if p.remaining == 0 {
		return false
	}
	p.remaining--
	return true
}
func (p *paidUntapChooser) ChoosePermanent(candidates []*Permanent, _ string, _ GameReader) *Permanent {
	if p.check != nil {
		p.check(candidates)
	}
	if p.invalid != nil {
		return p.invalid
	}
	return candidates[len(candidates)-1]
}

func TestPaidUntapSelectionAndTotalPayment(t *testing.T) {
	for _, tc := range []struct {
		name                          string
		choices, mana, untapped, left int
	}{
		{"all", 2, 8, 2, 0}, {"subset", 1, 8, 1, 4}, {"decline", 0, 8, 0, 8}, {"total-unaffordable", 2, 7, 0, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &paidUntapChooser{BasePlayer: NewBasePlayer("A"), remaining: tc.choices}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			first := g.PutOnBattlefield(NewCreature("First", "{U}", 1, 2, WithKeyword(core.Shroud)), a.PlayerID())
			second := g.PutOnBattlefield(NewCreature("Second", "{U}", 1, 2), a.PlayerID())
			other := g.PutOnBattlefield(NewCreature("Other", "{U}", 1, 2), b.PlayerID())
			for _, p := range []*Permanent{first, second, other} {
				g.TapPermanent(p)
			}
			a.ManaPool().Add(core.Colorless, tc.mana)
			a.check = func(candidates []*Permanent) {
				for _, p := range candidates {
					if !p.Tapped || p.ControllerID() != a.PlayerID() {
						t.Fatal("wrong candidates")
					}
				}
				if a.ManaPool().TotalMana() != tc.mana {
					t.Fatal("payment preceded choices")
				}
			}
			effect := UntapWithManaPayment(You, And(IsCreature, IsTapped, HasColorFilter(core.Blue)), core.ManaCost{Generic: 4})
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			count := 0
			for _, p := range []*Permanent{first, second} {
				if !g.FindPermanent(p.ID()).Tapped {
					count++
				}
			}
			if count != tc.untapped || a.ManaPool().TotalMana() != tc.left || !g.FindPermanent(other.ID()).Tapped {
				t.Fatalf("untapped=%d mana=%d", count, a.ManaPool().TotalMana())
			}
		})
	}
}

func TestPaidUntapInvalidAndStaleChoice(t *testing.T) {
	for _, change := range []string{"foreign", "return", "phase", "control"} {
		t.Run(change, func(t *testing.T) {
			a := &paidUntapChooser{BasePlayer: NewBasePlayer("A"), remaining: 1}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("First", "{U}", 1, 2)
			p := g.PutOnBattlefield(card, a.PlayerID())
			g.TapPermanent(p)
			foreign := g.PutOnBattlefield(NewCreature("Other", "{U}", 1, 2), b.PlayerID())
			g.TapPermanent(foreign)
			a.ManaPool().Add(core.Blue, 1)
			if change == "foreign" {
				a.invalid = foreign
			} else {
				a.check = func([]*Permanent) {
					switch change {
					case "return":
						g.RemoveFromBattlefield(p)
						g.PutOnBattlefield(card, a.PlayerID())
					case "phase":
						g.PhaseOut(p.ID())
					case "control":
						g.AddControlEffect(ControlEffectSpec{SourceID: foreign.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					}
				}
			}
			if err := ApplyEffect(g, UntapWithManaPayment(You, And(IsCreature, IsTapped), core.ManaCost{Blue: 1}), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().Count(core.Blue) != 1 || !g.FindPermanent(foreign.ID()).Tapped {
				t.Fatal("invalid choice paid")
			}
		})
	}
}

func TestPaidUntapDefinition(t *testing.T) {
	for _, effect := range []ResolvingEffect{
		UntapWithManaPayment(nil, IsCreature, core.ManaCost{}),
		UntapWithManaPayment(You, PermanentFilter{}, core.ManaCost{}),
		UntapWithManaPayment(You, IsCreature, core.ManaCost{HasX: true, XCount: 1}),
		UntapWithManaPayment(You, IsCreature, core.ManaCost{Blue: -1}),
	} {
		card := NewInstant("Invalid", "", WithAbility(Spell(effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
	valid := NewInstant("Valid", "", WithAbility(Spell(UntapWithManaPayment(You, IsCreature, core.ManaCost{}))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
}

func TestPaidUntapManaSourcesAndBatch(t *testing.T) {
	for _, useSources := range []bool{false, true} {
		t.Run(fmt.Sprint(useSources), func(t *testing.T) {
			a := &paidUntapChooser{BasePlayer: NewBasePlayer("A"), remaining: 3}
			g := NewGame(a, NewBasePlayer("B"))
			first := g.PutOnBattlefield(NewCreature("First", "", 1, 2), a.PlayerID())
			second := g.PutOnBattlefield(NewCreature("Second", "", 1, 2), a.PlayerID())
			stunned := g.PutOnBattlefield(NewCreature("Stunned", "", 1, 2), a.PlayerID())
			for _, p := range []*Permanent{first, second, stunned} {
				g.TapPermanent(p)
			}
			g.MutablePermanent(stunned.ID()).AddCounter(core.Stun, 1)
			var lands []*Permanent
			if useSources {
				for range 3 {
					lands = append(lands, g.PutOnBattlefield(NewLand("Island", WithSubTypes("Island")), a.PlayerID()))
				}
			} else {
				a.ManaPool().Add(core.Blue, 3)
			}
			events := 0
			observer := NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtBecameUntapped, false, GainLifeFor(You, Fixed(1))).SetCondition(func(event *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
				if event.SourceID == first.ID() || event.SourceID == second.ID() {
					events++
					if reader.FindPermanent(first.ID()).Tapped || reader.FindPermanent(second.ID()).Tapped || reader.FindPermanent(stunned.ID()).Counters[core.Stun] != 0 {
						t.Fatal("partial untap batch")
					}
				}
				return false
			})))
			g.PutOnBattlefield(observer, a.PlayerID())
			if err := ApplyEffect(g, UntapWithManaPayment(You, IsCreature, core.ManaCost{Blue: 1}), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if events != 2 || !g.FindPermanent(stunned.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
				t.Fatal("batch, stun replacement, or payment")
			}
			for _, land := range lands {
				if !g.FindPermanent(land.ID()).Tapped {
					t.Fatal("mana source not used")
				}
			}
		})
	}
}

func TestPaidUntapCannotFundPaymentWithResult(t *testing.T) {
	a := &paidUntapChooser{BasePlayer: NewBasePlayer("A"), remaining: 2}
	g := NewGame(a, NewBasePlayer("B"))
	for range 2 {
		p := g.PutOnBattlefield(NewCreature("Mana creature", "", 1, 2, WithManaAbility(core.Blue)), a.PlayerID())
		g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
		g.TapPermanent(p)
	}
	island := g.PutOnBattlefield(NewLand("Island", WithSubTypes("Island")), a.PlayerID())
	if err := ApplyEffect(g, UntapWithManaPayment(You, IsCreature, core.ManaCost{Blue: 1}), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(island.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
		t.Fatal("failed total payment spent mana")
	}
	for _, p := range g.FilterBattlefield(IsCreature) {
		if !p.Tapped {
			t.Fatal("untap result funded its own payment")
		}
	}
}

func TestPaidUntapPaymentDeparture(t *testing.T) {
	a := &paidUntapChooser{BasePlayer: NewBasePlayer("A"), remaining: 1}
	g := NewGame(a, NewBasePlayer("B"))
	card := NewCreature("Chosen", "", 1, 2)
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.TapPermanent(source)
	land := g.PutOnBattlefield(NewLand("Mana source", WithManaAbility(core.Blue, DestroyEach(MatchingPermanents(IsCreature)))), a.PlayerID())
	if err := ApplyEffect(g, UntapWithManaPayment(You, IsCreature, core.ManaCost{Blue: 1}), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(source.ID()) != nil || !g.FindPermanent(land.ID()).Tapped || len(a.Graveyard()) != 1 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("departed payment object or mana payment")
	}
}

package mage

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellStepRestriction(t *testing.T) {
	for step := Untap; step <= Cleanup; step++ {
		for _, playerIndex := range []int{0, 1} {
			for _, path := range []string{"hand", "free", "alternate", "copy cast"} {
				t.Run(fmt.Sprintf("%v/player%d/%s", step, playerIndex, path), func(t *testing.T) {
					g := newPriorityTestGame()
					g.SetStep(step)
					p := g.players[playerIndex]
					p.ManaPool().Add(Blue, 1)
					card := NewInstant("Step restricted", "{U}", NewSpell(GainLifeFor(You, Fixed(1)), WithStepOnly(DeclareAttackers)))
					card.SetOwner(p.PlayerID())
					p.AddToHand(card)
					want := step == DeclareAttackers
					if got := len(g.GetCastableSpells(p.PlayerID())) > 0; got != want {
						t.Errorf("castable = %v, want %v", got, want)
					}
					var err error
					switch path {
					case "hand":
						err = g.CastSpellByID(p.PlayerID(), card.ID(), nil, 0)
					case "free":
						err = g.CastCardFromZoneWithoutPaying(p.PlayerID(), card.ID(), ZoneHand, nil, 0)
					case "alternate":
						err = g.CastCardFromZoneWithAlternateCost(p.PlayerID(), card.ID(), ZoneHand, card.ManaCost(), nil, 0)
					case "copy cast":
						_, err = g.pushCastSpellObject(castStackObjectOptions{Card: card.Copy(), Controller: p.PlayerID(), IsCopy: true})
					}
					if (err == nil) != want {
						t.Fatalf("cast error = %v, allowed = %v", err, want)
					}
					if !want {
						if len(p.Hand()) != 1 || p.ManaPool().TotalMana() != 1 || !g.stack.IsEmpty() {
							t.Fatal("rejected cast changed hand, mana, or stack")
						}
					}
				})
			}
		}
	}
}

func TestSpellStepCopyAndResolution(t *testing.T) {
	g := newPriorityTestGame()
	p := g.players[0]
	g.SetStep(DeclareAttackers)
	card := NewInstant("Timed spell", "", WithAbility(Spell(GainLifeFor(You, Fixed(2)), OnlyDuring(BeginCombat, DeclareAttackers))))
	card.SetOwner(p.PlayerID())
	p.AddToHand(card)
	if err := g.CastSpellByID(p.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.SetStep(DeclareBlockers)
	clone := g.Clone()
	original := clone.stack.Peek()
	if clone.CopySpellOnStack(original.SourceID, p.PlayerID(), false) == nil {
		t.Fatal("stack copy was blocked by casting restriction")
	}
	clone.ResolveTopOfStack()
	clone.ResolveTopOfStack()
	if clone.GetPlayer(p.PlayerID()).Life() != p.Life()+4 {
		t.Fatal("restriction blocked resolution after the step changed")
	}
	if g.stack.Size() != 1 {
		t.Fatal("clone changed original stack")
	}
	copied := card.Copy()
	copied.SetID(uuid.New())
	p.AddToHand(copied)
	if err := g.CastSpellByID(p.PlayerID(), copied.ID(), nil, 0); err == nil {
		t.Fatal("card copy lost restriction")
	}
}

func TestSpellStepDoesNotGrantSorceryTiming(t *testing.T) {
	g := newPriorityTestGame()
	p := g.players[0]
	g.SetStep(DeclareAttackers)
	card := NewSorcery("Restricted sorcery", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)), OnlyDuring(DeclareAttackers))))
	card.SetOwner(p.PlayerID())
	p.AddToHand(card)
	if len(g.GetCastableSpells(p.PlayerID())) != 0 {
		t.Fatal("step restriction granted timing permission")
	}
	if err := g.CastSpellByID(p.PlayerID(), card.ID(), nil, 0); !errors.Is(err, ErrSorcerySpeed) {
		t.Fatalf("cast error = %v", err)
	}
	if err := g.CastCardFromZoneWithoutPaying(p.PlayerID(), card.ID(), ZoneHand, nil, 0); err != nil {
		t.Fatal(err)
	}
}

func TestSpellStepOptionValidation(t *testing.T) {
	for _, options := range [][]SpellOption{
		{OnlyDuring(PhaseStep(-1))},
		{OnlyDuring(Cleanup + 1)},
		{OnlyDuring(BeginCombat, Cleanup+1)},
		{OnlyDuring(BeginCombat, PhaseStep(-1))},
		{OnlyDuring(DeclareAttackers), OnlyDuring(DeclareAttackers)},
		{OnlyDuring(DeclareAttackers), OnlyDuring(DeclareBlockers)},
	} {
		if err := validateFactory("Invalid spell", func() Card {
			return NewInstant("Invalid spell", "", WithAbility(Spell(GainLifeFor(You, Fixed(1)), options...)))
		}); err == nil {
			t.Fatal("invalid spell timing accepted")
		}
	}
}

func TestSpellMultipleStepsRestriction(t *testing.T) {
	for step := Untap; step <= Cleanup; step++ {
		for _, playerIndex := range []int{0, 1} {
			for _, path := range []string{"hand", "free", "alternate", "copy cast"} {
				t.Run(fmt.Sprintf("%v/player%d/%s", step, playerIndex, path), func(t *testing.T) {
					g := newPriorityTestGame()
					g.SetStep(step)
					p := g.players[playerIndex]
					p.ManaPool().Add(Blue, 1)
					card := NewInstant("Step restricted", "{U}", WithAbility(Spell(GainLifeFor(You, Fixed(1)), OnlyDuring(BeginCombat, DeclareAttackers))))
					card.SetOwner(p.PlayerID())
					p.AddToHand(card)
					want := step == BeginCombat || step == DeclareAttackers
					if got := len(g.GetCastableSpells(p.PlayerID())) > 0; got != want {
						t.Errorf("castable = %v, want %v", got, want)
					}
					var err error
					switch path {
					case "hand":
						err = g.CastSpellByID(p.PlayerID(), card.ID(), nil, 0)
					case "free":
						err = g.CastCardFromZoneWithoutPaying(p.PlayerID(), card.ID(), ZoneHand, nil, 0)
					case "alternate":
						err = g.CastCardFromZoneWithAlternateCost(p.PlayerID(), card.ID(), ZoneHand, card.ManaCost(), nil, 0)
					case "copy cast":
						_, err = g.pushCastSpellObject(castStackObjectOptions{Card: card.Copy(), Controller: p.PlayerID(), IsCopy: true})
					}
					if (err == nil) != want {
						t.Fatalf("cast error = %v, allowed = %v", err, want)
					}
					if !want {
						if len(p.Hand()) != 1 || p.ManaPool().TotalMana() != 1 || !g.stack.IsEmpty() {
							t.Fatal("rejected cast changed hand, mana, or stack")
						}
					}
				})
			}
		}
	}
}

func TestMultipleStepActivationRestrictions(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		g := newPriorityTestGame()
		p := g.ActivePlayerObj()
		steps := []PhaseStep{DeclareAttackers}
		timing := OnlyDuring(BeginCombat, steps...)
		steps[0] = EndStep
		options := []ActivationOption{timing, OnlyOnYourTurn(), ActivationLimit(2)}
		if reverse {
			options[0], options[1] = options[1], options[0]
		}
		ability := Activate(Costs(), GainLifeFor(You, Fixed(1)), options...)
		for step := Untap; step <= Cleanup; step++ {
			g.SetStep(step)
			want := step == BeginCombat || step == DeclareAttackers
			if got := ability.CanActivate(p.PlayerID(), g); got != want {
				t.Fatalf("step %v: allowed %v, want %v", step, got, want)
			}
		}
		g.SetStep(DeclareAttackers)
		other := g.players[1]
		if other.PlayerID() == p.PlayerID() {
			other = g.players[0]
		}
		if ability.CanActivate(other.PlayerID(), g) {
			t.Fatal("turn restriction lost")
		}
	}
}

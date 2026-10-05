package mage

import (
	"fmt"
	"testing"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestSpellCastingConditionPaths(t *testing.T) {
	for _, own := range []bool{true, false} {
		for _, index := range []int{0, 1} {
			for _, path := range []string{"hand", "free", "alternate", "copy cast"} {
				t.Run(fmt.Sprintf("own%v/player%d/%s", own, index, path), func(t *testing.T) {
					g := newPriorityTestGame()
					g.SetStep(DeclareAttackers)
					p := g.players[index]
					condition := IsYourTurn()
					if !own {
						condition = NotCondition(condition)
					}
					card := NewInstant("Restricted", "{U}", WithAbility(NewAction(ActionSpell, WithEffect(GainLifeFor(You, Fixed(1))), WithCastingCondition(condition), WithStepOnly(DeclareAttackers))))
					card.SetOwner(p.PlayerID())
					p.AddToHand(card)
					p.ManaPool().Add(Blue, 1)
					want := (index == 0) == own
					if got := len(g.GetCastableSpells(p.PlayerID())) == 1; got != want {
						t.Fatalf("query = %v, want %v", got, want)
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
					if !want && (len(p.Hand()) != 1 || p.ManaPool().TotalMana() != 1 || !g.stack.IsEmpty()) {
						t.Fatal("rejected cast changed resources")
					}
				})
			}
		}
	}
}

func TestSpellCastingConditionCopiesAndValidation(t *testing.T) {
	g := newPriorityTestGame()
	p := g.players[0]
	card := NewInstant("Restricted", "", WithAbility(NewAction(ActionSpell, WithEffect(GainLifeFor(You, Fixed(2))), WithCastingCondition(IsYourTurn()))))
	card.SetOwner(p.PlayerID())
	p.AddToHand(card)
	if err := g.CastSpellByID(p.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.turns.SetActivePlayerIndex(1)
	clone := g.Clone()
	if clone.CopySpellOnStack(card.ID(), g.players[1].PlayerID(), false) == nil {
		t.Fatal("copy repeated casting check")
	}
	clone.ResolveStack()
	if clone.players[0].Life() != 22 || clone.players[1].Life() != 22 || p.Life() != 20 {
		t.Fatal("copy controller or clone isolation")
	}
	g.ResolveStack()
	if p.Life() != 22 {
		t.Fatal("resolution repeated casting check")
	}
	copied := card.Copy()
	p.AddToHand(copied)
	if err := g.CastSpellByID(p.PlayerID(), copied.ID(), nil, 0); err == nil {
		t.Fatal("card copy lost condition")
	}
	for _, condition := range []StateCondition{nil, NotCondition(nil), PermanentIsTapped(DeclarePermanentTarget("unbound", IsCreature))} {
		invalid := NewInstant("Invalid", "", WithAbility(NewAction(ActionSpell, WithEffect(GainLifeFor(You, Fixed(1))), WithCastingCondition(condition))))
		if ValidateCardDefinition(invalid.Name(), invalid) == nil {
			t.Fatal("invalid condition accepted")
		}
	}
	invalid := NewCreature("Invalid activation", "", 1, 1, WithAbility(NewAction(ActionActivated, WithEffect(GainLifeFor(You, Fixed(1))), WithCastingCondition(IsYourTurn()))))
	if ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("casting option accepted on activation")
	}
}

func TestSpellCastingStepsAndConjunction(t *testing.T) {
	steps := []PhaseStep{Upkeep, DeclareAttackers}
	condition := IsDuringSteps(steps...)
	steps[0] = Cleanup
	for step := Untap; step <= Cleanup; step++ {
		for _, index := range []int{0, 1} {
			g := newPriorityTestGame()
			g.SetStep(step)
			p := g.players[index]
			card := NewInstant("Conditions", "", WithAbility(NewAction(ActionSpell, WithEffect(GainLifeFor(You, Fixed(1))), WithCastingCondition(condition), WithCastingCondition(IsYourTurn()))))
			card.SetOwner(p.PlayerID())
			p.AddToHand(card)
			want := index == 0 && (step == Upkeep || step == DeclareAttackers)
			if err := g.CastSpellByID(p.PlayerID(), card.ID(), nil, 0); (err == nil) != want {
				t.Fatalf("step %s, player %d: %v", step, index, err)
			}
		}
	}
	for _, condition := range []StateCondition{IsDuringSteps(), IsDuringSteps(PhaseStep(-1)), IsDuringSteps(Cleanup + 1)} {
		invalid := NewInstant("Invalid", "", WithAbility(NewAction(ActionSpell, WithEffect(GainLifeFor(You, Fixed(1))), WithCastingCondition(condition))))
		if ValidateCardDefinition(invalid.Name(), invalid) == nil {
			t.Fatal("invalid step accepted")
		}
	}
	g := newPriorityTestGame()
	g.SetStep(DeclareAttackers)
	p := g.players[0]
	card := NewSorcery("Restricted sorcery", "", WithAbility(NewAction(ActionSpell, WithEffect(GainLifeFor(You, Fixed(1))), WithCastingCondition(IsDuringSteps(DeclareAttackers)))))
	card.SetOwner(p.PlayerID())
	p.AddToHand(card)
	if len(g.GetCastableSpells(p.PlayerID())) != 0 {
		t.Fatal("condition granted sorcery timing")
	}
	if err := g.CastSpellByID(p.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("condition granted sorcery timing")
	}
	target := DeclarePermanentTarget("target", IsCreature)
	invalid := NewInstant("Target at casting", "", WithAbility(Spell(TapSubject(target), WithTargets(target))))
	WithCastingCondition(PermanentIsTapped(target))(invalid.Abilities()[0].(*ActionDefinition))
	if ValidateCardDefinition(invalid.Name(), invalid) == nil {
		t.Fatal("preannouncement target condition accepted")
	}
}

func TestSpellAfterFirstUpkeep(t *testing.T) {
	g := newPriorityTestGame()
	p := g.players[0]
	condition := AfterFirstUpkeep()
	card := NewInstant("After upkeep", "", WithAbility(NewAction(ActionSpell, WithEffect(GainLifeFor(You, Fixed(1))), WithCastingCondition(condition))))
	check := func(game *Game, want bool) {
		t.Helper()
		if got := game.checkSpellCastingConditions(card, p.PlayerID()) == nil; got != want {
			t.Fatalf("after upkeep = %v, want %v", got, want)
		}
	}
	check(g, false)
	g.SetStep(Upkeep)
	check(g, false)
	clone := g.Clone()
	g.RunStepWithPriority(Upkeep)
	check(g, true)
	check(clone, false)
	g.SetStep(Upkeep)
	check(g, true)
	g.RunStepWithPriority(Upkeep)
	check(g, true)
	g.turns.IncrementTurn()
	g.SetStep(Untap)
	check(g, false)
	g.SetStep(Draw)
	check(g, false)
	g.SetStep(Upkeep)
	check(g, false)
	g.SetStep(PrecombatMain)
	check(g, true)
	g.SetStep(Upkeep)
	g.SetTurn(g.CurrentTurn() + 1)
	g.SetStep(Untap)
	check(g, false)
	g.turns.Schedule().SkipNextStep[Upkeep] = 1
	g.turns.BuildNextTurn()
	for {
		step, ok := g.turns.PopNextStep()
		if !ok || step == Draw {
			break
		}
		g.RunStepWithPriority(step)
	}
	g.SetStep(Draw)
	check(g, false)
	g.RunStepWithPriority(Upkeep)
	check(g, true)
}

func TestSpellCastingForeignController(t *testing.T) {
	g := newPriorityTestGame()
	owner, caster := g.players[0], g.players[1]
	card := NewInstant("Foreign-owned", "", WithAbility(NewAction(ActionSpell, WithEffect(GainLifeFor(You, Fixed(1))), WithCastingCondition(NotCondition(IsYourTurn())))))
	card.SetOwner(owner.PlayerID())
	if _, err := g.pushCastSpellObject(castStackObjectOptions{Card: card, Controller: caster.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if caster.Life() != 21 || owner.Life() != 20 {
		t.Fatal("casting condition or resolution used owner")
	}
}

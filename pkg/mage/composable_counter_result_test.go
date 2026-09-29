package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestCounterRemovalResultAmounts(t *testing.T) {
	for _, amount := range []int{-2, 0, 2, 5, 300} {
		t.Run(Fixed(amount).Text(), func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			p := g.PutOnBattlefield(NewArtifact("Subject", ""), a.PlayerID())
			p.AddCounter(Charge, 5)
			p.AddCounter(Tide, 2)
			result := BindNumber("removed")
			body := Sequence(RemoveCountersFromWithResult(This, Charge, Fixed(amount), result), GainLifeFor(You, FromNumber(result)))
			if err := ApplyEffect(g, body, p.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			want := min(max(amount, 0), 5)
			if a.Life() != 20+want || int(g.FindPermanent(p.ID()).Counters[Charge]) != 5-want || g.FindPermanent(p.ID()).Counters[Tide] != 2 {
				t.Fatal("incorrect actual removal result")
			}
		})
	}
}

func TestCounterRemovalResultLifetime(t *testing.T) {
	for _, state := range []string{"present", "absent", "returned", "phased", "phase cycle"} {
		t.Run(state, func(t *testing.T) {
			a := NewBasePlayer("A")
			g := NewGame(a, NewBasePlayer("B"))
			card := NewArtifact("Subject", "")
			p := g.PutOnBattlefield(card, a.PlayerID())
			p.AddCounter(Charge, 5)
			result := BindNumber("removed")
			ctx := &EffectContext{Game: g, SourceID: p.ID(), Controller: a.PlayerID(), sourceCaptured: true, sourceIncarnation: p.incarnationID}
			effect := RemoveCountersFromWithResult(This, Charge, Fixed(3), result)
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "absent":
				g.RemoveFromBattlefield(p)
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID()).AddCounter(Charge, 5)
			case "phased":
				g.PhaseOut(p.ID())
			case "phase cycle":
				g.PhaseOut(p.ID())
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			want := 0
			if state == "present" || state == "phase cycle" {
				want = 2
			}
			if got := g.resolution.numberBindings[result]; got != want {
				t.Fatalf("result=%d want %d", got, want)
			}
			if state == "returned" && g.FindPermanent(card.ID()).Counters[Charge] != 5 {
				t.Fatal("old reference removed returned counters")
			}
		})
	}
}

func TestCounterRemovalResultTargetCopyClone(t *testing.T) {
	g, a, b := newSpellCopyGame()
	result := BindNumber("removed")
	target := DeclarePermanentTarget("target", IsCreature)
	ability := Spell(Sequence(RemoveCountersFromWithResult(target, Charge, Fixed(3), result), GainLifeFor(You, FromNumber(result))), WithTargets(target))
	spell := NewInstant("Removal", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	victim := NewCreature("Victim", "", 2, 2)
	victim.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(victim, b.PlayerID())
	p.AddCounter(Charge, 5)
	g.pushStack(&StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{p.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()})
	if g.CopySpellOnStack(spell.ID(), a.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	clone.MutablePermanent(p.ID()).RemoveCounter(Charge, 4)
	clone.ResolveStack()
	g.ResolveStack()
	if a.Life() != 25 || clone.GetPlayer(a.PlayerID()).Life() != 21 || g.FindPermanent(p.ID()).Counters[Charge] != 0 {
		t.Fatal("copy or clone result isolation failed")
	}
}

func TestCounterRemovalResultDeclaration(t *testing.T) {
	b := BindNumber("removed")
	card := func(effect ResolvingEffect) Card {
		return NewArtifact("Synthetic", "", WithAbility(Activate(Costs(), effect)))
	}
	good := RemoveCountersFromWithResult(This, Charge, Fixed(2), b)
	RunDefinitionValidationTable(t, []DefinitionValidationCase{
		{Name: "valid", Card: card(Sequence(good, GainLifeFor(You, FromNumber(b))))},
		{Name: "nil subject", Card: card(RemoveCountersFromWithResult(nil, Charge, Fixed(2), b)), WantErr: true, WantErrSubstr: "subject"},
		{Name: "nil amount", Card: card(RemoveCountersFromWithResult(This, Charge, nil, b)), WantErr: true, WantErrSubstr: "amount"},
		{Name: "nil result", Card: card(RemoveCountersFromWithResult(This, Charge, Fixed(2), nil)), WantErr: true, WantErrSubstr: "binding"},
		{Name: "unnamed", Card: card(RemoveCountersFromWithResult(This, Charge, Fixed(2), BindNumber(""))), WantErr: true, WantErrSubstr: "name"},
		{Name: "invalid counter", Card: card(RemoveCountersFromWithResult(This, NumCounters, Fixed(2), b)), WantErr: true, WantErrSubstr: "counter"},
		{Name: "negative counter", Card: card(RemoveCountersFromWithResult(This, CounterType(-1), Fixed(2), b)), WantErr: true, WantErrSubstr: "counter"},
		{Name: "before producer", Card: card(Sequence(GainLifeFor(You, FromNumber(b)), good)), WantErr: true, WantErrSubstr: "producer"},
		{Name: "conditional producer", Card: card(Sequence(IfPositive(Fixed(1), good), GainLifeFor(You, FromNumber(b)))), WantErr: true, WantErrSubstr: "producer"},
	})
}

func TestCounterRemovalResultIllegalTargetContinues(t *testing.T) {
	for _, state := range []string{"present", "returned", "phased", "shroud"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Victim", "", 2, 2)
			card.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(card, b.PlayerID())
			p.AddCounter(Charge, 5)
			target := DeclarePermanentTarget("creature", IsCreature)
			player := DeclarePlayerTarget("player")
			result := BindNumber("removed")
			spell := NewInstant("Synthetic removal", "", Spell(Sequence(
				RemoveCountersFromWithResult(target, Charge, Fixed(3), result),
				GainLifeFor(player, FromNumber(result)), GainLifeFor(player, Fixed(1)),
			), WithTargets(target, player)))
			a.AddToHand(spell)
			g.SetStep(PrecombatMain)
			if err := g.CastSpellByName(a.PlayerID(), spell.Name(), []uuid.UUID{p.ID(), a.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "returned":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, b.PlayerID()).AddCounter(Charge, 5)
			case "phased":
				g.PhaseOut(p.ID())
			case "shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(Shroud)
			}
			g.ResolveStack()
			want := 21
			if state == "present" {
				want = 24
			}
			if a.Life() != want {
				t.Fatalf("life=%d want %d", a.Life(), want)
			}
			if state == "phased" {
				g.PhaseIn([]uuid.UUID{p.ID()})
			}
			if state != "present" && g.FindPermanent(p.ID()).Counters[Charge] != 5 {
				t.Fatal("illegal target lost counters")
			}
		})
	}
}

func TestCounterRemovalResultActivationAndTrigger(t *testing.T) {
	for _, trigger := range []bool{false, true} {
		a := NewBasePlayer("A")
		g := NewGame(a, NewBasePlayer("B"))
		result := BindNumber("removed")
		body := Sequence(RemoveCountersFromWithResult(This, Charge, Fixed(3), result), GainLifeFor(You, FromNumber(result)))
		var ability Ability = Activate(Costs(), body)
		if trigger {
			ability = At(YourUpkeep, body)
		}
		card := NewArtifact("Removal source", "", WithAbility(ability))
		card.SetOwner(a.PlayerID())
		p := g.PutOnBattlefield(card, a.PlayerID())
		p.AddCounter(Charge, 5)
		if trigger {
			g.FireEvent(GameEvent{Type: EvtUpkeep, PlayerID: a.PlayerID()})
		} else {
			g.SetStep(PrecombatMain)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
		}
		g.ResolveStack()
		if a.Life() != 23 || g.FindPermanent(p.ID()).Counters[Charge] != 2 {
			t.Fatal("stack result failed")
		}
	}
}

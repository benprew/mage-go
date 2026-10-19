package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestManaPaymentInstructionDeclaration(t *testing.T) {
	target := DeclareSpellTarget("spell")
	for _, cost := range []string{"{1}", "{X}"} {
		instruction := ManaPaymentInstruction{Player: ControllerOfSpell(target), Cost: core.ParseManaCost(cost), Unpaid: Counter(target)}
		if cost == "{X}" {
			instruction.Amount = Fixed(3)
		}
		effect := instruction.Effect()
		got, ok := InspectManaPaymentInstruction(effect)
		if !ok || got.Player != instruction.Player || got.Cost.Generic != instruction.Cost.Generic || got.Cost.HasX != instruction.Cost.HasX || got.Amount != instruction.Amount || got.Unpaid != instruction.Unpaid {
			t.Fatal("lost payment operands")
		}
	}
	var missing *PlayerTargetBinding
	for _, instruction := range []ManaPaymentInstruction{
		{}, {Player: missing, Unpaid: Counter(target)},
		{Player: You, Cost: core.ParseManaCost("{X}"), Unpaid: Counter(target)},
		{Player: You, Amount: Fixed(1), Unpaid: Counter(target)},
		{Player: You, Cost: core.ManaCost{Generic: -1}, Unpaid: Counter(target)},
		{Player: You, Cost: core.ParseManaCost("{1}")},
	} {
		if instruction.Effect() != nil {
			t.Fatal("accepted invalid declaration")
		}
	}
	for _, effect := range []ResolvingEffect{nil, (*resolutionManaPayment)(nil), (*unlessPaidEffect)(nil), Sequence(Counter(target), destructionInspectionTrap{}), UnlessPaid(You, Costs(LifePayCost(1)), Counter(target))} {
		if _, ok := InspectManaPaymentInstruction(effect); ok {
			t.Fatal("accepted non-mana or incomplete payment")
		}
	}
}

func TestManaPaymentInstructionSpell(t *testing.T) {
	for _, state := range []string{"alone", "prefix", "suffix", "illegal prefix", "unknown paid", "unknown unpaid", "foreign payer", "foreign branch", "missing zone", "duplicate", "all illegal", "opaque amount", "opaque collection", "foreign member", "wrong saved subject"} {
		t.Run(state, func(t *testing.T) {
			g, victim, obj, target := counterInstructionFixture(t)
			land := g.PutOnBattlefield(NewLand("Land"), obj.Controller)
			watched := []PermanentVisit{{land.ID(), land.incarnationID}}
			instruction := ManaPaymentInstruction{Player: ControllerOfSpellWithLKI(target), Cost: core.ParseManaCost("{X}"), Amount: Fixed(0), Unpaid: Counter(target)}
			certain, known := false, true
			switch state {
			case "unknown paid":
				instruction.Paid = destructionInspectionTrap{}
				known = false
			case "unknown unpaid":
				instruction.Unpaid = destructionInspectionTrap{}
				known = false
			case "foreign payer":
				instruction.Player = ControllerOfSpell(DeclareSpellTarget("spell"))
				known = false
			case "foreign branch":
				instruction.Unpaid = Counter(DeclareSpellTarget("spell"))
				known = false
			case "opaque amount":
				instruction.Amount = FromNumber(BindNumber("absent"))
				known = false
			case "missing zone":
				obj.TargetZones = nil
				known = false
			case "duplicate":
				obj.TargetSpecs = append(obj.TargetSpecs, target.targetSpec())
				obj.Targets = append(obj.Targets, victim.ID)
				known = false
			case "all illegal", "illegal prefix":
				g.GetStack().RemoveByID(victim.ID)
			case "opaque collection", "foreign member", "wrong saved subject":
				member := BindPlayer("payer")
				collection := PermanentsControlledBy(FromPlayer(member), IsLand)
				if state == "foreign member" {
					collection = PermanentsControlledBy(FromPlayer(BindPlayer("payer")), IsLand)
				}
				if state == "opaque collection" {
					collection = PermanentsControlledBy(FromPlayer(member), NewPermanentFilter("opaque", func(*Permanent, *Game) bool { panic("filter callback") }))
				}
				subject := instruction.Player
				if state == "wrong saved subject" {
					subject = You
				}
				instruction.Unpaid = Sequence(Counter(target), (SavedPlayerInstruction{Subject: subject, Binding: member, Body: Sequence((TapInstruction{Subjects: collection, Operation: TapOperation}).Effect(), LoseUnspentManaFor(FromPlayer(member)))}).Effect())
				known = false
			}
			payment := instruction.Effect()
			obj.Effects = []Effect{payment}
			switch state {
			case "prefix":
				obj.Effects = []Effect{payment, DestroyEach(MatchingPermanents(IsLand))}
				known = false
			case "illegal prefix":
				second := DeclarePermanentTarget("land", IsLand)
				obj.TargetSpecs = append(obj.TargetSpecs, second.targetSpec())
				obj.Targets = append(obj.Targets, land.ID())
				obj.TargetZones[land.ID()] = core.ZoneBattlefield
				obj.targetIncarnations = map[uuid.UUID]uuid.UUID{land.ID(): land.incarnationID}
				obj.Effects = []Effect{payment, Destroy(second)}
				certain = true
			case "suffix":
				obj.Effects = []Effect{DestroyEach(MatchingPermanents(IsLand)), payment}
				certain = true
			}
			for range 2 {
				got, gotKnown := g.InspectSpellDestruction(obj, watched)
				if got != certain || gotKnown != known {
					t.Fatalf("got %v/%v want %v/%v", got, gotKnown, certain, known)
				}
			}
		})
	}
}

func TestManaPaymentInstructionBranches(t *testing.T) {
	for _, cost := range []string{"{1}", "{X}"} {
		for _, accept := range []bool{false, true} {
			a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: accept}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			a.ManaPool().Add(core.Colorless, 3)
			instruction := ManaPaymentInstruction{Player: You, Cost: core.ParseManaCost(cost), Paid: GainLifeFor(You, Fixed(2)), Unpaid: LoseLifeFor(You, Fixed(3))}
			if cost == "{X}" {
				instruction.Amount = Fixed(2)
			}
			effect := instruction.Effect()
			card := NewInstant("Payment", "", Spell(effect))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			wantLife, wantMana := 17, 3
			if accept {
				wantLife = 22
				wantMana = 2
				if cost == "{X}" {
					wantMana = 1
				}
			}
			if a.Life() != wantLife || a.ManaPool().TotalMana() != wantMana || b.Life() != 20 {
				t.Fatal("wrong branch or payment")
			}
		}
	}
}

func TestManaPaymentInstructionHybridIsolation(t *testing.T) {
	instruction := ManaPaymentInstruction{Player: You, Cost: core.ParseManaCost("{W/U}"), Unpaid: GainLifeFor(You, Fixed(1))}
	effect := instruction.Effect()
	instruction.Cost.Hybrid[0].A = core.Red
	got, ok := InspectManaPaymentInstruction(effect)
	if !ok || got.Cost.Hybrid[0].A != core.White {
		t.Fatal("effect shares input cost")
	}
	got.Cost.Hybrid[0].A = core.Green
	again, ok := InspectManaPaymentInstruction(effect)
	if !ok || again.Cost.Hybrid[0].A != core.White {
		t.Fatal("inspector shares effect cost")
	}
}

func TestManaPaymentInstructionCollection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("Mana", WithManaAbility(core.Green)), a.PlayerID())
	plain := g.PutOnBattlefield(NewLand("Empty"), a.PlayerID())
	predicate, ok := InspectContextualPermanentCollection(MatchingPermanents(HasManaAbility))
	if !ok {
		t.Fatal("native mana predicate missing")
	}
	for _, state := range []string{"present", "phase", "return", "removed", "granted"} {
		switch state {
		case "phase":
			g.PhaseOut(land.ID())
		case "return":
			g.PhaseIn([]uuid.UUID{land.ID()})
		case "removed":
			g.MutablePermanent(land.ID()).RuntimeAbilities = nil
		case "granted":
			g.MutablePermanent(plain.ID()).RuntimeAbilities = append(g.FindPermanent(plain.ID()).RuntimeAbilities, NewManaAbility(core.Blue))
		}
		want := 0
		for _, p := range g.AllBattlefield() {
			if !p.PhasedOut && HasManaAbility.Match(p, g) {
				want++
			}
		}
		selected := predicate.Select(&EffectContext{Game: g, Controller: a.PlayerID()})
		if len(selected) != want {
			t.Fatalf("%s: native collection differs", state)
		}
	}
}

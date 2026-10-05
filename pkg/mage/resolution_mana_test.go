package mage

import (
	"fmt"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestResolutionManaPayment(t *testing.T) {
	for _, scenario := range []string{"pay", "decline", "unaffordable", "zero-pay", "zero-decline", "negative", "mana-source", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: scenario != "decline" && scenario != "zero-decline"}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			amount := 2
			if scenario == "zero-pay" || scenario == "zero-decline" {
				amount = 0
			}
			if scenario == "negative" {
				amount = -2
			}
			cost := "{X}{X}{U}{1}"
			if scenario == "zero-pay" || scenario == "zero-decline" || scenario == "negative" || scenario == "mana-source" {
				cost = "{X}"
			}
			a.ManaPool().Add(Blue, 1)
			a.ManaPool().Add(Colorless, 5)
			if scenario == "unaffordable" {
				a.ManaPool().Clear()
				a.ManaPool().Add(Blue, 1)
				a.ManaPool().Add(Colorless, 4)
			}
			var land *Permanent
			if scenario == "mana-source" {
				amount = 1
				a.ManaPool().Clear()
				land = g.PutOnBattlefield(NewLand("Mana", WithManaAbility(Green)), a.PlayerID())
			}
			before := a.ManaPool().TotalMana()
			effect := UnlessManaPaid(You, ParseManaCost(cost), Fixed(amount), GainLifeFor(You, Fixed(3)))
			if scenario == "clone" {
				clone := g.Clone()
				if err := ApplyEffect(clone, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().TotalMana() != before {
					t.Fatal("clone spent live mana")
				}
			}
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			paid := scenario != "decline" && scenario != "zero-decline" && scenario != "unaffordable"
			wantLife := 23
			if paid {
				wantLife = 20
			}
			if a.Life() != wantLife {
				t.Fatalf("life = %d, want %d", a.Life(), wantLife)
			}
			wantMana := before
			if paid && cost != "{X}" {
				wantMana -= 6
			}
			if a.ManaPool().TotalMana() != wantMana {
				t.Fatal("payment spent the wrong amount")
			}
			if land != nil && !g.FindPermanent(land.ID()).Tapped {
				t.Fatal("payment did not use the mana service")
			}
		})
	}
}

func TestResolutionManaPaymentBindingsAndDeclarations(t *testing.T) {
	spell := DeclareSpellTarget("spell")
	valid := NewInstant("Payment", "", Spell(UnlessManaPaid(ControllerOfSpell(spell), ParseManaCost("{X}"), ManaValueOfSpellWithLKI(spell), Counter(spell)), WithTargets(spell)))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{
		UnlessManaPaid(nil, ParseManaCost("{X}"), Fixed(2), GainLifeFor(You, Fixed(1))),
		UnlessManaPaid(You, ParseManaCost("{X}"), nil, GainLifeFor(You, Fixed(1))),
		UnlessManaPaid(You, ParseManaCost("{X}"), Fixed(2), nil),
		UnlessManaPaid(You, ParseManaCost("{1}"), Fixed(2), GainLifeFor(You, Fixed(1))),
		UnlessManaPaid(You, ManaCost{HasX: true, XCount: 1, Generic: -1}, Fixed(2), GainLifeFor(You, Fixed(1))),
		UnlessManaPaid(You, ParseManaCost("{X}"), ManaValueOfSpellWithLKI(spell), GainLifeFor(You, Fixed(1))),
	} {
		c := NewInstant("Invalid", "", Spell(effect))
		if ValidateCardDefinition(c.Name(), c) == nil {
			t.Fatal("invalid payment definition accepted")
		}
	}
	a := &optionalPaymentPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	a.ManaPool().Add(Colorless, 3)
	effect := Sequence(IfManaPaid(You, ParseManaCost("{X}"), Fixed(2), GainLifeFor(You, Fixed(4))), UnlessManaPaid(You, ParseManaCost("{X}"), Fixed(2), GainLifeFor(You, Fixed(1))))
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 25 || a.ManaPool().TotalMana() != 1 {
		t.Fatal("separate payment outcomes were lost")
	}
}

func TestResolutionManaPaymentDoesNotReduce(t *testing.T) {
	a := &countedMayPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	p := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
	g.effects.Rules.ActivationCostReductions[p.ID()] = 2
	a.ManaPool().Add(Colorless, 1)
	if err := ApplyEffect(g, UnlessManaPaid(You, ParseManaCost("{X}"), Fixed(3), GainLifeFor(You, Fixed(1))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.offers != 0 || a.Life() != 21 || a.ManaPool().TotalMana() != 1 {
		t.Fatal("activation reduction changed resolution payment")
	}
}

func TestSpellManaValueDeparture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	spell := NewInstant("Spell", "{X}{X}{U}")
	obj := newStackObject(a.PlayerID(), spell.ID(), spell, nil, nil, 2, false)
	g.stack.Push(obj)
	binding := BindSpell("spell")
	g.resolution.spellBindings = map[*SpellBinding]uuid.UUID{binding: obj.ID}
	value := ManaValueOfSpellWithLKI(FromSpell(binding))
	ctx := &EffectContext{Game: g, Controller: b.PlayerID()}
	if value.resolveValue(ctx) != 5 {
		t.Fatal("live mana value lost X")
	}
	obj.XValue = 3
	g.stack.RemoveByID(obj.ID)
	returned := newStackObject(a.PlayerID(), spell.ID(), spell, nil, nil, 9, false)
	g.stack.Push(returned)
	if value.resolveValue(ctx) != 7 {
		t.Fatal("departure mana value followed the returned spell")
	}
	clone := g.Clone()
	if value.resolveValue(&EffectContext{Game: clone, Controller: b.PlayerID()}) != 7 {
		t.Fatal("clone lost spell information")
	}
	target := DeclareSpellTarget("target")
	ctx.targetBindings = map[*targetDeclaration][]capturedPermanent{target.targetDeclaration: {{id: obj.ID}}}
	if ManaValueOfSpellWithLKI(target).resolveValue(ctx) != 7 {
		t.Fatal("typed target lost departure information")
	}
	if ManaValueOfSpell(FromSpell(binding)).resolveValue(ctx) != 0 || ManaValueOfSpell(target).resolveValue(ctx) != 0 {
		t.Fatal("live-only queries changed their departure contract")
	}
}

func TestSpellManaValueRemovalPaths(t *testing.T) {
	for _, remove := range []string{"pop", "source", "id", "ante"} {
		t.Run(remove, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewInstant("Spell", "{X}{U}")
			card.SetOwner(a.PlayerID())
			obj := newStackObject(a.PlayerID(), card.ID(), card, nil, nil, 4, false)
			g.stack.Push(obj)
			binding := BindSpell("spell")
			g.resolution.spellBindings = map[*SpellBinding]uuid.UUID{binding: obj.ID}
			switch remove {
			case "pop":
				g.stack.Pop()
			case "source":
				g.stack.RemoveBySourceID(card.ID())
			case "id":
				g.stack.RemoveByID(obj.ID)
			case "ante":
				if _, _, _, err := g.removeCardForAnte(card.ID(), ZoneStack); err != nil {
					t.Fatal(err)
				}
			}
			value := ManaValueOfSpellWithLKI(FromSpell(binding))
			if value.resolveValue(&EffectContext{Game: g}) != 5 {
				t.Fatal("removal did not save mana value")
			}
			clone := g.Clone()
			other := newStackObject(a.PlayerID(), card.ID(), card, nil, nil, 7, false)
			clone.stack.Push(other)
			clone.stack.Pop()
			if len(g.stack.spellManaValues) != 1 || len(clone.stack.spellManaValues) != 2 {
				t.Fatal("clones share spell information")
			}
			if _, ok := FromSpell(binding).resolveStackObject(&EffectContext{Game: g}); ok {
				t.Fatal("information restored a departed spell")
			}
		})
	}
}

type resolutionManaChoicePlayer struct {
	*BasePlayer
	choose func()
	offers int
}

func (p *resolutionManaChoicePlayer) ChooseMayAbility(string) bool {
	p.offers++
	p.choose()
	return true
}

func TestResolutionManaPaymentCaptureAndRollback(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			a := &resolutionManaChoicePlayer{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			a.AddToHand(NewLand("First"))
			a.AddToHand(NewLand("Second"))
			a.ManaPool().Add(Colorless, 2)
			a.choose = func() {
				a.AddToHand(NewLand("Later"))
				if fail {
					a.ManaPool().Clear()
					a.ManaPool().Add(Colorless, 1)
				}
			}
			if err := ApplyEffect(g, UnlessManaPaid(You, ParseManaCost("{X}"), HandSizeOf(You), GainLifeFor(You, Fixed(1))), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			wantMana, wantLife := 0, 20
			if fail {
				wantMana, wantLife = 1, 21
			}
			if a.ManaPool().TotalMana() != wantMana || a.Life() != wantLife || a.offers != 1 {
				t.Fatal("amount changed during choice or failed payment spent resources")
			}
		})
	}
}

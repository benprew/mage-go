package mage

import (
	"strconv"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

type chosenManaPlayer struct {
	*BasePlayer
	amount int
	calls  int
	before func()
}

func (p *chosenManaPlayer) ChooseNonnegativeNumber(_ string) int {
	p.calls++
	if p.before != nil {
		p.before()
	}
	return p.amount
}
func (p *chosenManaPlayer) ChooseMayAbility(string) bool { panic("extra optional offer") }

func TestChosenManaPayment(t *testing.T) {
	for _, amount := range []int{-1, 0, 1, 2, 3, 5} {
		t.Run(strconv.Itoa(amount), func(t *testing.T) {
			a := &chosenManaPlayer{BasePlayer: NewBasePlayer("A"), amount: amount}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			a.ManaPool().Add(Blue, 3)
			result := BindNumber("paid")
			effect := Sequence(PayChosenManaInto(You, result), GainLifeFor(You, FromNumber(result)))
			if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			paid := amount
			if amount < 0 || amount > 3 {
				paid = 0
			}
			if a.Life() != 20+paid || a.ManaPool().TotalMana() != 3-paid || a.calls != 1 {
				t.Fatal("incorrect payment result")
			}
		})
	}
}

func TestChosenManaPaymentManaAbilitiesAndRollback(t *testing.T) {
	for _, short := range []bool{false, true} {
		a := &chosenManaPlayer{BasePlayer: NewBasePlayer("A"), amount: 2}
		g := NewGame(a, NewBasePlayer("B"))
		land := g.PutOnBattlefield(NewLand("mana", WithManaAbility(Green)), a.PlayerID())
		a.ManaPool().Add(Blue, 1)
		if short {
			a.before = func() { a.ManaPool().Clear() }
		}
		result := BindNumber("paid")
		effect := Sequence(PayChosenManaInto(You, result), GainLifeFor(You, FromNumber(result)))
		if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if short {
			if a.Life() != 20 || g.FindPermanent(land.ID()).Tapped {
				t.Fatal("failed payment spent resources")
			}
		} else if a.Life() != 22 || !g.FindPermanent(land.ID()).Tapped || a.ManaPool().TotalMana() != 0 {
			t.Fatal("mana ability failed to pay")
		}
	}
}

func TestInstructionDamagePrevention(t *testing.T) {
	for _, n := range []int{0, 1, 2, 3} {
		a := NewBasePlayer("A")
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		source := g.PutOnBattlefield(NewCreature("source", "", 1, 3), a.PlayerID())
		effect := DealDamageWithPreventionFrom(DamageSourceOf(This), You, Fixed(2), Fixed(n))
		if err := ApplyEffect(g, effect, source.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if b.Life() != 20-max(0, 2-n) {
			t.Fatal("incorrect prevention")
		}
		g.DealDamageToPlayer(b, 2, source.ID())
		if b.Life() != 18-max(0, 2-n) {
			t.Fatal("instruction prevention leaked")
		}
	}
}

func TestInstructionDamagePreventionOrdering(t *testing.T) {
	for _, choice := range []int{0, 1} {
		a := &auditChoicePlayer{BasePlayer: NewBasePlayer("A")}
		b := &auditChoicePlayer{BasePlayer: NewBasePlayer("B"), choice: choice}
		g := NewGame(a, b)
		p := g.PutOnBattlefield(NewArtifact("source", ""), a.PlayerID())
		g.effects.AddReplacement(&auditReplacement{multiply: 2})
		effect := DealDamageWithPreventionFrom(DamageSourceOf(This), You, Fixed(2), Fixed(1))
		if err := ApplyEffect(g, effect, p.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := 17
		if choice == 1 {
			want = 18
		}
		if b.Life() != want || b.calls != 1 || a.calls != 0 {
			t.Fatalf("life=%d choices=%d/%d", b.Life(), a.calls, b.calls)
		}
	}
}

func TestInstructionDamagePreventionRedirectAndProhibition(t *testing.T) {
	for _, state := range []string{"redirect", "blocked", "normal", "zero", "negative"} {
		a := NewBasePlayer("A")
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		g.PutOnBattlefield(NewArtifact("source", ""), a.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("victim", "", 1, 5), b.PlayerID())
		if state == "redirect" {
			g.SetCreatureDamageRedirect(victim.ID(), b.PlayerID())
		}
		if state == "blocked" {
			g.MutablePermanent(victim.ID()).GrantBaseAttr(AttrDamageCantBePreventedOrRedirected)
		}
		n := 2
		if state == "zero" {
			n = 0
		}
		if state == "negative" {
			n = -1
		}
		effect := DealDamageWithPreventionFrom(DamageSourceOf(This), This, Fixed(n), Fixed(1))
		if err := ApplyEffect(g, effect, victim.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		wantDamage, wantLife := 1, 20
		switch state {
		case "redirect":
			wantDamage, wantLife = 0, 19
		case "blocked":
			wantDamage = 2
		case "zero", "negative":
			wantDamage = 0
		}
		if g.FindPermanent(victim.ID()).Damage != wantDamage || b.Life() != wantLife {
			t.Fatal("wrong redirected or prohibited prevention")
		}
	}
}

func TestChosenManaPaymentBindingsAndClones(t *testing.T) {
	a := &chosenManaPlayer{BasePlayer: NewBasePlayer("A"), amount: 2}
	g := NewGame(a, NewBasePlayer("B"))
	a.ManaPool().Add(Green, 3)
	one, two := BindNumber("one"), BindNumber("two")
	effect := Sequence(PayChosenManaInto(You, one), PayChosenManaInto(You, two), GainLifeFor(You, FromNumber(one)), GainLifeFor(You, FromNumber(two)))
	clone := g.Clone()
	clone.players[0] = &chosenManaClonePlayer{Player: clone.GetPlayer(a.PlayerID()), amount: 1}
	if err := ApplyEffect(clone, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || a.ManaPool().TotalMana() != 3 {
		t.Fatal("clone changed original")
	}
	if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 22 || a.ManaPool().TotalMana() != 1 || clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 1 {
		t.Fatal("independent results lost")
	}
	g.resolution.numberBindings = map[*NumberBinding]int{one: 9}
	ctx := &EffectContext{Game: g, Controller: a.PlayerID()}
	a.amount = 5
	if err := PayChosenManaInto(You, one).Apply(ctx); err != nil {
		t.Fatal(err)
	}
	if FromNumber(one).resolveValue(ctx) != 0 {
		t.Fatal("failed payment retained old result")
	}
}

type chosenManaClonePlayer struct {
	Player
	amount int
}

func (p *chosenManaClonePlayer) ChooseNonnegativeNumber(string) int { return p.amount }

func TestChosenManaPaymentDefinition(t *testing.T) {
	binding := BindNumber("paid")
	for _, effect := range []Effect{
		PayChosenManaInto(nil, binding), PayChosenManaInto(You, nil),
		DealDamageWithPreventionFrom(nil, You, Fixed(2), Fixed(1)),
		DealDamageWithPreventionFrom(DamageSourceOf(This), nil, Fixed(2), Fixed(1)),
		DealDamageWithPreventionFrom(DamageSourceOf(This), You, nil, Fixed(1)),
		DealDamageWithPreventionFrom(DamageSourceOf(This), You, Fixed(2), nil),
		DealDamageWithPreventionFrom(DamageSourceOf(This), You, Fixed(2), FromNumber(binding)),
	} {
		card := NewArtifact("invalid", "", WithAbility(Activate(Costs(), Sequence(effect))))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid definition accepted")
		}
	}
	card := NewArtifact("valid", "", WithAbility(Activate(Costs(), Sequence(PayChosenManaInto(You, binding), DealDamageWithPreventionFrom(DamageSourceOf(This), You, Fixed(2), FromNumber(binding))))))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
}

type manaDigitPlayer struct {
	*BasePlayer
	digits []int
	index  int
	menus  int
}

func (p *manaDigitPlayer) ChooseMode(_ []string, _ string) int {
	p.menus++
	if p.index < len(p.digits) {
		return 1
	}
	return 0
}
func (p *manaDigitPlayer) ChooseNumber(minimum, maximum int, _ string) int {
	if minimum != 0 || maximum > 9 {
		panic("number menu is too large")
	}
	digit := p.digits[p.index]
	p.index++
	return digit
}
func TestChosenManaPaymentDigitDecision(t *testing.T) {
	for _, digits := range [][]int{nil, {0}, {1, 2, 3}, {9, 9, 9}} {
		a := &manaDigitPlayer{BasePlayer: NewBasePlayer("A"), digits: digits}
		g := NewGame(a, NewBasePlayer("B"))
		a.ManaPool().Add(Blue, 123)
		binding := BindNumber("paid")
		effect := Sequence(PayChosenManaInto(You, binding), GainLifeFor(You, FromNumber(binding)))
		if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		paid := 0
		if len(digits) == 3 && digits[0] == 1 {
			paid = 123
		}
		if a.Life() != 20+paid || a.ManaPool().TotalMana() != 123-paid || a.index != len(digits) {
			t.Fatal("digit decision failed")
		}
	}
}

func TestChosenManaPaymentLimitsAndRestrictions(t *testing.T) {
	for _, state := range []string{"maximum", "restricted", "absent"} {
		a := &chosenManaPlayer{BasePlayer: NewBasePlayer("A"), amount: 1}
		g := NewGame(a, NewBasePlayer("B"))
		binding := BindNumber("paid")
		player := PlayerRef(You)
		a.ManaPool().Add(Blue, 1)
		switch state {
		case "maximum":
			a.amount = int(^uint(0) >> 1)
		case "restricted":
			a.ManaPool().Clear()
			a.ManaPool().AddRestricted(Blue, 1, CreatureSpellsOnly{})
		case "absent":
			player = PlayerWithUniqueHighestLife()
		}
		if err := ApplyEffect(g, Sequence(PayChosenManaInto(player, binding), GainLifeFor(You, FromNumber(binding))), uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 20 || a.ManaPool().TotalMana() != 1 {
			t.Fatal("invalid payment spent restricted or insufficient mana")
		}
		if state == "absent" && a.calls != 0 {
			t.Fatal("absent payer made a choice")
		}
	}
	for _, input := range []string{strconv.Itoa(int(^uint(0) >> 1)), "123"} {
		digits := []int{}
		for _, digit := range input {
			digits = append(digits, int(digit-'0'))
		}
		p := &manaDigitPlayer{BasePlayer: NewBasePlayer("digits"), digits: digits}
		amount := chooseNonnegativeNumber(p, "test")
		expected, err := strconv.Atoi(input)
		if err != nil || amount != expected || p.index != len(digits) {
			t.Fatal("number representation lost")
		}
	}
}

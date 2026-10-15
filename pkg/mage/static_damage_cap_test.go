package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func damageCapFixture() Card {
	return NewArtifact("Damage cap", "", WithAbility(Static(LimitPlayerDamageFromTypes(You, 3, 2, core.TypeInstant, core.TypeSorcery, core.TypeArtifact))))
}

func TestStaticDamageCapPackets(t *testing.T) {
	for _, amount := range []int{-1, 0, 1, 2, 3, 10} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		g.PutOnBattlefield(damageCapFixture(), a.PlayerID())
		victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 20), a.PlayerID())
		source := NewSorcery("Source", "", nil)
		saved := DamageSource{id: source.ID(), card: source, controller: b.PlayerID()}
		other := NewInstant("Other source", "", nil)
		second := DamageSource{id: other.ID(), card: other, controller: b.PlayerID()}
		outcomes := g.dealSimultaneousDamageOutcomes(PlayerDamageAssignment(a, amount, saved), PlayerDamageAssignment(a, amount, second), PlayerDamageAssignment(b, amount, saved), PermanentDamageAssignment(victim, amount, saved))
		want := max(0, min(amount, 2))
		if a.Life() != 20-2*want || b.Life() != 20-max(0, amount) || victim.Damage != max(0, amount) {
			t.Fatalf("amount %d: life %d/%d, marked %d", amount, a.Life(), b.Life(), victim.Damage)
		}
		if amount > 0 && (outcomes[0].amount != want || outcomes[1].amount != want || outcomes[2].amount != amount) {
			t.Fatal("actual damage results differ from replaced amounts")
		}
	}
}

func TestStaticDamageCapSameSourceBatch(t *testing.T) {
	for _, amounts := range [][2]int{{1, 1}, {1, 2}, {3, 4}, {int(^uint(0) >> 1), 1}} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		g.PutOnBattlefield(damageCapFixture(), a.PlayerID())
		spell := NewInstant("Source", "", nil)
		source := DamageSource{id: spell.ID(), stackID: uuid.New(), card: spell, controller: b.PlayerID(), keywords: map[core.Attr]bool{core.Lifelink: true}}
		g.DealSimultaneousDamage(PlayerDamageAssignment(a, amounts[0], source), PlayerDamageAssignment(a, amounts[1], source))
		if a.Life() != 18 || b.Life() != 22 {
			t.Fatalf("amounts %v: life %d/%d, want 18/22", amounts, a.Life(), b.Life())
		}
	}
}

func TestStaticDamageCapSourceInformation(t *testing.T) {
	for _, change := range []string{"matching", "nonmatching", "lose-type", "gain-type", "depart", "return", "phase", "unknown"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			g.PutOnBattlefield(damageCapFixture(), a.PlayerID())
			card := NewArtifact("Source", "")
			if change == "nonmatching" || change == "gain-type" {
				card = NewCreature("Source", "", 1, 10)
			}
			source := g.PutOnBattlefield(card, b.PlayerID())
			saved := g.CaptureDamageSource(source.ID())
			want := 18
			switch change {
			case "nonmatching":
				want = 14
			case "lose-type":
				g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrIsArtifact)
				want = 14
			case "gain-type":
				g.MutablePermanent(source.ID()).GrantBaseAttr(core.AttrIsArtifact)
			case "depart", "return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "return" {
					g.PutOnBattlefield(card, b.PlayerID()).RevokeBaseAttr(core.AttrIsArtifact)
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "unknown":
				saved = DamageSource{}
				want = 14
			}
			g.DealDamageToPlayerFromSource(a, 6, saved)
			if a.Life() != want {
				t.Fatalf("life %d, want %d", a.Life(), want)
			}
		})
	}
}

type damageCapOrderPlayer struct {
	*BasePlayer
	first string
	calls int
	order *[]string
}

func (p *damageCapOrderPlayer) ChooseMode(options []string, _ string) int {
	p.calls++
	if p.order != nil {
		*p.order = append(*p.order, p.Name())
	}
	for i, label := range options {
		if strings.Contains(label, "Damage cap") == (p.first == "Damage cap") {
			return i
		}
	}
	return 0
}

func TestStaticDamageCapAPNAP(t *testing.T) {
	var order []string
	a := &damageCapOrderPlayer{BasePlayer: NewBasePlayer("A"), first: "Damage cap", order: &order}
	b := &damageCapOrderPlayer{BasePlayer: NewBasePlayer("B"), first: "Damage cap", order: &order}
	g := NewGame(a, b)
	g.PutOnBattlefield(damageCapFixture(), a.PlayerID())
	g.PutOnBattlefield(damageCapFixture(), b.PlayerID())
	g.AddPreventionShield(a.PlayerID(), 1)
	g.AddPreventionShield(b.PlayerID(), 1)
	spell := NewInstant("Source", "", nil)
	source := DamageSource{id: spell.ID(), stackID: uuid.New(), card: spell}
	g.DealSimultaneousDamage(PlayerDamageAssignment(b, 3, source), PlayerDamageAssignment(a, 3, source))
	if a.Life() != 19 || b.Life() != 19 || strings.Join(order, ",") != "A,B" {
		t.Fatalf("life %d/%d, choice order %v", a.Life(), b.Life(), order)
	}
}

func TestStaticDamageCapSeparateVisitsAndZero(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	capSource := g.PutOnBattlefield(damageCapFixture(), a.PlayerID())
	spell := NewInstant("Source", "", nil)
	first := DamageSource{id: spell.ID(), stackID: uuid.New(), card: spell}
	second := first
	second.stackID = uuid.New()
	g.DealSimultaneousDamage(PlayerDamageAssignment(a, 2, first), PlayerDamageAssignment(a, 2, second))
	if a.Life() != 16 {
		t.Fatal("separate source visits were combined")
	}
	g.RemoveFromBattlefield(g.FindPermanent(capSource.ID()))
	kinds := []core.CardType{core.TypeInstant}
	rule := LimitPlayerDamageFromTypes(You, 3, 0, kinds...)
	kinds[0] = core.TypeCreature
	card := NewArtifact("Zero damage", "", WithAbility(Static(rule)))
	if err := ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(card, a.PlayerID())
	g.DealDamageToPlayerFromSource(a, 3, first)
	if a.Life() != 16 {
		t.Fatal("zero replacement or declaration input isolation")
	}
	a.LoseLife(2)
	if a.Life() != 14 {
		t.Fatal("damage replacement changed independent life loss")
	}
}

func TestStaticDamageCapEventsAndSimultaneousDeparture(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	amount := BindNumber("damage")
	g.PutOnBattlefield(NewEnchantment("Watcher", "", WithAbility(WhenEvent(ControllerIsDealtDamage(amount), GainLifeFor(You, FromNumber(amount))))), a.PlayerID())
	capSource := g.PutOnBattlefield(NewCreature("Cap source", "", 1, 1, WithAbility(Static(LimitPlayerDamageFromTypes(You, 3, 2, core.TypeInstant)))), a.PlayerID())
	spell := NewInstant("Source", "", nil)
	source := DamageSource{id: spell.ID(), stackID: uuid.New(), card: spell}
	g.DealSimultaneousDamage(PermanentDamageAssignment(capSource, 1, source), PlayerDamageAssignment(a, 6, source))
	g.CheckStateBasedActions()
	g.PutTriggersOnStack()
	if a.Life() != 18 || g.FindPermanent(capSource.ID()) != nil || g.StackSize() != 1 {
		t.Fatal("simultaneous damage or source departure")
	}
	g.ResolveStack()
	if a.Life() != 20 {
		t.Fatal("damage event did not save the replaced amount")
	}
	g.DealDamageToPlayerFromSource(a, 4, source)
	if a.Life() != 16 {
		t.Fatal("departed source kept its damage rule")
	}
}

func TestStaticDamageCapAfterRedirection(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(damageCapFixture(), a.PlayerID())
	source := g.PutOnBattlefield(NewArtifact("Source", ""), b.PlayerID())
	victim := g.PutOnBattlefield(NewCreature("Victim", "", 1, 10), a.PlayerID())
	saved := g.CaptureDamageSource(source.ID())
	g.effects.AddReplacement(&selectedSourceRedirectReplacement{
		replacementBase: replacementBase{duration: core.EndOfTurn},
		source:          saved,
		subject:         capturedPermanent{victim.ID(), victim.incarnationID},
		player:          a.PlayerID(),
	})
	g.DealSimultaneousDamage(PlayerDamageAssignment(a, 1, saved), PermanentDamageAssignment(victim, 3, saved))
	if a.Life() != 18 || victim.Damage != 0 {
		t.Fatal("redirected packets did not cause a grouped threshold recheck")
	}
}

func TestStaticDamageCapReplacementOrder(t *testing.T) {
	for _, first := range []string{"Damage cap", "prevention"} {
		a := &damageCapOrderPlayer{BasePlayer: NewBasePlayer("A"), first: first}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		g.PutOnBattlefield(damageCapFixture(), a.PlayerID())
		g.AddPreventionShield(a.PlayerID(), 1)
		spell := NewInstant("Source", "", nil)
		g.DealDamageToPlayerFromSource(a, 3, DamageSource{id: spell.ID(), card: spell})
		want := 19
		if first == "prevention" {
			want = 18
		}
		if a.Life() != want || a.calls == 0 {
			t.Fatalf("order %s: life %d, choices %d", first, a.Life(), a.calls)
		}
	}
}

func TestStaticDamageCapDefinition(t *testing.T) {
	for _, rule := range []StaticRule{
		LimitPlayerDamageFromTypes(nil, 3, 2, core.TypeInstant),
		LimitPlayerDamageFromTypes(You, 0, 0, core.TypeInstant),
		LimitPlayerDamageFromTypes(You, 3, -1, core.TypeInstant),
		LimitPlayerDamageFromTypes(You, 3, 3, core.TypeInstant),
		LimitPlayerDamageFromTypes(You, 3, 2),
		LimitPlayerDamageFromTypes(You, 3, 2, core.CardType(-1)),
		LimitPlayerDamageFromTypes(You, 3, 2, core.TypeInstant, core.TypeInstant),
	} {
		card := NewArtifact("Invalid", "", WithAbility(Static(rule)))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("invalid rule accepted", rule)
		}
	}
}

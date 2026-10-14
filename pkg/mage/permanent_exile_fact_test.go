package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentExileFactPayment(t *testing.T) {
	for _, state := range []string{"printed", "gained", "lost", "token", "rollback", "phase"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			thrull, creature := BindNumber("paid Thrull"), BindNumber("paid creature")
			cost := ExileMatchingPermanentCostWithFacts(IsCreature, "Exile a creature", PermanentPaymentFact{And(IsCreature, HasSubType("Thrull")), thrull}, PermanentPaymentFact{IsCreature, creature})
			source := g.PutOnBattlefield(NewArtifact("Source", "", WithAbility(Activate(Costs(Tap(), cost, ManaCostOf("{1}")), Sequence(IfPositive(FromNumber(thrull), GainLifeFor(You, Fixed(2))), IfPositive(FromNumber(creature), GainLifeFor(You, Fixed(1))))))), a.PlayerID())
			card := NewCreature("Guest", "", 2, 2, WithSubTypes("Thrull"))
			if state == "gained" {
				card = NewCreature("Guest", "", 2, 2)
			}
			p := g.PutOnBattlefield(card, a.PlayerID())
			if state == "gained" {
				p.SubTypeOverride = []string{"Thrull"}
			}
			if state == "lost" {
				p.SubTypeOverride = []string{"Goblin"}
			}
			if state == "token" {
				p.IsToken = true
			}
			if state == "phase" {
				g.PhaseOut(p.ID())
			}
			if state != "rollback" {
				a.ManaPool().Add(core.Colorless, 1)
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			if state == "rollback" || state == "phase" {
				if err == nil || g.FindPermanentIncludingPhased(p.ID()) == nil || source.Tapped || g.StackSize() != 0 {
					t.Fatal("failed payment changed live state", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if a.Life() != 20 {
				t.Fatal("body resolved during payment")
			}
			clone := g.Clone()
			g.RemoveFromBattlefield(source)
			g.ResolveStack()
			clone.ResolveStack()
			want := 23
			if state == "lost" {
				want = 21
			}
			if a.Life() != want || clone.GetPlayer(a.PlayerID()).Life() != want {
				t.Fatal("payment-time characteristics were lost", a.Life(), want)
			}
		})
	}
}

func TestPermanentExileFactDefinition(t *testing.T) {
	for _, fact := range []PermanentPaymentFact{{IsCreature, nil}, {PermanentFilter{}, BindNumber("bad")}} {
		card := NewInstant("Bad", "", WithAdditionalCost(ExileMatchingPermanentCostWithFacts(IsCreature, "Exile", fact)), Spell(GainLifeFor(You, Fixed(1))))
		if err := ValidateCardDefinition("Bad", card); err == nil {
			t.Fatal("invalid fact accepted")
		}
	}
	result := BindNumber("missing producer")
	if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", Spell(IfPositive(FromNumber(result), GainLifeFor(You, Fixed(1)))))); err == nil {
		t.Fatal("unbound fact accepted")
	}
	duplicate := ExileMatchingPermanentCostWithFacts(IsCreature, "Exile", PermanentPaymentFact{IsCreature, result}, PermanentPaymentFact{IsLand, result})
	if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", WithAdditionalCost(duplicate), Spell(GainLifeFor(You, Fixed(1))))); err == nil {
		t.Fatal("duplicate output accepted")
	}
	if err := ValidateCardDefinition("Bad", NewInstant("Bad", "", Spell(PayOrElse(You, Costs(ExileMatchingPermanentCostWithFacts(IsCreature, "Exile", PermanentPaymentFact{IsCreature, result})), GainLifeFor(You, Fixed(1)), GainLifeFor(You, Fixed(2)))))); err == nil {
		t.Fatal("unsupported resolving payment accepted")
	}
}

func TestPermanentExileFactStaleAndCommitTime(t *testing.T) {
	for _, state := range []string{"return", "phase", "cycle", "control", "changed"} {
		t.Run(state, func(t *testing.T) {
			a := &matchingExileChooser{BasePlayer: NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			source := g.PutOnBattlefield(NewArtifact("Source", ""), a.PlayerID())
			card := NewCreature("Guest", "", 2, 2, WithSubTypes("Thrull"))
			p := g.PutOnBattlefield(card, a.PlayerID())
			result := BindNumber("paid Thrull")
			cost := ExileMatchingPermanentCostWithFacts(IsCreature, "Exile", PermanentPaymentFact{HasSubType("Thrull"), result})
			for range 3 {
				if !cost.CanPay(source.ID(), a.PlayerID(), g) {
					t.Fatal("pure payment query failed")
				}
			}
			if a.choices != 0 {
				t.Fatal("query made a choice")
			}
			a.ManaPool().Add(core.Colorless, 1)
			tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{Controller: a.PlayerID(), SourceID: source.ID(), Mana: core.ParseManaCost("{1}"), Costs: []Cost{Tap(), cost}}, nil)
			if err != nil {
				t.Fatal(err)
			}
			switch state {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(card, a.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				p.computedController, p.baseController = b.PlayerID(), b.PlayerID()
			case "changed":
				p.SubTypeOverride = []string{"Goblin"}
			}
			err = tx.Commit()
			if state == "changed" || state == "cycle" {
				want := 0
				if state == "cycle" {
					want = 1
				}
				if err != nil || len(tx.paidNumbers) != 1 || tx.paidNumbers[result] != want || a.choices != 1 {
					t.Fatal("facts did not use commit-time characteristics", err, tx.paidNumbers)
				}
			} else if err == nil || len(tx.paidNumbers) != 0 || source.Tapped || a.ManaPool().TotalMana() != 1 || len(g.GetExile()) != 0 {
				t.Fatal("stale payment published facts or spent resources", err)
			}
		})
	}
}

package mage

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage/core"
)

type receiptChoicePlayer struct{ *BasePlayer }

func (*receiptChoicePlayer) ChooseMayAbility(string) bool { return true }

func TestActivationManaReceipt(t *testing.T) {
	for _, cost := range []string{"{U}{U}", "{1}{U}", "{2}", "{W/U}{W/U}", "{2}{U}", "{G}", "{X}{U}", "{0}", ""} {
		t.Run(cost, func(t *testing.T) {
			g, a, _ := newSpellCopyGame()
			g.SetPlayerAt(0, &receiptChoicePlayer{a.BasePlayer})
			binding := BindActivationMana("payment")
			result := BindCard("choice")
			body := ChooseHandCardWithActivationManaOrElse(You, CardFilter{}, binding, result, GainLifeFor(You, Fixed(1)), nil)
			action := NewActivated(ManaCostOf("{2}"), body, WithActivationManaReceipt(binding))
			card := NewArtifact("Receipt", "", WithAbility(action))
			if err := ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			a.AddToHand(NewCreature("Candidate", cost, 1, 1))
			a.ManaPool().Add(core.Blue, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			a.ManaPool().Add(core.Green, 9)
			a.ManaPool().LastDrainedColors = map[core.Color]int{core.Green: 99}
			clone := g.Clone()
			clone.SetPlayerAt(0, &receiptChoicePlayer{clone.GetPlayer(a.PlayerID()).(*SearchPlayer).BasePlayer})
			g.ResolveStack()
			clone.ResolveStack()
			want := 20
			if cost != "{2}{U}" && cost != "{G}" && cost != "" {
				want = 21
			}
			for _, game := range []*Game{g, clone} {
				if got := game.GetPlayer(a.PlayerID()).Life(); got != want {
					t.Fatalf("life=%d want=%d", got, want)
				}
			}
		})
	}
}

func TestActivationManaReceiptDefinition(t *testing.T) {
	b := BindActivationMana("receipt")
	broken := &ActivationManaBinding{name: "broken"}
	choose := func(receipt *ActivationManaBinding) ResolvingEffect {
		return ChooseHandCardWithActivationManaOrElse(You, CardFilter{}, receipt, BindCard("choice"), GainLifeFor(You, Fixed(1)), nil)
	}
	for name, card := range map[string]Card{
		"undeclared":    NewArtifact("Bad", "", WithAbility(NewActivated(GenericCost(1), choose(b)))),
		"nil":           NewArtifact("Bad", "", WithAbility(NewActivated(GenericCost(1), choose(nil), WithActivationManaReceipt(nil)))),
		"duplicate":     NewArtifact("Bad", "", WithAbility(NewActivated(GenericCost(1), choose(b), WithActivationManaReceipt(b), WithActivationManaReceipt(b)))),
		"spell":         NewInstant("Bad", "", WithAbility(NewAction(ActionSpell, WithEffect(choose(b)), WithActivationManaReceipt(b)))),
		"two owners":    NewArtifact("Bad", "", WithAbility(NewActivated(GenericCost(1), choose(b), WithActivationManaReceipt(b))), WithAbility(NewActivated(GenericCost(1), choose(b), WithActivationManaReceipt(b)))),
		"empty":         NewArtifact("Bad", "", WithAbility(NewActivated(GenericCost(1), choose(BindActivationMana("")), WithActivationManaReceipt(BindActivationMana(""))))),
		"uninitialized": NewArtifact("Bad", "", WithAbility(NewActivated(GenericCost(1), choose(broken), WithActivationManaReceipt(broken)))),
	} {
		t.Run(name, func(t *testing.T) {
			if err := ValidateCardDefinition(card.Name(), card); err == nil {
				t.Fatal("accepted invalid receipt")
			}
		})
	}
}

func TestActivationManaReceiptIndependentPayments(t *testing.T) {
	a := &optionalHandChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	receipt := BindActivationMana("payment")
	action := NewActivated(GenericCost(1), ChooseHandCardWithActivationManaOrElse(You, CardFilter{}, receipt, BindCard("chosen"), GainLifeFor(You, Fixed(1)), nil), WithActivationManaReceipt(receipt))
	source := g.PutOnBattlefield(NewArtifact("Receipt", "", WithAbility(action)), a.PlayerID())
	a.AddToHand(NewCreature("Blue", "{U}", 1, 1))
	for _, color := range []core.Color{core.Blue, core.Green} {
		a.ManaPool().Add(color, 1)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
	}
	if cp := g.CopyStackObjectDirect(g.StackObjects()[0], b.PlayerID(), false); cp == nil {
		t.Fatal("copy failed")
	}
	b.AddToHand(NewCreature("Blue copy choice", "{U}", 1, 1))
	clone := g.Clone()
	clone.SetPlayerAt(0, &optionalHandChoicePlayer{BasePlayer: clone.GetPlayer(a.PlayerID()).(*SearchPlayer).BasePlayer, accept: true})
	clone.SetPlayerAt(1, &optionalHandChoicePlayer{BasePlayer: clone.GetPlayer(b.PlayerID()).(*SearchPlayer).BasePlayer, accept: true})
	g.SetPlayerAt(1, &optionalHandChoicePlayer{BasePlayer: b, accept: true})
	g.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 20 {
		t.Fatal("live resolution changed clone")
	}
	clone.ResolveStack()
	for _, game := range []*Game{g, clone} {
		if game.GetPlayer(a.PlayerID()).Life() != 21 || game.GetPlayer(b.PlayerID()).Life() != 21 {
			t.Fatal("payment receipts collided or copy lost original payment")
		}
	}
}

func TestActivationManaReceiptRollbackAndZero(t *testing.T) {
	for _, scenario := range []string{"unaffordable", "life", "zero", "auto-mana", "conversion", "conversion-actual"} {
		t.Run(scenario, func(t *testing.T) {
			a := &optionalHandChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			g := NewGame(a, NewBasePlayer("B"))
			receipt := BindActivationMana("paid")
			cost := GenericCost(1)
			candidate := "{U}"
			if scenario == "zero" {
				cost = GenericCost(0)
				candidate = "{0}"
			}
			action := NewActivated(cost, ChooseHandCardWithActivationManaOrElse(You, CardFilter{}, receipt, BindCard("chosen"), GainLifeFor(You, Fixed(1)), nil), WithActivationManaReceipt(receipt))
			if scenario == "life" {
				WithCost(LifePayCost(100))(action)
				a.ManaPool().Add(core.Blue, 1)
			}
			if scenario == "auto-mana" {
				g.PutOnBattlefield(NewLand("Producer", WithAbility(NewManaAbility(core.Blue))), a.PlayerID())
			}
			if scenario == "conversion" || scenario == "conversion-actual" {
				action.costs = []Cost{ManaCostOf("{R}")}
				g.PutOnBattlefield(NewArtifact("Permission", "", WithStaticAbility(ControllerManaSpending(core.Blue, core.Red))), a.PlayerID())
				a.ManaPool().Add(core.Blue, 1)
				candidate = "{R}"
				if scenario == "conversion-actual" {
					candidate = "{U}"
				}
			}
			source := g.PutOnBattlefield(NewArtifact("Receipt", "", WithAbility(action)), a.PlayerID())
			a.AddToHand(NewCreature("Candidate", candidate, 1, 1))
			before := a.ManaPool().TotalMana()
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil)
			if scenario == "unaffordable" || scenario == "life" {
				if err == nil || g.StackSize() != 0 || a.ManaPool().TotalMana() != before || a.offers != 0 || a.Life() != 20 {
					t.Fatal("failed payment changed state")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if a.offers != 0 {
				t.Fatal("selection occurred during payment")
			}
			g.ResolveStack()
			want := 21
			if scenario == "conversion" {
				want = 20
			}
			if a.Life() != want {
				t.Fatalf("life=%d want=%d", a.Life(), want)
			}
		})
	}
}

func TestActivationManaReceiptXComponent(t *testing.T) {
	for _, scenario := range []string{"colored-tax", "generic-tax", "reduced", "zero-tax", "zero"} {
		t.Run(scenario, func(t *testing.T) {
			a := &optionalHandChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: true}
			g := NewGame(a, NewBasePlayer("B"))
			receipt := BindActivationMana("X")
			action := NewActivated(ManaCostOf("{X}"), ChooseHandCardWithActivationManaOrElse(You, CardFilter{}, receipt, BindCard("chosen"), GainLifeFor(You, Fixed(1)), nil), WithActivationManaXReceipt(receipt))
			options := []CardOption{WithAbility(action)}
			if scenario == "reduced" {
				options = append(options, WithStaticAbility(ReduceActivationCostStatic(This, 1, 0)))
			}
			source := g.PutOnBattlefield(NewArtifact("Receipt", "", options...), a.PlayerID())
			x := 2
			want := 21
			switch scenario {
			case "colored-tax":
				g.PutOnBattlefield(NewArtifact("Tax", "", WithStaticAbility(IncreaseActivationCostStatic(IsArtifact, core.ManaCost{Green: 1}))), a.PlayerID())
				a.ManaPool().Add(core.Green, 1)
			case "generic-tax":
				g.PutOnBattlefield(NewArtifact("Tax", "", WithStaticAbility(IncreaseActivationCostStatic(IsArtifact, core.ManaCost{Generic: 1}))), a.PlayerID())
				a.ManaPool().Add(core.Colorless, 1)
			case "zero-tax":
				x = 0
				want = 20
				g.PutOnBattlefield(NewArtifact("Tax", "", WithStaticAbility(IncreaseActivationCostStatic(IsArtifact, core.ManaCost{Blue: 1}))), a.PlayerID())
			case "zero":
				x = 0
				want = 20
			case "reduced":
				want = 20
			}
			a.ManaPool().Add(core.Blue, 2)
			a.AddToHand(NewCreature("Candidate", "{U}{U}", 1, 1))
			g.SetXValue(x)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			paid := g.StackPeek().capturedNumbers
			if paid[receipt.amounts[core.Green]] != 0 || paid[receipt.amounts[core.Colorless]] != 0 {
				t.Fatal("tax mana entered X receipt")
			}
			g.ResolveStack()
			if a.Life() != want {
				t.Fatalf("life=%d want=%d", a.Life(), want)
			}
		})
	}
}

func TestActivationManaReceiptSelectionRecheck(t *testing.T) {
	a := &optionalHandChoicePlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	g := NewGame(a, NewBasePlayer("B"))
	receipt := BindActivationMana("paid")
	candidate := NewCreature("Candidate", "{U}", 1, 1)
	a.AddToHand(candidate)
	a.choose = func(cards []Card) Card { candidate.manaCost = core.ParseManaCost("{G}"); return cards[0] }
	source := g.PutOnBattlefield(NewArtifact("Receipt", "", WithAbility(NewActivated(GenericCost(1), ChooseHandCardWithActivationManaOrElse(You, CardFilter{}, receipt, BindCard("chosen"), GainLifeFor(You, Fixed(1)), nil), WithActivationManaReceipt(receipt)))), a.PlayerID())
	a.ManaPool().Add(core.Blue, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.choices != 1 || a.Life() != 20 {
		t.Fatal("selection did not recheck the canonical card against the receipt")
	}
}

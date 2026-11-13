package eval

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRemovalHandValueValuesFlexibilityAndScarcity(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	swords := mage.NewInstant("Flexible answer", "{W}", mage.NewTargetedSpell(mage.TargetCreature(), mage.SwordsToPlowsharesEffect()))
	bolt := mage.NewInstant("Three damage", "{R}", mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.DealDamage(mage.Fixed(3))))
	g.SetTurn(4)
	a.SetHand([]*mage.Card{swords})
	single := RemovalHandValue(g, a)
	a.SetHand([]*mage.Card{bolt})
	limited := RemovalHandValue(g, a)
	if single <= limited || limited <= 0 {
		t.Fatalf("expected flexible answer above damage: flexible %v, damage %v", single, limited)
	}
	a.SetHand([]*mage.Card{swords, swords.NewCopy()})
	two := RemovalHandValue(g, a)
	if two <= single || two-single >= single {
		t.Fatalf("expected diminishing value: one %v, two %v", single, two)
	}
	a.SetHand([]*mage.Card{swords})
	g.SetTurn(5)
	if late := RemovalHandValue(g, a); late >= single {
		t.Fatalf("expected early flexible answer to be worth more: early %v, late %v", single, late)
	}
}

func TestRemovalCardValueUsesCompleteManaProduction(t *testing.T) {
	valueWithSource := func(options ...mage.CardOption) float64 {
		a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.SetTurn(4)
		source := mage.NewPermanent(mage.NewLand("Mana source", options...), a.PlayerID())
		source.ChosenColor = core.Green
		g.AddToBattlefield(source)
		answer := mage.NewInstant("Expensive answer", "{4}", mage.NewTargetedSpell(mage.TargetCreature(), mage.DestroyTarget()))
		return RemovalCardValue(g, a, answer)
	}
	for _, tc := range []struct {
		name    string
		amount  int
		options []mage.CardOption
	}{
		{
			name:   "mixed colors",
			amount: 2,
			options: []mage.CardOption{mage.WithMultiManaAbility(
				mage.ManaProduction{Color: core.Green, Amount: 1},
				mage.ManaProduction{Color: core.White, Amount: 1},
			)},
		},
		{
			name:    "chosen color",
			amount:  1,
			options: []mage.CardOption{mage.WithDynamicManaAbility(mage.ChosenColorManaProductions)},
		},
		{
			name:    "alternative abilities",
			amount:  1,
			options: []mage.CardOption{mage.WithManaAbility(core.Green), mage.WithManaAbility(core.White)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := valueWithSource(mage.WithMultiManaAbility(mage.ManaProduction{Color: core.Colorless, Amount: tc.amount}))
			if got := valueWithSource(tc.options...); got != want {
				t.Fatalf("equivalent mana sources must give the same answer value: got %v, want %v", got, want)
			}
		})
	}
}

func TestRemovalHandValueExcludesDamagePrevention(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	source := mage.NewLand("Mana source", mage.WithMultiManaAbility(mage.ManaProduction{Color: core.White, Amount: 2}))
	g.AddToBattlefield(mage.NewPermanent(source, a.PlayerID()))
	prevention := mage.NewInstant("Prevention", "{X}{W}", mage.NewTargetedSpell(mage.TargetDamageAnyTarget(), mage.PreventDamageToTarget(mage.XValue())))
	a.SetHand([]*mage.Card{prevention})
	if got := RemovalHandValue(g, a); got != 0 {
		t.Fatalf("damage prevention must not count as removal in hand, got %v", got)
	}
}

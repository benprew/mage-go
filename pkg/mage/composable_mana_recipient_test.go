package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestAddManaToPlayerUsesSavedRecipient(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	player := BindPlayer("producer")
	watcher := NewEnchantment("Watcher", "", WithAbility(WhenEvent(
		LandTappedForMana().WithPlayer(player), AddManaToPlayer(FromPlayer(player), core.Green, Fixed(2)))))
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	land := NewLand("Land", WithAbility(Activate(Costs(TapCost(This)), AddManaAmount(core.Blue, Fixed(1)))))
	g.PutOnBattlefield(land, b.PlayerID())
	if err := g.ActivateAbilityByIndex(b.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if got := b.ManaPool().Count(core.Green); got != 2 {
		t.Fatalf("immediate recipient green mana = %d, want 2", got)
	}
	g.RemoveFromBattlefield(g.FindPermanent(land.ID()))
	g.PutOnBattlefield(land, a.PlayerID())
	g.PutTriggersOnStack()
	clone := g.Clone()
	clone.ResolveStack()
	if got := clone.GetPlayer(b.PlayerID()).ManaPool().Count(core.Green); got != 2 {
		t.Fatalf("clone recipient green mana = %d, want 2", got)
	}
	g.ResolveStack()
	if got := b.ManaPool().Count(core.Green); got != 2 {
		t.Fatalf("recipient green mana = %d, want 2", got)
	}
	if got := a.ManaPool().Count(core.Green); got != 0 {
		t.Fatalf("controller green mana = %d, want 0", got)
	}
}

func TestAddManaToPlayerAbsenceAndValidation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if err := ApplyEffect(g, AddManaToPlayer(FromPlayer(BindPlayer("missing")), core.Green, Fixed(2)),
		uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if got := a.ManaPool().Count(core.Green); got != 0 {
		t.Fatalf("absent recipient added %d mana", got)
	}
	for _, effect := range []ResolvingEffect{
		AddManaToPlayer(nil, core.Green, Fixed(1)),
		AddManaToPlayer(You, core.AnyColor, Fixed(1)),
		AddManaToPlayer(You, core.Green, nil),
	} {
		card := NewSorcery("Bad", "", Spell(effect))
		if err := ValidateCardDefinition(card.Name(), card); err == nil {
			t.Fatal("accepted invalid mana recipient action")
		}
	}
	card := NewSorcery("Unbound", "", Spell(AddManaToPlayer(FromPlayer(BindPlayer("missing")), core.Green, Fixed(1))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil {
		t.Fatal("accepted unbound player")
	}
	if got := b.ManaPool().Count(core.Green); got != 0 {
		t.Fatalf("other player received %d mana", got)
	}
}

func TestAddManaToPlayerActivatedManaClassification(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := NewLand("Shared", WithAbility(Activate(Costs(TapCost(This)),
		AddManaToPlayer(ActivePlayer, core.Red, Fixed(1)))))
	g.PutOnBattlefield(land, a.PlayerID())
	if !IsActivatedManaAbility(land.Abilities()[0]) {
		t.Fatal("targetless player mana action was not a mana ability")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if g.StackSize() != 0 || a.ManaPool().Count(core.Red) != 1 || b.ManaPool().Count(core.Red) != 0 {
		t.Fatal("mana activation did not resolve immediately for its recipient")
	}
}

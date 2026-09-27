package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestOptionalGraveyardTargetTriggerWithoutCard(t *testing.T) {
	g, a, _ := newTriggerTargetGame()
	target := DeclareGraveyardCardTarget("card", GraveyardUpToOne(), GraveyardCardFilter(IsCreatureCard))
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(Enters(This),
		Sequence(GainLifeFor(You, Fixed(1)), ReturnCardToBattlefield(target, You)), WithTargets(target))))
	watcher.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(watcher.Name(), watcher); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutTriggersOnStack()
	if obj := g.StackPeek(); obj == nil || len(obj.Targets) != 0 {
		t.Fatalf("zero-target trigger = %v", obj)
	}
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatalf("zero-target trigger life = %d", a.Life())
	}
}

func TestOptionalGraveyardTargetMayChooseZeroWithLegalCard(t *testing.T) {
	g, a, _ := newTriggerTargetGame()
	card := NewCreature("Creature", "", 1, 1)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	target := DeclareGraveyardCardTarget("card", GraveyardUpToOne())
	watcher := NewCreature("Watcher", "", 1, 1, WithAbility(WhenEvent(Enters(This),
		GainLifeFor(You, Fixed(1)), WithTargets(target))))
	watcher.SetOwner(a.PlayerID())
	a.chooseQueue = [][]uuid.UUID{{}}
	g.PutOnBattlefield(watcher, a.PlayerID())
	g.PutTriggersOnStack()
	if obj := g.StackPeek(); obj == nil || len(obj.Targets) != 0 {
		t.Fatalf("voluntary zero-target trigger = %v", obj)
	}
	copyObj := g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false)
	if copyObj == nil || len(copyObj.Targets) != 0 {
		t.Fatalf("zero-target copy = %v", copyObj)
	}
	g.ResolveStack()
	if a.Life() != 22 {
		t.Fatalf("copied zero-target trigger life = %d", a.Life())
	}
}

func TestOptionalGraveyardTargetSelectedCopyAndRecheck(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewCreature("Creature", "{1}", 1, 1)
	card.SetOwner(a.PlayerID())
	a.AddToGraveyard(card)
	target := DeclareGraveyardCardTarget("card", GraveyardUpToOne(), GraveyardCardManaValueAtMost(FixedTargetNumber(1)))
	ability := Spell(ReturnCardToHand(target), WithTargets(target))
	spell := NewSorcery("Recover", "", WithAbility(ability))
	spell.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition(spell.Name(), spell); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: spell.ID(), Card: spell, Controller: a.PlayerID(), Targets: []uuid.UUID{card.ID()}, TargetSpecs: ability.Targets(), Effects: ability.Effects()}
	g.pushStack(obj)
	copyObj := g.CopyStackObjectDirect(obj, a.PlayerID(), false)
	if copyObj == nil || len(copyObj.Targets) != 1 || copyObj.Targets[0] != card.ID() {
		t.Fatalf("selected target copy = %v", copyObj)
	}
	clone := g.Clone()
	clone.ResolveTopOfStack()
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(a.Hand()) != 0 {
		t.Fatal("clone changed original target")
	}
	moved, ok := g.MoveFromGraveyard(a.PlayerID(), card.ID(), core.ZoneHand)
	if !ok {
		t.Fatal("could not move target")
	}
	a.AddToHand(moved)
	returned, _ := a.RemoveFromHand(card.ID())
	a.AddToGraveyard(returned)
	g.ResolveTopOfStack()
	if len(a.Hand()) != 0 {
		t.Fatal("old target followed a new graveyard visit")
	}
}

func TestOptionalGraveyardTargetRejectsDuplicateOption(t *testing.T) {
	target := DeclareGraveyardCardTarget("card", GraveyardUpToOne(), GraveyardUpToOne())
	card := NewSorcery("Invalid", "", WithAbility(Spell(ReturnCardToHand(target), WithTargets(target))))
	if err := ValidateCardDefinition(card.Name(), card); err == nil || !strings.Contains(err.Error(), "up to one") {
		t.Fatalf("duplicate option validation = %v", err)
	}
}

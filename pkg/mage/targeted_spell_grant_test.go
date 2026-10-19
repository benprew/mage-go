package mage

import (
	"testing"

	"github.com/google/uuid"
)

func TestTargetedSpellGrantInstances(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	target := DeclareSpellTarget("spell")
	donor := NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(MatchingPermanents(IsLand), Costs(TapCost(This)), Sequence(Counter(target), GainLifeFor(You, Fixed(2))), WithTargets(target)))))
	if err := ValidateCardDefinition(donor.Name(), donor); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(donor, a.PlayerID())
	first := g.PutOnBattlefield(NewLand("First"), a.PlayerID())
	second := g.PutOnBattlefield(NewLand("Second"), b.PlayerID())
	g.ApplyContinuousEffects()
	objects := make([]*StackObject, 2)
	for i := range objects {
		card := NewArtifact("Victim", "")
		card.SetOwner(b.PlayerID())
		objects[i] = &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: b.PlayerID()}
		g.pushStack(objects[i])
	}
	for _, entry := range []struct{ player, source, target uuid.UUID }{{a.PlayerID(), first.ID(), objects[0].ID}, {b.PlayerID(), second.ID(), objects[1].ID}} {
		if err := g.ActivateAbilityByIndex(entry.player, entry.source, 0, []uuid.UUID{entry.target}); err != nil {
			t.Fatal(err)
		}
	}
	clone := g.Clone()
	clone.CounterStackObject(objects[1].ID)
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 20 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("spell target or clone state escaped its activation")
	}
	g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
	g.ResolveStack()
	if a.Life() != 22 || b.Life() != 22 || len(b.Graveyard()) != 2 {
		t.Fatal("independent granted spell targets did not resolve")
	}
}

func TestTargetedSpellGrantRetargetCopy(t *testing.T) {
	g, a, b := newSpellCopyGame()
	target := DeclareSpellTarget("spell")
	donor := g.PutOnBattlefield(NewEnchantment("Donor", "", WithAbility(Static(GrantEachActivatedAbility(MatchingPermanents(IsLand), Costs(), Sequence(Counter(target), GainLifeFor(You, Fixed(2))), WithTargets(target))))), a.PlayerID())
	recipient := g.PutOnBattlefield(NewLand("Recipient"), a.PlayerID())
	g.ApplyContinuousEffects()
	objects := make([]*StackObject, 2)
	for i := range objects {
		card := NewArtifact("Victim", "")
		card.SetOwner(b.PlayerID())
		objects[i] = &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: b.PlayerID()}
		g.pushStack(objects[i])
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), recipient.ID(), 0, []uuid.UUID{objects[0].ID}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	a.chooseQueue = [][]uuid.UUID{{objects[1].ID}}
	cp := g.CopyStackObjectDirect(original, a.PlayerID(), true)
	if cp == nil || cp.Targets[0] != objects[1].ID || original.Targets[0] != objects[0].ID {
		t.Fatal("retargeting changed the original activation")
	}
	g.RemoveFromBattlefield(g.FindPermanent(donor.ID()))
	g.ResolveStack()
	if a.Life() != 24 || len(b.Graveyard()) != 2 {
		t.Fatal("retargeted spell grant copy lost its own binding")
	}
}

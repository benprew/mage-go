package ron

import (
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextAbilityCorrespondence(t *testing.T) {
	change, _ := mage.ReplaceColorWord(core.White, core.Red)
	original := repositoryCard(t, "LEA", "Black Knight")
	revision, err := mage.BuildTextDefinitionRevision(original, change)
	if err != nil {
		t.Fatal(err)
	}
	changed := revision.Definition()
	if changed.ID() == original.ID() {
		t.Fatal("revision reused physical identity")
	}
	for _, ability := range original.Abilities() {
		replacement, ok := revision.ReplacementAbility(ability.AbilityID())
		if !ok || replacement.AbilityID() == ability.AbilityID() {
			t.Fatal("missing independent replacement")
		}
	}
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(changed, a.PlayerID())
	if !p.HasProtectionFrom(textColorCard(core.Red)) || p.HasProtectionFrom(textColorCard(core.White)) {
		t.Fatal("changed rules did not execute")
	}
	if _, ok := revision.ReplacementAbility(repositoryCard(t, "LEA", "Black Knight").Abilities()[0].AbilityID()); ok {
		t.Fatal("other instance inherited correspondence")
	}
	next, _ := mage.ReplaceColorWord(core.Red, core.Green)
	second, err := mage.BuildTextDefinitionRevision(changed, next)
	if err != nil {
		t.Fatal(err)
	}
	for _, ability := range changed.Abilities() {
		if _, ok := second.ReplacementAbility(ability.AbilityID()); !ok {
			t.Fatal("successive revision lost declaration")
		}
	}
	if _, err := mage.BuildTextDefinitionRevision(original.Copy(), change); err != nil {
		t.Fatal(err)
	}
}

func TestRONTextAbilityCorrespondenceQueuedActivation(t *testing.T) {
	original := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsColor(Red))],ActionList([DestroyPermanent(Ref_TargetPermanent)])))`)()
	g, a, b := ronGame()
	red := g.PutOnBattlefield(textColorCard(core.Red), b.PlayerID())
	green := g.PutOnBattlefield(textColorCard(core.Green), b.PlayerID())
	source := g.PutOnBattlefield(original, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{red.ID()}); err != nil {
		t.Fatal(err)
	}
	change, _ := mage.ReplaceColorWord(core.Red, core.Green)
	revision, err := mage.BuildTextDefinitionRevision(original, change)
	if err != nil {
		t.Fatal(err)
	}
	old := original.Abilities()[0].(*mage.ActionDefinition)
	replacement, ok := revision.ReplacementAbility(old.AbilityID())
	if !ok {
		t.Fatal("activation correspondence missing")
	}
	fresh := replacement.(*mage.ActionDefinition)
	if old.Targets()[0] == fresh.Targets()[0] || fresh.Targets()[0].IsChosen() {
		t.Fatal("target choices transferred")
	}
	if fresh.Source() != uuid.Nil || fresh.Controller() != uuid.Nil {
		t.Fatal("live context transferred")
	}
	changed := g.PutOnBattlefield(revision.Definition(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), changed.ID(), 0, []uuid.UUID{red.ID()}); err == nil {
		t.Fatal("old target filter remained")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), changed.ID(), 0, []uuid.UUID{green.ID()}); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	g.ResolveStack()
	if g.FindPermanent(red.ID()) != nil || g.FindPermanent(green.ID()) != nil {
		t.Fatal("independent pending bodies lost their filters")
	}
	if clone.FindPermanent(red.ID()) == nil || clone.FindPermanent(green.ID()) == nil {
		t.Fatal("clone shared resolution")
	}
	clone.ResolveStack()
	if clone.FindPermanent(red.ID()) != nil || clone.FindPermanent(green.ID()) != nil {
		t.Fatal("clone lost pending definitions")
	}
}

func TestRONTextAbilityCorrespondenceLandAndTrigger(t *testing.T) {
	change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	original := repositoryCard(t, "LEA", "Lifetap")
	revision, err := mage.BuildTextDefinitionRevision(original, change)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := revision.ReplacementAbility(original.Abilities()[0].AbilityID()); !ok {
		t.Fatal("trigger correspondence missing")
	}
	g, a, b := ronGame()
	g.PutOnBattlefield(revision.Definition(), a.PlayerID())
	forest := g.PutOnBattlefield(repositoryCard(t, "LEA", "Forest"), b.PlayerID())
	island := g.PutOnBattlefield(repositoryCard(t, "LEA", "Island"), b.PlayerID())
	g.TapPermanent(forest)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("original land word triggered")
	}
	g.TapPermanent(island)
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 21 {
		t.Fatal("rebuilt trigger did not execute")
	}
	revision, err = mage.BuildTextDefinitionRevision(repositoryCard(t, "LEA", "Tropical Island"), change)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(revision.Definition().SubTypes(), []string{"Island"}) {
		t.Fatal("ability-free type transformation failed")
	}
	for _, name := range []string{"Magical Hack", "Sleight of Mind"} {
		for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
			if _, err := Compile(staticBaseRecord(t, set, name)); err != nil {
				t.Fatal("complete text consumer failed", set, name)
			}
		}
	}
}

func TestRONTextAbilityCorrespondenceCountDefinition(t *testing.T) {
	card := repositoryCard(t, "LEA", "Keldon Warlord")
	revision, err := mage.BuildTextDefinitionRevision(card)
	if err != nil {
		t.Fatal(err)
	}
	for _, ability := range card.Abilities() {
		if _, ok := revision.ReplacementAbility(ability.AbilityID()); !ok {
			t.Fatal("combined CDA correspondence missing")
		}
	}
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(revision.Definition(), a.PlayerID())
	g.PutOnBattlefield(repositoryCard(t, "LEA", "Black Knight"), a.PlayerID())
	g.ApplyContinuousEffects()
	if p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 2 {
		t.Fatal("combined CDA did not execute")
	}
}

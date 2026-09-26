package mage

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDealDamageEachMixedRecipientsObserveOneBatch(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Recipient", "", 4, 4), b.PlayerID())
	g.AddPreventionShield(b.PlayerID(), 1)
	observed := 0
	g.damage.SetOnDamageDealt(func(_, _ string, _ int, _ bool) {
		observed++
		if a.Life() != 18 || b.Life() != 19 || creature.Damage != 2 {
			t.Errorf("callback saw partial batch: A=%d B=%d creature=%d", a.Life(), b.Life(), creature.Damage)
		}
	})
	effect := DealDamageEach(This, DamageRecipients(EachPlayer, WithoutPermanent(MatchingPermanents(IsCreature), This)), Fixed(2))
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if observed != 3 {
		t.Fatalf("damage callbacks = %d, want 3", observed)
	}
}

func TestDealDamageEachFiltersAndCloneIsolation(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1, WithKeyword(Lifelink)), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Recipient", "", 4, 4), b.PlayerID())
	g.PhaseOut(creature.ID())
	clone := g.Clone()
	effect := DealDamageEach(This, DamageRecipients(EachPlayer, WithoutPermanent(MatchingPermanents(IsCreature), This)), Fixed(2))
	if err := ApplyEffect(clone, effect, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 20 || b.Life() != 20 || creature.Damage != 0 {
		t.Fatal("batch changed original game")
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 18 {
		t.Fatal("clone did not apply two damage assignments with captured lifelink")
	}
}

func TestDealDamageEachDeclarationValidation(t *testing.T) {
	valid := NewSorcery("Valid", "", WithAbility(Spell(DealDamageEach(This,
		DamageRecipients(EachPlayer, MatchingPermanents(IsCreature)), Fixed(2)))))
	if err := ValidateCardDefinition("Valid", valid); err != nil {
		t.Fatal(err)
	}
	invalid := NewSorcery("Invalid", "", WithAbility(Spell(DealDamageEach(This,
		DamageRecipients(nil, MatchingPermanents(IsCreature)), Fixed(2)))))
	if err := ValidateCardDefinition("Invalid", invalid); err == nil || !strings.Contains(err.Error(), "player") {
		t.Fatalf("invalid recipient declaration: %v", err)
	}
	targets := DeclarePermanentTargets("creature", 1, 1, IsCreature)
	missing := NewSorcery("Missing", "", WithAbility(Spell(DealDamageEach(This,
		DamageRecipients(EachPlayer, targets), Fixed(2)))))
	if err := ValidateCardDefinition("Missing", missing); err == nil || !strings.Contains(err.Error(), "target binding requires") {
		t.Fatalf("undeclared target set: %v", err)
	}
	declared := NewSorcery("Declared", "", WithAbility(Spell(DealDamageEach(This,
		DamageRecipients(EachPlayer, targets), Fixed(2)), WithTargets(targets))))
	if err := ValidateCardDefinition("Declared", declared); err != nil {
		t.Fatal(err)
	}
	multiple := DeclarePermanentTargets("multiple", 1, 2, IsCreature)
	supported := NewSorcery("Supported", "", WithAbility(Spell(DealDamageEach(This,
		DamageRecipients(EachPlayer, multiple), Fixed(2)), WithTargets(multiple))))
	if err := ValidateCardDefinition("Supported", supported); err != nil {
		t.Fatalf("one variable target group: %v", err)
	}
	other := DeclarePermanentTargets("other", 0, 2, IsCreature)
	ambiguous := NewSorcery("Ambiguous", "", WithAbility(Spell(DealDamageEach(This,
		DamageRecipients(EachPlayer, multiple), Fixed(2)), WithTargets(multiple, other))))
	if err := ValidateCardDefinition("Ambiguous", ambiguous); err == nil || !strings.Contains(err.Error(), "recorded selection boundaries") {
		t.Fatalf("two variable target groups must remain rejected: %v", err)
	}
}

func TestDealDamageEachMultiplePermanentTargets(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 3, 4), b.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 3, 4), b.PlayerID())
	third := g.PutOnBattlefield(NewCreature("Third", "", 3, 4), b.PlayerID())
	targets := DeclarePermanentTargets("creatures", 1, 2, IsCreature)
	ability := Spell(DealDamageEach(This, DamageRecipients(EachPlayer, targets), Fixed(2)), WithTargets(targets))
	card := NewSorcery("Batch", "", WithAbility(ability))
	card.SetOwner(a.PlayerID())
	if err := ValidateCardDefinition("Batch", card); err != nil {
		t.Fatal(err)
	}
	obj := &StackObject{ID: uuid.New(), SourceID: card.ID(), Card: card, Controller: a.PlayerID(), Targets: []uuid.UUID{first.ID(), second.ID()}, TargetSpecs: expandTargetSpecs(ability.Targets(), []uuid.UUID{first.ID(), second.ID()}, 0), Effects: ability.Effects()}
	g.pushStack(obj)
	if obj.targetIncarnations[first.ID()] != first.incarnationID || obj.targetIncarnations[second.ID()] != second.incarnationID {
		t.Fatal("target incarnations were not captured")
	}
	if !g.isTargetStillLegal(first.ID(), card, a.PlayerID(), obj.TargetZones[first.ID()], obj.TargetSpecs[0]) {
		t.Fatal("selected creature became illegal before resolution")
	}
	clone := g.Clone()
	copyObj := clone.CopyStackObjectDirect(clone.StackPeek(), a.PlayerID(), false)
	if copyObj == nil || len(copyObj.TargetSpecs) != 2 {
		t.Fatal("copy lost the selected group")
	}
	cloneSecond := clone.FindPermanent(second.ID())
	clone.RemoveFromBattlefield(cloneSecond)
	clone.ResolveStackObject(copyObj)
	if clone.FindPermanent(first.ID()).Damage != 2 || clone.FindPermanent(third.ID()).Damage != 0 {
		t.Fatal("copy damaged an unselected or illegal permanent")
	}
	if clone.GetPlayer(a.PlayerID()).Life() != 18 || clone.GetPlayer(b.PlayerID()).Life() != 18 {
		t.Fatal("copy lost the player recipients")
	}
	if first.Damage != 0 || second.Damage != 0 || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("clone changed the original game")
	}
	g.ResolveStackObject(obj)
	if g.FindPermanent(first.ID()).Damage != 2 || g.FindPermanent(second.ID()).Damage != 2 || g.FindPermanent(third.ID()).Damage != 0 || a.Life() != 18 || b.Life() != 18 {
		t.Fatal("batch did not damage exactly the selected recipients")
	}
}

func TestDealDamageEachTargetIncarnationAndAbsentSource(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), a.PlayerID())
	card := NewCreature("Target", "", 2, 3)
	old := g.PutOnBattlefield(card, b.PlayerID())
	targets := DeclarePermanentTargets("creature", 1, 1, IsCreature)
	effect := DealDamageEach(This, DamageRecipients(EachPlayer, targets), Fixed(2))
	g.resolution.targetSpecs = []Target{targets.targetDeclaration}
	g.resolution.targetIncarnations = map[uuid.UUID]uuid.UUID{old.ID(): old.incarnationID}
	g.RemoveFromBattlefield(old)
	current := g.PutOnBattlefield(card, b.PlayerID())
	if err := ApplyEffect(g, effect, source.ID(), a.PlayerID(), []uuid.UUID{old.ID()}); err != nil {
		t.Fatal(err)
	}
	if current.Damage != 0 || a.Life() != 18 || b.Life() != 18 {
		t.Fatal("returned target received damage or player set was lost")
	}
	if err := ApplyEffect(g, effect, uuid.New(), a.PlayerID(), []uuid.UUID{old.ID()}); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 18 || b.Life() != 18 {
		t.Fatal("absent source dealt damage")
	}
}

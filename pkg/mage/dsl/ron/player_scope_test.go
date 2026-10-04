package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPlayerReadsUseDeclaredTypedTargets(t *testing.T) {
	for _, name := range []string{"Ancestral Recall", "Braingeyser"} {
		card := repositoryCard(t, "LEB", name)
		if err := mage.ValidateCardDefinition(name, card); err != nil {
			t.Fatal(err)
		}
		var reads int
		var walk func(mage.DefinitionMetadata)
		walk = func(n mage.DefinitionMetadata) {
			if n.Role == "player-ref" && n.Text == "Ref_TargetPlayer" {
				reads++
			}
			for _, child := range n.Children {
				walk(child)
			}
		}
		walk(mage.InspectDefinition(card.Abilities()[0]))
		if reads != 1 {
			t.Fatalf("%s: expected one typed player target read, got %d", name, reads)
		}
	}
}

func TestRONPlayerScopeRejectsInvalidReads(t *testing.T) {
	base := `Card(Name: "Scope", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [%s])`
	for _, rule := range []string{
		`Activated(PayMana([]), ActionList([PlayerAction(Ref_TargetPlayer, DrawNumberCards(Integer(1)))]))`,
		`Activated(PayMana([]), Targeted([TargetPermanent(AnyPermanent)], ActionList([PlayerAction(Ref_TargetPlayer, DrawNumberCards(Integer(1)))])))`,
		`Activated(PayMana([]), Targeted([TargetPlayer(AnyPlayer)], ActionList([PlayerAction(Ref_TargetPermanent, DrawNumberCards(Integer(1)))])))`,
		`Activated(PayMana([]), Targeted([TargetPlayer(AnyPlayer)], ActionList([PlayerAction(Ref_TargetPlayer(), DrawNumberCards(Integer(1)))])))`,
		`Activated(PayMana([]), Targeted([TargetPlayer(AnyPlayer)], ActionList([]))), Activated(PayMana([]), ActionList([PlayerAction(Ref_TargetPlayer, DrawNumberCards(Integer(1)))]))`,
	} {
		src := strings.Replace(base, "%s", rule, 1)
		nodes, err := Parse(strings.NewReader(src), "invalid-player.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("invalid player scope compiled: %s", rule)
		}
	}
}

func fillRONLibrary(t *testing.T, p *mage.BasePlayer, count int) {
	t.Helper()
	for range count {
		card := repositoryCard(t, "LEB", "Forest")
		card.SetOwner(p.PlayerID())
		p.AddToLibrary(card)
	}
}

func TestRONPlayerTargetInstancesCopiesAndClones(t *testing.T) {
	g, a, b := ronGame()
	fillRONLibrary(t, a, 12)
	fillRONLibrary(t, b, 12)
	first, second := repositoryCard(t, "LEB", "Ancestral Recall"), repositoryCard(t, "LEB", "Ancestral Recall")
	first.SetOwner(a.PlayerID())
	second.SetOwner(a.PlayerID())
	a.AddToHand(first)
	a.AddToHand(second)
	a.ManaPool().Add(core.Blue, 2)
	if err := g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByID(a.PlayerID(), second.ID(), []uuid.UUID{a.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	if cp := g.CopySpellOnStack(first.ID(), b.PlayerID(), true); cp == nil || len(cp.Targets) != 1 || cp.Targets[0] != a.PlayerID() {
		t.Fatal("spell copy did not use its new player target")
	}
	g.ResolveStack()
	if len(a.Hand()) != 6 || len(b.Hand()) != 3 {
		t.Fatalf("instance/copy targets: %d/%d", len(a.Hand()), len(b.Hand()))
	}
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 0 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 0 {
		t.Fatal("live resolution changed clone hands")
	}
	clone.ResolveStack()
	if len(clone.GetPlayer(a.PlayerID()).Hand()) != 3 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 3 {
		t.Fatal("clone lost its target bindings")
	}
	if len(a.Hand()) != 6 || len(b.Hand()) != 3 {
		t.Fatal("clone resolution changed live hands")
	}
}

func TestRONBraingeyserAnnouncementAndRollback(t *testing.T) {
	g, a, b := ronGame()
	fillRONLibrary(t, b, 6)
	spell := repositoryCard(t, "LEB", "Braingeyser")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Blue, 2)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{b.PlayerID()}, 4); err == nil {
		t.Fatal("unpaid X cast succeeded")
	}
	if len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 2 || !g.GetStack().IsEmpty() {
		t.Fatal("failed cast changed live state")
	}
	a.ManaPool().Add(core.Colorless, 4)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{uuid.New()}, 4); err == nil {
		t.Fatal("invalid player target cast succeeded")
	}
	if len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 6 {
		t.Fatal("invalid target consumed payment")
	}
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{b.PlayerID()}, 4); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(b.Hand()) != 4 || len(a.Hand()) != 0 || len(a.Graveyard()) != 1 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("announced X or explicit player was lost")
	}
}

func TestRONPlayerFactoryBindsReadsToEachDeclaration(t *testing.T) {
	src := `Card(Name: "Scoped Draw", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(Targeted([TargetPlayer(AnyPlayer)], ActionList([PlayerAction(Ref_TargetPlayer, DrawNumberCards(Integer(1))), PlayerAction(You, DrawNumberCards(Integer(1)))])))])`
	nodes, err := Parse(strings.NewReader(src), "scope.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	first, second := factory(), factory()
	a, b := first.Abilities()[0].(*mage.ActionDefinition), second.Abilities()[0].(*mage.ActionDefinition)
	if a.Targets()[0] == b.Targets()[0] {
		t.Fatal("factory instances share declarations")
	}
	for _, card := range []mage.Card{first, second} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	invalid := mage.NewInstant("Missing declaration", "", mage.WithAbility(mage.NewAction(mage.ActionSpell, mage.WithEffects(a.Effects()...))))
	if err := mage.ValidateCardDefinition(invalid.Name(), invalid); err == nil {
		t.Fatal("engine validator did not detect the missing player target declaration")
	}
	g, player, opponent := ronGame()
	fillRONLibrary(t, player, 3)
	fillRONLibrary(t, opponent, 3)
	first.SetOwner(player.PlayerID())
	player.AddToHand(first)
	if err := g.CastSpellByID(player.PlayerID(), first.ID(), []uuid.UUID{opponent.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if len(player.Hand()) != 1 || len(opponent.Hand()) != 1 {
		t.Fatal("explicit player scope escaped into its sibling action")
	}
}

func TestRONPlayerTargetActivationSurvivesSourceDeparture(t *testing.T) {
	src := `Card(Name: "Draw Source", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(TapPermanent(ThisPermanent), Targeted([TargetPlayer(AnyPlayer)], ActionList([PlayerAction(Ref_TargetPlayer, DrawNumberCards(Integer(1)))])))])`
	nodes, err := Parse(strings.NewReader(src), "activation.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	card := factory()
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	fillRONLibrary(t, b, 2)
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if len(b.Hand()) != 0 || !g.FindPermanent(source.ID()).Tapped {
		t.Fatal("draw timing or tap cost changed")
	}
	g.DestroyPermanent(g.FindPermanent(source.ID()))
	clone := g.Clone()
	clone.ResolveStack()
	if len(clone.GetPlayer(b.PlayerID()).Hand()) != 1 || len(b.Hand()) != 0 {
		t.Fatal("departed-source clone lost or shared its player target")
	}
	g.ResolveStack()
	if len(b.Hand()) != 1 {
		t.Fatal("source departure erased the player target")
	}
}

func TestRONPlayerTargetTriggerUsesPlacementBinding(t *testing.T) {
	src := `Card(Name: "Draw Entry", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), Targeted([TargetPlayer(AnyPlayer)], ActionList([PlayerAction(Ref_TargetPlayer, DrawNumberCards(Integer(1)))])))])`
	nodes, err := Parse(strings.NewReader(src), "trigger-player.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	card := factory()
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	fillRONLibrary(t, a, 2)
	card.SetOwner(b.PlayerID())
	g.PutOnBattlefield(card, b.PlayerID())
	if len(a.Hand()) != 0 {
		t.Fatal("trigger drew before resolution")
	}
	g.ResolveStack()
	if len(a.Hand()) != 1 || len(b.Hand()) != 0 {
		t.Fatal("trigger used its controller instead of the chosen target")
	}
}

func TestRONScopeTriggerRepairKeepsRealDeathActionsValid(t *testing.T) {
	for _, name := range []string{"Onulet", "Su-Chi"} {
		t.Run(name, func(t *testing.T) {
			card := repositoryCard(t, "ATQ", name)
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.DestroyPermanent(g.FindPermanent(source.ID()))
			if a.Life() != 20 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("death trigger acted before resolution")
			}
			g.ResolveStack()
			if name == "Onulet" && a.Life() != 22 {
				t.Fatal("death trigger did not gain two life")
			}
			if name == "Su-Chi" && a.ManaPool().TotalMana() != 4 {
				t.Fatal("death trigger did not add four mana")
			}
		})
	}
}

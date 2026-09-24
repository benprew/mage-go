package dsl_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/dsl"
)

func TestTypedMillComposition(t *testing.T) {
	player := dsl.TargetPlayer("player")
	body := dsl.Sequence(dsl.Mill(dsl.You, dsl.Fixed(2)), dsl.Draw(dsl.You, dsl.Fixed(1)))
	for _, card := range []mage.Card{
		dsl.NewInstant("Target", "", dsl.WithAbility(dsl.Spell(dsl.Mill(player, dsl.Fixed(2)), dsl.WithTargets(player)))),
		dsl.NewSorcery("Sequence", "", dsl.WithAbility(dsl.Spell(body))),
		dsl.NewSorcery("Mode", "", dsl.WithAbility(dsl.SpellModes(dsl.ModeOf("Mill", body), dsl.ModeOf("Draw", dsl.Draw(dsl.You, dsl.Fixed(1)))))),
		dsl.NewCreature("Entry target", "", 0, 4, dsl.WithAbility(dsl.WhenEvent(dsl.Enters(dsl.This), dsl.Mill(player, dsl.Fixed(4)), dsl.WithTargets(player)))),
		dsl.NewCreature("Paid activation", "", 1, 4, dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.Mana("{U}"), dsl.TapCost(dsl.This)), dsl.Mill(player, dsl.Fixed(2)), dsl.WithTargets(player)))),
		dsl.NewArtifact("Generic paid activation", "", dsl.WithAbility(dsl.Activate(dsl.Costs(dsl.Mana("{2}"), dsl.TapCost(dsl.This)), dsl.Mill(player, dsl.Fixed(2)), dsl.WithTargets(player)))),
		dsl.NewCreature("Trigger", "", 1, 1, dsl.WithAbility(dsl.WhenEvent(dsl.Dies(dsl.This), body))),
		dsl.NewCreature("Activation", "", 1, 1, dsl.WithAbility(dsl.Activate(dsl.Costs(), body))),
	} {
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTypedMillSequenceAfterSourcePhasesIn(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	source := dsl.NewArtifact("Modifier", "")
	g.PutOnBattlefield(source, b.PlayerID())
	g.AddMillReplacement(source.ID(), dsl.DoubleMillForOpponents(source.ID(), b.PlayerID()))
	a.SetLibrary([]mage.Card{
		dsl.NewArtifact("One", ""), dsl.NewArtifact("Two", ""),
		dsl.NewArtifact("Three", ""), dsl.NewArtifact("Four", ""),
		dsl.NewArtifact("Five", ""), dsl.NewArtifact("Six", ""),
	})
	body := dsl.Sequence(dsl.Mill(dsl.You, dsl.Fixed(1)), dsl.Draw(dsl.You, dsl.Fixed(1)))
	g.PhaseOut(source.ID())
	if err := mage.ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 1 || len(a.Hand()) != 1 || a.Hand()[0].Name() != "Two" {
		t.Fatal("phased source changed the mill or the following draw")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	if err := mage.ApplyEffect(g, body, source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 3 || len(a.Hand()) != 2 || a.Hand()[1].Name() != "Five" || len(a.Library()) != 1 {
		t.Fatal("returned modifier did not preserve the mill and draw sequence")
	}
}

func TestTypedMillUsesModifierCurrentController(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	source := dsl.NewArtifact("Modifier", "")
	g.PutOnBattlefield(source, a.PlayerID())
	g.AddMillReplacement(source.ID(), dsl.DoubleMillForOpponents(source.ID(), a.PlayerID()))
	if err := mage.ApplyEffect(g, dsl.GainControl(), b.PlayerID(), b.PlayerID(), []uuid.UUID{source.ID()}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		player mage.Player
		want   int
		drawn  string
	}{{a, 2, "Three"}, {b, 1, "Two"}} {
		tc.player.SetLibrary([]mage.Card{dsl.NewArtifact("One", ""), dsl.NewArtifact("Two", ""), dsl.NewArtifact("Three", ""), dsl.NewArtifact("Four", "")})
		body := dsl.Sequence(dsl.Mill(dsl.You, dsl.Fixed(1)), dsl.Draw(dsl.You, dsl.Fixed(1)))
		if err := mage.ApplyEffect(g, body, source.ID(), tc.player.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if len(tc.player.Graveyard()) != tc.want || len(tc.player.Hand()) != 1 || tc.player.Hand()[0].Name() != tc.drawn {
			t.Fatalf("wrong mill or following draw for %s: graveyard=%d hand=%v", tc.player.Name(), len(tc.player.Graveyard()), tc.player.Hand())
		}
	}
}

func TestTypedMillStaticReplacementSequence(t *testing.T) {
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	source := dsl.NewCreature("Static mill", "", 1, 4, dsl.WithMillReplacement(dsl.DoubleMillForOpponents))
	if err := mage.ValidateCardDefinition(source.Name(), source); err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(source, b.PlayerID())
	a.SetLibrary([]mage.Card{dsl.NewArtifact("One", ""), dsl.NewArtifact("Two", ""), dsl.NewArtifact("Three", ""), dsl.NewArtifact("Four", "")})
	body := dsl.Sequence(dsl.Mill(dsl.You, dsl.Fixed(1)), dsl.Draw(dsl.You, dsl.Fixed(1)))
	if err := mage.ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if len(a.Graveyard()) != 2 || len(a.Hand()) != 1 || a.Hand()[0].Name() != "Three" || len(a.Library()) != 1 {
		t.Fatal("static replacement did not preserve the immediate mill and draw sequence")
	}
}

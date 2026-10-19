package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func glyphDeathResultFactory(t *testing.T) mage.CardFactory {
	t.Helper()
	record := hostUntapRecord(t, "LEG", "Glyph of Reincarnation")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Glyph clause", factory()); err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONDeathResultReturnGlyph(t *testing.T) {
	for _, change := range []string{"normal", "control", "clone", "copy", "phased", "return-attacker", "indestructible", "exile", "replaced", "phase-cycle", "return-wall", "last-block", "invalid-target"} {
		t.Run(change, func(t *testing.T) {
			factory := glyphDeathResultFactory(t)
			g, a, b := ronGame()
			wall := g.PutOnBattlefield(repositoryCard(t, "LEB", "Wall of Wood"), b.PlayerID())
			attackers := make([]*mage.Permanent, 0, 2)
			for range 2 {
				c := mage.NewCreature("Attacker", "", 2, 5)
				c.SetOwner(a.PlayerID())
				p := g.PutOnBattlefield(c, a.PlayerID())
				attackers = append(attackers, p)
				g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: wall.ID(), AttackerID: p.ID()}})
				g.GetCombat().Reset()
			}
			replacement := mage.NewCreature("Replacement", "", 3, 4)
			replacement.SetOwner(a.PlayerID())
			a.AddToGraveyard(replacement)
			g.SetStep(core.EndCombat)
			g.SetStep(core.PostcombatMain)
			if change == "last-block" {
				g.SetActivePlayerIndex(1)
				for _, p := range attackers {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: wall.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					g.GetCombat().AddAttacker(p.ID(), a.PlayerID())
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: wall.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: wall.ID(), AttackerID: p.ID()}})
					g.GetCombat().Reset()
				}
			}
			spell := factory()
			spell.SetOwner(b.PlayerID())
			b.AddToHand(spell)
			b.ManaPool().Add(core.Green, 1)
			if err := g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{wall.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "replaced":
				for _, p := range attackers {
					g.AddExileIfWouldGoToGraveyardThisTurn(p.ID(), p.ID())
				}
			case "phase-cycle":
				for _, p := range attackers {
					g.PhaseIn(g.PhaseOut(p.ID()))
				}
			case "invalid-target":
				wall.GrantBaseAttr(core.Shroud)
			case "return-wall":
				g.BouncePermanentToHand(wall)
				b.RemoveFromHand(wall.Card.ID())
				g.PutOnBattlefield(wall.Card, b.PlayerID())
			case "control":
				for _, p := range attackers {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: wall.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				}
			case "phased":
				for _, p := range attackers {
					g.PhaseOut(p.ID())
				}
			case "return-attacker":
				for _, p := range attackers {
					g.BouncePermanentToHand(p)
					a.RemoveFromHand(p.Card.ID())
					g.PutOnBattlefield(p.Card, a.PlayerID())
				}
			case "indestructible":
				for _, p := range attackers {
					p.GrantBaseAttr(core.Indestructible)
				}
			case "exile":
				for _, p := range attackers {
					g.ExilePermanent(p)
				}
			case "copy":
				if g.CopySpellOnStack(spell.ID(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			for _, p := range attackers {
				g.AddRegenerationShield(p.ID())
			}
			original := g
			if change == "clone" {
				g = g.Clone()
			}
			collection, ok := mage.InspectPermanentCollection(mage.MatchingPermanents(mage.IsCreature))
			if !ok {
				t.Fatal("missing creature collection")
			}
			watched := collection.Select(g)
			wantProof := change == "normal" || change == "control" || change == "clone" || change == "copy" || change == "phase-cycle" || change == "last-block"
			for range 2 {
				if certain, known := g.InspectSpellDestruction(g.StackPeek(), watched); certain != wantProof || known != (change != "replaced") {
					t.Fatalf("wrong destruction proof %v/%v", certain, known)
				}
				if certain, known := g.InspectSpellDestruction(g.StackPeek(), nil); certain || known != (change != "replaced") {
					t.Fatal("incomplete death-return tree")
				}
			}
			g.ResolveStack()
			returned := change == "normal" || change == "control" || change == "clone" || change == "copy" || change == "phase-cycle"
			if (g.FindPermanent(replacement.ID()) != nil) != returned {
				t.Fatal("wrong return count or historical graveyard")
			}
			if returned && g.FindPermanent(replacement.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("wrong owner control")
			}
			if change == "clone" && original.FindPermanent(replacement.ID()) != nil {
				t.Fatal("clone changed original")
			}
		})
	}
}

func TestRONDeathResultReturnInvalid(t *testing.T) {
	valid := `ActionForEachPermanentThatDiedThisWay([ReturnACardFromPlayersGraveyardToBattlefield(IsCardtype(Creature),You,[EntersUnderOwnersControl])])`
	for _, body := range []string{
		valid,
		`DestroyEachPermanent(IsCardtype(Creature)),GainLife(Integer(1)),` + valid,
		`DestroyEachPermanent(IsCardtype(Creature)),ActionForEachPermanentThatDiedThisWay`,
		`DestroyEachPermanent(IsCardtype(Creature)),ActionForEachPermanentThatDiedThisWay()`,
		`DestroyEachPermanent(IsCardtype(Creature)),ActionForEachPermanentThatDiedThisWay([])`,
		`DestroyEachPermanent(IsCardtype(Creature)),ActionForEachPermanentThatDiedThisWay([GainLife(Integer(1))])`,
		`DestroyEachPermanent(IsCardtype(Creature)),` + strings.ReplaceAll(valid, "EntersUnderOwnersControl", "EntersUnderOwnersControl()"),
		`DestroyEachPermanent(IsCardtype(Creature)),` + strings.ReplaceAll(valid, "You", "ControllerOfPermanentTheLastTimeItWasBlockedByPermanentThisTurn(EachablePermanent(),ThisPermanent)"),
		`DestroyEachPermanent(IsCardtype(Creature)),` + valid + `,TapPermanent(EachablePermanent)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", body)
		}
	}
}

func TestRONDeathResultReturnInstances(t *testing.T) {
	factory := glyphDeathResultFactory(t)
	g, a, b := ronGame()
	original := make([]mage.DamageSource, 0, 2)
	victims := make([]*mage.Permanent, 0, 2)
	walls := make([]*mage.Permanent, 0, 2)
	for _, player := range []mage.Player{a, b} {
		wall := g.PutOnBattlefield(repositoryCard(t, "LEB", "Wall of Wood"), player.PlayerID())
		card := mage.NewCreature("Victim", "", 2, 5)
		card.SetOwner(player.PlayerID())
		victim := g.PutOnBattlefield(card, player.PlayerID())
		walls = append(walls, wall)
		victims = append(victims, victim)
		original = append(original, g.CaptureDamageSource(victim.ID()))
	}
	for i := range 2 {
		g.SetActivePlayerIndex(i)
		g.GetCombat().AddAttacker(victims[i].ID(), walls[1-i].ControllerID())
		g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: walls[1-i].ID(), AttackerID: victims[i].ID()}})
		g.GetCombat().Reset()
	}
	g.SetStep(core.EndCombat)
	g.SetStep(core.PostcombatMain)
	for i, player := range []mage.Player{a, b} {
		spell := factory()
		spell.SetOwner(player.PlayerID())
		player.AddToHand(spell)
		player.ManaPool().Add(core.Green, 1)
		if err := g.CastSpellByID(player.PlayerID(), spell.ID(), []uuid.UUID{walls[1-i].ID()}, 0); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveStack()
	for i, p := range victims {
		current := g.FindPermanent(p.ID())
		if current == nil || current.ControllerID() != p.Card.Owner() || g.CaptureDamageSource(p.ID()).SameObject(original[i]) {
			t.Fatal("factory target or death-member binding crossed instances")
		}
	}
}

func TestRONDeathResultReturnContexts(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`DestroyEachPermanent(IsCardtype(Creature)) ActionForEachPermanentThatDiedThisWay([ReturnACardFromPlayersGraveyardToBattlefield(IsCardtype(Creature),You,[EntersUnderOwnersControl])])`), "contexts.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, ctx := range []scope{{}, {resolving: true, enteringSource: true}, {resolving: true, drawReplacementBody: true}} {
		if _, err := compileDeathResultReturn(nodes[0], nodes[1], ctx); err == nil {
			t.Fatal("invalid context accepted")
		}
	}
}

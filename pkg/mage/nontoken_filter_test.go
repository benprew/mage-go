package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestNontokenFilterPhysicalCopyStatus(t *testing.T) {
	factory, err := NewTokenDefinitionFactory(func() TokenDefinition {
		return DefineToken("Same", TokenTypes(core.TypeCreature), TokenPT(2, 3))
	})
	if err != nil {
		t.Fatal(err)
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if err := ApplyEffect(g, CreateOneToken(You, factory(), BindPermanent("token")), a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	token := g.FilterBattlefield(IsToken)[0]
	values, err := g.CaptureCopiableValues(token)
	if err != nil {
		t.Fatal(err)
	}
	copied, err := BuildEntryCopy(entryCopyPhysical(t), values)
	if err != nil {
		t.Fatal(err)
	}
	copied.SetOwner(b.PlayerID())
	normal := g.PutOnBattlefield(copied, a.PlayerID())
	filter := Not(IsToken)
	if !filter.Match(normal, g) || filter.Match(token, g) || normal.Name() != token.Name() {
		t.Fatal("copy acquired physical token status")
	}
	clone := g.Clone()
	g.PhaseOut(normal.ID())
	if len(g.FilterBattlefield(filter)) != 0 || len(clone.FilterBattlefield(filter)) != 1 {
		t.Fatal("phase or clone selection")
	}
	g.PhaseIn([]uuid.UUID{normal.ID()})
	if len(g.FilterBattlefield(filter)) != 1 {
		t.Fatal("phase cycle lost selection")
	}
	tokenValues, err := g.CaptureCopiableValues(normal)
	if err != nil {
		t.Fatal(err)
	}
	tokenCopy, err := BuildEntryCopy(token.Card, tokenValues)
	if err != nil {
		t.Fatal(err)
	}
	if !tokenCopy.IsToken() || copied.IsToken() {
		t.Fatal("entry copy changed physical status")
	}
}

func TestNontokenFilterSimultaneousExile(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 3), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 3), b.PlayerID())
	token := g.PutOnBattlefield(NewToken("First", 2, 3, []core.CardType{core.TypeCreature}, nil), a.PlayerID())
	events := 0
	watcher := NewArtifact("Observer", "", WithAbility(NewTriggered(core.EvtZoneChange, false).SetCondition(func(evt *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		if evt.SourceID == first.ID() || evt.SourceID == second.ID() {
			events++
			if g.FindPermanent(first.ID()) != nil || g.FindPermanent(second.ID()) != nil || g.FindPermanent(token.ID()) == nil {
				t.Error("zone event observed partial nontoken batch")
			}
			if reader.FindPermanent(first.ID()) == nil || reader.FindPermanent(second.ID()) == nil {
				t.Error("leave trigger lost the former battlefield")
			}
		}
		return false
	})))
	g.PutOnBattlefield(watcher, a.PlayerID())
	effect := ExileEach(MatchingPermanents(And(Not(IsToken), IsCreature)))
	if err := ApplyEffect(g, effect, a.PlayerID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if events != 2 || g.FindExiledCard(first.ID()) == nil || g.FindExiledCard(second.ID()) == nil || g.FindPermanent(token.ID()) == nil {
		t.Fatal("wrong exile batch", events)
	}
}

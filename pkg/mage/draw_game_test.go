package mage

import (
	"testing"

	"github.com/google/uuid"

	. "github.com/benprew/mage-go/pkg/mage/core"
)

func TestDrawGameResult(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	before := g.Clone()
	if err := ApplyEffect(g, DrawGame(), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !g.IsGameOver() || !g.IsDraw() || g.Winner() != "" || !a.IsAlive() || !b.IsAlive() || a.Life() != 20 || b.Life() != 20 {
		t.Fatal("draw must end without a winner or loser")
	}
	if before.IsGameOver() {
		t.Fatal("draw changed earlier clone")
	}
	after := g.Clone()
	if !after.IsDraw() || after.Winner() != "" {
		t.Fatal("clone lost draw result")
	}
	a.SetLost()
	if g.Winner() != "" || !g.IsDraw() || !after.GetPlayer(a.PlayerID()).IsAlive() {
		t.Fatal("terminal draw changed")
	}
	if err := ApplyEffect(before, DrawGame(), uuid.Nil, b.PlayerID(), nil); err != nil || !before.IsDraw() {
		t.Fatal("independent clone draw", err)
	}
	if err := ValidateCardDefinition("Draw", NewArtifact("Draw", "", WithAbility(Activate(Costs(), DrawGame())))); err != nil {
		t.Fatal(err)
	}
}

func TestDrawGameStopsResolution(t *testing.T) {
	for _, shape := range []string{"sequence", "pipeline", "loop", "stack effects", "permanent spell"} {
		t.Run(shape, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			gain := GainLifeFor(You, Fixed(7))
			body := Sequence(DrawGame(), gain)
			switch shape {
			case "pipeline":
				body = Pipeline("", EffectProperties{}, DrawGame(), gain)
			case "loop":
				body = ForEachPermanent(AnyPermanent, body, "")
			}
			g.PutOnBattlefield(NewArtifact("Member", ""), a.PlayerID())
			g.stack.Push(&StackObject{ID: uuid.New(), Controller: a.PlayerID(), IsAbility: true, Effects: []Effect{gain}})
			obj := &StackObject{ID: uuid.New(), Controller: a.PlayerID(), IsAbility: true, Effects: []Effect{body, gain}}
			if shape == "permanent spell" {
				obj.Card = NewArtifact("Spell", "")
				obj.Card.SetOwner(a.PlayerID())
				obj.IsAbility = false
			}
			g.stack.Push(obj)
			g.ResolveStack()
			if !g.IsDraw() || a.Life() != 20 || g.StackSize() != 1 {
				t.Fatal("resolution continued after draw", a.Life(), g.StackSize())
			}
			if len(a.Graveyard()) != 0 || len(g.AllBattlefield()) != 1 {
				t.Fatal("spell moved after draw")
			}
			g.ResolveTopOfStack()
			g.ResolveStackObject(&StackObject{ID: uuid.New(), Controller: a.PlayerID(), Effects: []Effect{gain}})
			if err := ApplyEffect(g, gain, uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 20 || g.StackSize() != 1 {
				t.Fatal("ended game resumed resolution")
			}
		})
	}
}

func TestDrawGameStopsStateAndPriority(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	marker := NewCreature("Doomed", "", 1, 1)
	p := g.PutOnBattlefield(marker, a.PlayerID())
	effect := Sequence(FuncEffect("", EffectProperties{}, func(g *Game, _, _ uuid.UUID, _ []uuid.UUID) error {
		g.MutablePermanent(p.ID()).Damage = 1
		a.SetLife(0)
		return nil
	}), DrawGame())
	g.stack.Push(&StackObject{ID: uuid.New(), Controller: a.PlayerID(), IsAbility: true, Effects: []Effect{effect}})
	calls := 0
	g.onPriority = func(*Game, int, bool) PriorityAction { calls++; return PriorityAction{Type: PriorityPass} }
	g.runPriorityRound(false)
	before := calls
	g.CheckStateBasedActions()
	g.RunStepWithPriority(Draw)
	if !g.IsDraw() || !a.IsAlive() || g.FindPermanent(p.ID()) == nil || calls != before || a.DrewFromEmpty() {
		t.Fatal("state actions, step or priority ran after draw")
	}
}

func TestDrawGameTypedLoopAndAnte(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	ac, bc := ownedTestCard(a.PlayerID(), "Ante A"), ownedTestCard(b.PlayerID(), "Ante B")
	a.SetLibrary([]Card{ac})
	b.SetLibrary([]Card{bc})
	g, err := NewGameWithAnte(a, b, []Card{ac}, []Card{bc})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		g.PutOnBattlefield(NewArtifact("Member", ""), a.PlayerID())
	}
	body := ForEach(MatchingPermanents(AnyPermanent), BindPermanent("member"), Sequence(GainLifeFor(You, Fixed(1)), DrawGame(), GainLifeFor(You, Fixed(50))))
	if err := ApplyEffect(g, body, uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if a.Life() != 21 || !g.IsDraw() {
		t.Fatal("typed loop continued after draw")
	}
	a.SetLost()
	if _, err := g.AnteResult(); err == nil || g.anteSettled || ac.Owner() != a.PlayerID() || bc.Owner() != b.PlayerID() {
		t.Fatal("draw settled ante")
	}
}

func TestDrawGameTurnBoundary(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewArtifact("Draw trigger", "", WithAbility(At(YourUpkeep, DrawGame()))), a.PlayerID())
	a.ManaPool().Add(White, 3)
	g.Run(0, Untap, 3)
	if !g.IsDraw() || g.GetStep() != Upkeep || g.CurrentTurn() != 1 {
		t.Fatal("turn continued after draw", g.GetStep(), g.CurrentTurn())
	}
	g.Run(0, Untap, 3)
	if g.GetStep() != Upkeep || g.CurrentTurn() != 1 {
		t.Fatal("terminal turn resumed")
	}
}

func TestDrawGamePreservesEarlierResult(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.SetLost()
	if err := ApplyEffect(g, DrawGame(), uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if g.IsDraw() || g.Winner() != b.Name() {
		t.Fatal("draw replaced an earlier win")
	}
}

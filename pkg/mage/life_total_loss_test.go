package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLifeTotalLossTiming(t *testing.T) {
	for _, life := range []int{0, -4} {
		for _, protected := range []bool{false, true} {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			if protected {
				g.PutOnBattlefield(NewArtifact("Exemption", "", WithStaticAbility(PreventLifeTotalLossStatic(You))), a.PlayerID())
			}
			a.SetLife(life)
			if !a.IsAlive() || g.IsGameOver() {
				t.Fatal("life total caused loss before state-based actions")
			}
			g.CheckStateBasedActions()
			if a.IsAlive() != protected || g.IsGameOver() == protected {
				t.Fatal("wrong life-total loss decision")
			}
			if !protected {
				a.SetLife(20)
				if a.IsAlive() || g.Winner() != b.Name() {
					t.Fatal("life gain reversed recorded loss")
				}
			}
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.SetLife(0)
	a.SetLife(1)
	g.CheckStateBasedActions()
	if !a.IsAlive() {
		t.Fatal("temporary zero life caused loss")
	}
	a.SetLife(0)
	b.SetLife(-2)
	g.CheckStateBasedActions()
	if !g.IsDraw() || g.Winner() != "" {
		t.Fatal("simultaneous loss did not draw")
	}
}

func TestLifeTotalLossStaticLifecycle(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	card := NewArtifact("Exemption", "", WithStaticAbility(PreventLifeTotalLossStatic(You)))
	card.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(card, a.PlayerID())
	a.SetLife(-3)
	g.CheckStateBasedActions()
	if !a.IsAlive() {
		t.Fatal("current controller unprotected")
	}
	clone := g.Clone()
	g.AddControlEffect(ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	if g.PreventsLifeTotalLoss(a.PlayerID()) || !g.PreventsLifeTotalLoss(b.PlayerID()) || !clone.PreventsLifeTotalLoss(a.PlayerID()) {
		t.Fatal("control or clone isolation")
	}
	ids := g.PhaseOut(p.ID())
	if g.PreventsLifeTotalLoss(b.PlayerID()) {
		t.Fatal("phased source supplied rule")
	}
	g.PhaseIn(ids)
	if !g.PreventsLifeTotalLoss(b.PlayerID()) {
		t.Fatal("phase cycle lost rule")
	}
	g.AddContinuousEffect(RemoveAllAbilities(p.ID(), core.EndOfTurn))
	if g.PreventsLifeTotalLoss(b.PlayerID()) {
		t.Fatal("removed ability supplied rule")
	}
	a.SetLife(20)
	g.RunStepWithPriority(core.Cleanup)
	if !g.PreventsLifeTotalLoss(b.PlayerID()) {
		t.Fatal("restored ability missing")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	if g.PreventsLifeTotalLoss(b.PlayerID()) {
		t.Fatal("departed source supplied LKI rule")
	}
	b.RemoveFromHand(card.ID())
	g.PutOnBattlefield(card, a.PlayerID())
	if !g.PreventsLifeTotalLoss(a.PlayerID()) || g.PreventsLifeTotalLoss(b.PlayerID()) {
		t.Fatal("return reused old rule")
	}
	clone.PhaseOut(p.ID())
	clone.CheckStateBasedActions()
	if clone.GetPlayer(a.PlayerID()).IsAlive() || !a.IsAlive() {
		t.Fatal("clone loss state shared")
	}
}

func TestLifeTotalLossOtherCausesAndClones(t *testing.T) {
	for _, cause := range []string{"poison", "draw", "explicit"} {
		t.Run(cause, func(t *testing.T) {
			a := &lifeLossWrappedPlayer{NewBasePlayer("A")}
			b := NewBasePlayer("B")
			g := NewGame(a, b)
			g.PutOnBattlefield(NewArtifact("Exemption", "", WithStaticAbility(PreventLifeTotalLossStatic(You))), a.PlayerID())
			a.SetLife(-1)
			switch cause {
			case "poison":
				a.AddPoisonCounters(10)
			case "draw":
				a.DrawCard()
			case "explicit":
				a.SetLost()
			}
			g.CheckStateBasedActions()
			if a.IsAlive() || !g.IsGameOver() || g.Winner() != b.Name() {
				t.Fatal("exemption blocked another loss")
			}
			clone := g.Clone()
			clone.GetPlayer(a.PlayerID()).SetLife(5)
			if clone.GetPlayer(a.PlayerID()).IsAlive() {
				t.Fatal("clone lost explicit loss at negative life")
			}
		})
	}
}

type lifeLossWrappedPlayer struct{ *BasePlayer }

func TestLifeTotalLossDeclarationsAndInstances(t *testing.T) {
	for _, subject := range []PlayerRef{nil, FromPlayer(BindPlayer("unbound"))} {
		if err := ValidateCardDefinition("Invalid", NewArtifact("Invalid", "", WithStaticAbility(PreventLifeTotalLossStatic(subject)))); err == nil {
			t.Fatal("invalid subject accepted")
		}
	}
	if err := ValidateCardDefinition("Invalid", NewInstant("Invalid", "", WithStaticAbility(PreventLifeTotalLossStatic(You)))); err == nil {
		t.Fatal("spell source accepted")
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	for range 2 {
		g.PutOnBattlefield(NewArtifact("Independent", "", WithStaticAbility(PreventLifeTotalLossStatic(You))), a.PlayerID())
	}
	g.PhaseOut(g.AllBattlefield()[0].ID())
	a.SetLife(0)
	g.CheckStateBasedActions()
	if !a.IsAlive() || g.PreventsLifeTotalLoss(uuid.New()) {
		t.Fatal("independent or absent subjects")
	}
}

func TestLifeTotalLossEntryCopy(t *testing.T) {
	a := &optionalEntryPlayer{BasePlayer: NewBasePlayer("A"), accept: true}
	b := NewBasePlayer("B")
	g := NewGame(a, b)
	donorPlan := entryCopyPlan(t, "Donor", func() []CardOption { return []CardOption{WithStaticAbility(PreventLifeTotalLossStatic(You))} })
	donor, err := donorPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	g.PutOnBattlefield(donor, b.PlayerID())
	copyPlan := entryCopyPlan(t, "Copy", func() []CardOption { return []CardOption{WithAbility(MayCopyAsEnters(IsCreature))} })
	card, err := copyPlan.Build()
	if err != nil {
		t.Fatal(err)
	}
	p := g.PutOnBattlefield(card, a.PlayerID())
	a.SetLife(0)
	g.CheckStateBasedActions()
	if !a.IsAlive() {
		t.Fatal("entry copy lost static rule")
	}
	g.BouncePermanentToHand(p)
	g.CheckStateBasedActions()
	if a.IsAlive() {
		t.Fatal("departed copy kept rule")
	}
}

func TestLifeTotalLossSimultaneousSnapshot(t *testing.T) {
	for _, protected := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		exemption := g.PutOnBattlefield(NewArtifact("Exemption", "", WithStaticAbility(PreventLifeTotalLossStatic(You))), a.PlayerID())
		if !protected {
			g.PhaseOut(exemption.ID())
		}
		dying := g.PutOnBattlefield(NewCreature("Dying", "", 1, 1), b.PlayerID())
		g.MutablePermanent(dying.ID()).Damage = 1
		a.SetLife(0)
		replaced := 0
		g.effects.AddReplacement(&testPrioritizedReplacement{
			matchFunc: func(action Action) bool { _, ok := action.(*DestroyPermanentAction); return ok },
			replaceFunc: func(action Action, game *Game) Action {
				replaced++
				a.SetLife(1)
				if protected {
					game.PhaseOut(exemption.ID())
				} else {
					game.PhaseIn([]uuid.UUID{exemption.ID()})
				}
				return action
			},
		})
		g.CheckStateBasedActions()
		if replaced != 1 || g.FindPermanent(dying.ID()) != nil || a.IsAlive() != protected {
			t.Fatal("loss did not use pre-batch exemption and life")
		}
	}
}

func TestLifeTotalLossPureQuery(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.PutOnBattlefield(NewArtifact("Exemption", "", WithStaticAbility(PreventLifeTotalLossStatic(You))), a.PlayerID())
	checks := 0
	g.AddContinuousEffect(&handSizeQueryProbeEffect{ContinuousEffect: FuncContinuousEffect(core.LayerPT, core.Indefinite, func(*Game, uuid.UUID) error { return nil }), checks: &checks})
	before := checks
	a.SetLife(0)
	for range 3 {
		if !g.PreventsLifeTotalLoss(a.PlayerID()) || g.IsGameOver() || !a.IsAlive() {
			t.Fatal("query changed player state")
		}
	}
	if checks != before {
		t.Fatal("query evaluated unrelated effect lifetimes")
	}
}

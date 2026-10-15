package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLifeExchangeTotals(t *testing.T) {
	for _, totals := range [][2]int{{5, 18}, {18, 5}, {5, 5}, {-3, 7}, {0, 7}} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		a.SetLife(totals[0])
		b.SetLife(totals[1])
		if err := ApplyEffect(g, ExchangeLifeTotals(You, ActivePlayer), uuid.Nil, b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if a.Life() != totals[1] || b.Life() != totals[0] {
			t.Fatalf("%v became %d/%d", totals, a.Life(), b.Life())
		}
	}
}

func TestLifeExchangeRestrictions(t *testing.T) {
	for _, gain := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		a.SetLife(5)
		b.SetLife(18)
		g.PutOnBattlefield(NewArtifact("Restriction", "", WithStaticAbility(FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(g *Game, _ uuid.UUID) error {
			if gain {
				g.effects.Rules.AddCantGainLife(a.PlayerID())
			} else {
				g.effects.Rules.AddCantLoseLife(b.PlayerID())
			}
			return nil
		}))), a.PlayerID())
		if err := ApplyEffect(g, ExchangeLifeTotals(You, ActivePlayer), uuid.Nil, b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 5 || b.Life() != 18 {
			t.Fatal("blocked exchange changed a total")
		}
		if gain {
			g.PlayerGainLife(a, 2)
			if a.Life() != 5 {
				t.Fatal("ordinary gain ignored restriction")
			}
		} else {
			g.PlayerLoseLife(b, 2)
			if b.Life() != 18 || LifePayCost(1).CanPay(uuid.Nil, b.PlayerID(), g) || !LifePayCost(0).CanPay(uuid.Nil, b.PlayerID(), g) {
				t.Fatal("ordinary loss or payment ignored restriction")
			}
		}
	}
}

func TestLifeExchangeReplacementsAndObservations(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.SetLife(5)
	b.SetLife(18)
	g.AddReplacementEffect(&lifeDrawRewrite{replace: func(gain *LifeGainAction, g *Game) Action {
		if g.GetPlayer(a.PlayerID()).Life() != 5 || g.GetPlayer(b.PlayerID()).Life() != 18 {
			t.Fatal("replacement saw partial exchange")
		}
		return NewLifeGainAction(gain.ActionSource(), gain.PlayerID(), gain.Amount()*2)
	}})
	g.PutOnBattlefield(NewArtifact("Observer", "", WithAbility(WheneverYouGainLifeTrigger(LoseLifeFor(You, LifeTotalOf(ActivePlayer)), false))), a.PlayerID())
	observations := 0
	observer := g.PutOnBattlefield(NewArtifact("Snapshot observer", "", WithAbility(NewTriggered(core.EvtLifeGained, false).SetCondition(func(_ *core.GameEvent, reader GameReader, _, _ uuid.UUID) bool {
		observations++
		if reader.GetPlayer(a.PlayerID()).Life() != 31 || reader.GetPlayer(b.PlayerID()).Life() != 5 {
			t.Fatal("event saw partial exchange")
		}
		return false
	}))), a.PlayerID())
	if err := ApplyEffect(g, ExchangeLifeTotals(You, ActivePlayer), uuid.Nil, b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if a.Life() != 31 || b.Life() != 5 || g.StackSize() != 1 || observations != 1 {
		t.Fatal("replacement or gain event lost", a.Life(), b.Life(), g.StackSize())
	}
	clone := g.Clone()
	g.BouncePermanentToHand(observer)
	g.ResolveStack()
	if a.Life() != 0 || clone.GetPlayer(a.PlayerID()).Life() != 31 {
		t.Fatal("trigger or clone state mixed")
	}
}

func TestLifeExchangeInvalidAndMissing(t *testing.T) {
	for _, effect := range []Effect{ExchangeLifeTotals(nil, You), ExchangeLifeTotals(You, nil), ExchangeLifeTotals(You, FromPlayer(BindPlayer("unbound")))} {
		card := NewInstant("Invalid", "", WithAbility(Spell(effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid exchange accepted")
		}
	}
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.SetLife(5)
	for _, effect := range []Effect{ExchangeLifeTotals(You, You), ExchangeLifeTotals(You, FromPlayer(BindPlayer("absent")))} {
		if err := ApplyEffect(g, effect, uuid.Nil, a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if a.Life() != 5 || b.Life() != 20 {
			t.Fatal("invalid participants changed life")
		}
	}
}

func TestLifeExchangeRestrictionServices(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	if err := ApplyEffect(g, PlayerLifeManaThisTurn(You, 1, core.Green), uuid.Nil, a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.effects.Rules.AddCantLoseLife(a.PlayerID())
	if len(g.GetPlayerManaAbilities(a.PlayerID())) != 0 || g.CanAfford(a.PlayerID(), core.ManaCost{Green: 1}, nil) {
		t.Fatal("prohibited life payment supplied mana")
	}
	source := g.PutOnBattlefield(NewCreature("Source", "", 1, 1), b.PlayerID())
	g.effects.Rules.AddCantLoseLife(a.PlayerID())
	replacements := 0
	g.AddReplacementEffect(&testPrioritizedReplacement{
		matchFunc: func(action Action) bool { _, ok := action.(*damageLifeLossAction); return ok },
		replaceFunc: func(action Action, _ *Game) Action {
			replacements++
			return action
		},
	})
	g.DealDamageToPlayer(a, 3, source.ID())
	if a.Life() != 20 || replacements != 0 {
		t.Fatal("prohibited loss reached damage life replacements")
	}
}

func TestLifeExchangeRestrictionLifetimeAndRollback(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	a.SetLife(5)
	b.SetLife(18)
	source := g.PutOnBattlefield(NewArtifact("Restriction", "", WithStaticAbility(FuncContinuousEffect(core.LayerAbility, core.WhileOnBattlefield, func(game *Game, id uuid.UUID) error {
		game.effects.Rules.AddCantGainLife(game.FindPermanent(id).ControllerID())
		return nil
	}))), a.PlayerID())
	clone := g.Clone()
	g.PhaseOut(source.ID())
	g.ApplyContinuousEffects()
	if !g.CanGainLife(a.PlayerID()) || clone.CanGainLife(a.PlayerID()) {
		t.Fatal("phasing or clone permission state mixed")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	g.ApplyContinuousEffects()
	if g.CanGainLife(a.PlayerID()) {
		t.Fatal("phase-in did not restore prohibition")
	}
	g.AddContinuousEffect(RemoveAllAbilities(source.ID(), core.EndOfTurn))
	if !g.CanGainLife(a.PlayerID()) {
		t.Fatal("removed ability kept prohibition")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if g.CanGainLife(a.PlayerID()) {
		t.Fatal("cleanup did not restore prohibition")
	}
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	if !g.CanGainLife(a.PlayerID()) {
		t.Fatal("departed source kept prohibition")
	}
	a.ManaPool().Add(core.Red, 1)
	tx, err := g.prepareCostPaymentTransaction(costPaymentSpec{Controller: a.PlayerID(), Mana: core.ManaCost{Red: 1}, Costs: []Cost{LifePayCost(1)}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	g.effects.Rules.AddCantLoseLife(a.PlayerID())
	if err := tx.Commit(); err == nil || a.Life() != 5 || a.ManaPool().Count(core.Red) != 1 {
		t.Fatal("stale payment spent resources")
	}
}

type lifeExchangeLossRewrite struct {
	replacementBase
	rewrite func(*LifeLossAction, *Game) Action
}

func (*lifeExchangeLossRewrite) Matches(a Action, _ GameReader) bool {
	_, ok := a.(*LifeLossAction)
	return ok
}
func (*lifeExchangeLossRewrite) IsActive(GameReader) bool   { return true }
func (r *lifeExchangeLossRewrite) Clone() ReplacementEffect { c := *r; return &c }
func (r *lifeExchangeLossRewrite) Replace(a Action, g *Game) Action {
	return r.rewrite(a.(*LifeLossAction), g)
}

func TestLifeExchangeReplacedLossAndGain(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		a.SetLife(5)
		b.SetLife(18)
		lifeDrawLibrary(a, 20)
		g.PutOnBattlefield(NewArtifact("Draw replacement", "", WithStaticAbility(DrawInsteadOfLifeGain(You))), a.PlayerID())
		g.AddReplacementEffect(&lifeExchangeLossRewrite{rewrite: func(loss *LifeLossAction, _ *Game) Action {
			if cancel {
				return nil
			}
			return NewLifeLossAction(loss.ActionSource(), loss.PlayerID(), 1)
		}})
		g.AddPreventionShield(b.PlayerID(), 20)
		if err := ApplyEffect(g, ExchangeLifeTotals(You, ActivePlayer), uuid.Nil, b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := 17
		if cancel {
			want = 18
		}
		if a.Life() != 5 || b.Life() != want || len(a.Hand()) != 13 {
			t.Fatal("replacement changed independent exchange leg", a.Life(), b.Life(), len(a.Hand()))
		}
	}
}

func TestLifeExchangeDelayedSourceIdentity(t *testing.T) {
	for _, phased := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		a.SetLife(5)
		b.SetLife(18)
		card := NewArtifact("Source", "")
		card.SetOwner(b.PlayerID())
		source := g.PutOnBattlefield(card, a.PlayerID())
		saved := g.CaptureDamageSource(source.ID())
		if err := ApplyEffect(g, AtNext(EachEndStep, ExchangeLifeTotals(ControllerOfWithLKI(This), You)), source.ID(), b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if phased {
			g.PhaseOut(source.ID())
		} else {
			g.BouncePermanentToHand(source)
			b.RemoveFromHand(card.ID())
			g.PutOnBattlefield(card, b.PlayerID())
		}
		refs := g.WaitingDelayedPermanentSources()
		if len(refs) != 1 || !refs[0].SameObject(saved) || a.Life() != 5 || b.Life() != 18 {
			t.Fatal("waiting query lost exact source or changed life")
		}
		clone := g.Clone()
		g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
		g.PutTriggersOnStack()
		g.ResolveStack()
		if a.Life() != 18 || b.Life() != 5 || clone.GetPlayer(a.PlayerID()).Life() != 5 {
			t.Fatal("delayed exchange followed returned source or changed clone")
		}
	}
}

func TestLifeExchangeReplacementOrder(t *testing.T) {
	for _, choice := range []int{0, 1} {
		a := &drawReplacementChooser{BasePlayer: NewBasePlayer("A"), choice: choice}
		b := NewBasePlayer("B")
		g := NewGame(a, b)
		a.SetLife(5)
		b.SetLife(18)
		g.AddReplacementEffect(&lifeDrawRewrite{replace: func(gain *LifeGainAction, _ *Game) Action {
			return NewLifeGainAction(gain.ActionSource(), gain.PlayerID(), gain.Amount()*2)
		}})
		g.AddReplacementEffect(&lifeDrawRewrite{replace: func(gain *LifeGainAction, _ *Game) Action {
			return NewLifeGainAction(gain.ActionSource(), gain.PlayerID(), gain.Amount()+1)
		}})
		if err := ApplyEffect(g, ExchangeLifeTotals(You, ActivePlayer), uuid.Nil, b.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		want := 32
		if choice == 1 {
			want = 33
		}
		if a.Life() != want || b.Life() != 5 {
			t.Fatal("affected player's replacement order lost", choice, a.Life(), b.Life())
		}
	}
}

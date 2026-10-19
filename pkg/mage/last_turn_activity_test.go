package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestLastTurnActivityArchive(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	check := func(game *Game, player uuid.UUID, cast, entry bool) {
		t.Helper()
		if game.PlayerCastSpellDuringLastTurn(player) != cast || game.PlayerPutNontokenPermanentDuringLastTurn(player) != entry {
			t.Fatalf("activity for %v: cast=%v entry=%v", player, game.PlayerCastSpellDuringLastTurn(player), game.PlayerPutNontokenPermanentDuringLastTurn(player))
		}
	}
	check(g, a.PlayerID(), false, false)
	g.FireEvent(core.GameEvent{Type: core.EvtSpellCast, PlayerID: a.PlayerID()})
	g.FireEvent(core.GameEvent{Type: core.EvtSpellCast, PlayerID: b.PlayerID()})
	token := g.PutOnBattlefieldBy(NewToken("Token", 1, 1, []core.CardType{core.TypeCreature}, nil), a.PlayerID(), a.PlayerID())
	g.RemoveFromBattlefield(token)
	card := NewCreature("Entry", "", 1, 2)
	card.SetOwner(b.PlayerID())
	p := g.PutOnBattlefieldBy(card, a.PlayerID(), b.PlayerID())
	g.RemoveFromBattlefield(p)
	check(g, a.PlayerID(), false, false)
	g.trackers.Turn.ResetForNewTurn(a.PlayerID())
	check(g, a.PlayerID(), true, false)
	check(g, b.PlayerID(), false, false)
	g.trackers.Turn.ResetForNewTurn(b.PlayerID())
	check(g, b.PlayerID(), false, false)
	clone := g.Clone()
	g.SetActivePlayerIndex(1)
	g.PutOnBattlefieldBy(NewLand("Entry land"), a.PlayerID(), b.PlayerID())
	g.FireEvent(core.GameEvent{Type: core.EvtSpellCast, PlayerID: b.PlayerID()})
	currentClone := g.Clone()
	g.trackers.Turn.ResetForNewTurn(b.PlayerID())
	check(g, b.PlayerID(), true, true)
	check(clone, b.PlayerID(), false, false)
	currentClone.trackers.Turn.ResetForNewTurn(b.PlayerID())
	check(currentClone, b.PlayerID(), true, true)
	g.trackers.Turn.ResetForNewTurn(a.PlayerID())
	check(g, a.PlayerID(), false, false)
	check(g, b.PlayerID(), true, true)
	g.trackers.Turn.ResetForNewTurn(b.PlayerID())
	check(g, b.PlayerID(), false, false)
	check(currentClone, b.PlayerID(), true, true)
}

func TestLastTurnActivityConditions(t *testing.T) {
	for _, entry := range []bool{false, true} {
		a, b := NewBasePlayer("A"), NewBasePlayer("B")
		g := NewGame(a, b)
		g.SetActivePlayerIndex(0)
		g.SetStep(core.PrecombatMain)
		condition := CastSpellDuringLastTurn(You)
		if entry {
			condition = PutNontokenPermanentDuringLastTurn(You)
		}
		ctx := StateContext{ControllerID: a.PlayerID()}
		if condition.checkState(g, ctx) {
			t.Fatal("empty history")
		}
		if entry {
			g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
		} else {
			g.FireEvent(core.GameEvent{Type: core.EvtSpellCast, PlayerID: a.PlayerID()})
		}
		if condition.checkState(g, ctx) {
			t.Fatal("current activity replaced last turn")
		}
		g.RunStepWithPriority(core.Cleanup)
		if !condition.checkState(g, ctx) {
			t.Fatal("cleanup lost activity")
		}
		g.SetActivePlayerIndex(1)
		g.RunStepWithPriority(core.Cleanup)
		if !condition.checkState(g, ctx) {
			t.Fatal("opponent cleanup lost activity")
		}
		g.SetActivePlayerIndex(0)
		g.RunStepWithPriority(core.Cleanup)
		if condition.checkState(g, ctx) {
			t.Fatal("empty extra turn did not replace history")
		}
	}
	for _, condition := range []StateCondition{CastSpellDuringLastTurn(nil), PutNontokenPermanentDuringLastTurn(nil), CastSpellDuringLastTurn(FromPlayer(BindPlayer("missing")))} {
		card := NewCreature("Invalid", "", 1, 1, WithAbility(Static(CantAttackEachIf(MatchingPermanents(IsCreature), condition))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid condition accepted")
		}
	}
}

func TestLastTurnActivityAttackPaths(t *testing.T) {
	for _, path := range []string{"native", "explicit"} {
		t.Run(path, func(t *testing.T) {
			a := NewBasePlayer("A")
			b := &playerRestrictionAttacker{BasePlayer: NewBasePlayer("B")}
			g := NewGame(a, b)
			condition := NotCondition(AnyCondition(CastSpellDuringLastTurn(CombatDefendingPlayer()), PutNontokenPermanentDuringLastTurn(CombatDefendingPlayer())))
			source := g.PutOnBattlefield(NewEnchantment("Rule", "", WithAbility(Static(CantAttackEachIf(MatchingPermanents(IsCreature), condition)))), a.PlayerID())
			p := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 4), b.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			b.attackers = []uuid.UUID{p.ID()}
			g.SetActivePlayerIndex(1)
			g.SetStep(core.DeclareAttackers)
			if path == "native" {
				g.doDeclareAttackers()
			} else {
				g.ExecuteAttackers(b.PlayerID(), b.attackers)
			}
			if g.GetCombat().IsAttacking(p.ID()) || g.FindPermanent(p.ID()).Tapped {
				t.Fatal("prohibited declaration changed combat")
			}
			g.PhaseOut(source.ID())
			if !g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("phased source still restricts")
			}
			g.PhaseIn([]uuid.UUID{source.ID()})
			if g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("phase cycle lost restriction")
			}
			g.FireEvent(core.GameEvent{Type: core.EvtSpellCast, PlayerID: a.PlayerID()})
			g.trackers.Turn.ResetForNewTurn(a.PlayerID())
			if !g.FindPermanent(p.ID()).CanDeclareAsAttacker(g) {
				t.Fatal("history did not permit attack")
			}
			if path == "native" {
				g.doDeclareAttackers()
			} else {
				g.ExecuteAttackers(b.PlayerID(), b.attackers)
			}
			if !g.GetCombat().IsAttacking(p.ID()) {
				t.Fatal("permitted attack failed")
			}
			g.trackers.Turn.ResetForNewTurn(a.PlayerID())
			if !g.GetCombat().IsAttacking(p.ID()) {
				t.Fatal("later restriction removed attacker")
			}
		})
	}
}

func TestLastTurnActivityEntryPaths(t *testing.T) {
	for _, mode := range []string{"default", "actor", "graveyard", "blocked", "canceled", "phase", "control", "token", "payment-trial"} {
		t.Run(mode, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewLand("Entry")
			card.SetOwner(b.PlayerID())
			actor := a.PlayerID()
			want := true
			switch mode {
			case "blocked":
				g.effects.Rules.SetLandsCantEnter(true)
				if g.PutOnBattlefieldBy(card, b.PlayerID(), actor) != nil {
					t.Fatal("land entry was permitted")
				}
				want = false
			case "canceled":
				card.AddAbility(&graveyardRejectEntry{})
				b.AddToGraveyard(card)
				if returnGraveyardCardToBattlefieldWithActor(g, card.ID(), b.PlayerID(), actor, false) != nil {
					t.Fatal("entry was permitted")
				}
				want = false
			case "graveyard":
				b.AddToGraveyard(card)
				p := returnGraveyardCardToBattlefieldWithActor(g, card.ID(), b.PlayerID(), actor, false)
				if p == nil || p.ControllerID() != b.PlayerID() {
					t.Fatal("graveyard entry failed")
				}
				g.RemoveFromBattlefield(p)
			case "default":
				g.PutOnBattlefield(card, actor)
			case "token":
				g.PutOnBattlefieldBy(NewToken("Token", 1, 1, []core.CardType{core.TypeCreature}, nil), b.PlayerID(), actor)
				want = false
			case "payment-trial":
				trial := cloneGameForCostPayment(g)
				trial.PutOnBattlefieldBy(card, b.PlayerID(), actor)
				trial.trackers.Turn.ResetForNewTurn(actor)
				if !trial.PlayerPutNontokenPermanentDuringLastTurn(actor) {
					t.Fatal("trial did not retain independent history")
				}
				want = false
			default:
				p := g.PutOnBattlefieldBy(card, b.PlayerID(), actor)
				if mode == "phase" {
					g.PhaseIn(g.PhaseOut(p.ID()))
				}
				if mode == "control" {
					g.AddControlEffect(ControlEffectSpec{SourceID: actor, TargetID: p.ID(), ControllerID: actor, Duration: core.Indefinite})
				}
			}
			g.trackers.Turn.ResetForNewTurn(actor)
			if g.PlayerPutNontokenPermanentDuringLastTurn(actor) != want || g.PlayerPutNontokenPermanentDuringLastTurn(b.PlayerID()) {
				t.Fatal("entry history used the wrong actor or result")
			}
		})
	}
}

func TestLastTurnActivityCleanupAndSkippedTurn(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.FireEvent(core.GameEvent{Type: core.EvtSpellCast, PlayerID: a.PlayerID()})
	g.GetStack().Push(&StackObject{ID: uuid.New(), Controller: a.PlayerID()})
	if !g.doCleanupActions() || g.PlayerCastSpellDuringLastTurn(a.PlayerID()) {
		t.Fatal("cleanup priority archived history early")
	}
	g.GetStack().Pop()
	g.PutOnBattlefield(NewLand("Cleanup entry"), a.PlayerID())
	if g.doCleanupActions() || !g.PlayerCastSpellDuringLastTurn(a.PlayerID()) || !g.PlayerPutNontokenPermanentDuringLastTurn(a.PlayerID()) {
		t.Fatal("final cleanup lost activity")
	}
	g.SkipNextTurnFor(a.PlayerID())
	g.AdvanceToNextTurn()
	g.RunStepWithPriority(core.Cleanup)
	g.AdvanceToNextTurn()
	if !g.turns.ConsumeTurnSkip(a.PlayerID()) {
		t.Fatal("turn was not skipped")
	}
	g.AdvanceToNextTurn()
	if !g.PlayerCastSpellDuringLastTurn(a.PlayerID()) || !g.PlayerPutNontokenPermanentDuringLastTurn(a.PlayerID()) {
		t.Fatal("skipped turn replaced completed-turn history")
	}
	g.GrantExtraTurn(b.PlayerID())
	g.RunStepWithPriority(core.Cleanup)
	g.AdvanceToNextTurn()
	if g.ActivePlayerObj().PlayerID() != b.PlayerID() {
		t.Fatal("extra turn was not selected")
	}
	g.RunStepWithPriority(core.Cleanup)
	if !g.PlayerCastSpellDuringLastTurn(a.PlayerID()) {
		t.Fatal("other player's extra turn changed archive")
	}
}

func TestLastTurnActivityCasting(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	unpaid := NewInstant("Unpaid", "{U}", WithAbility(Spell(GainLifeFor(You, Fixed(1)))))
	unpaid.SetOwner(a.PlayerID())
	a.AddToHand(unpaid)
	if err := g.CastSpellByID(a.PlayerID(), unpaid.ID(), nil, 0); err == nil {
		t.Fatal("unpaid spell was cast")
	}
	g.RunStepWithPriority(core.Cleanup)
	if g.PlayerCastSpellDuringLastTurn(a.PlayerID()) {
		t.Fatal("failed cast entered history")
	}
	a.ManaPool().Add(core.Blue, 1)
	g.SetStep(core.PrecombatMain)
	if err := g.CastSpellByID(a.PlayerID(), unpaid.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.CopySpellOnStack(unpaid.ID(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.CounterSpellOnStack(unpaid.ID())
	g.ResolveStack()
	g.RunStepWithPriority(core.Cleanup)
	if !g.PlayerCastSpellDuringLastTurn(a.PlayerID()) {
		t.Fatal("countered cast did not count")
	}
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Cleanup)
	if g.PlayerCastSpellDuringLastTurn(b.PlayerID()) {
		t.Fatal("stack copy counted as a cast")
	}
	for _, player := range []PlayerRef{You, ActivePlayer, CombatDefendingPlayer()} {
		ctx := StateContext{ControllerID: a.PlayerID()}
		g.SetStep(core.DeclareAttackers)
		g.SetActivePlayerIndex(0)
		want := true
		if _, ok := player.(combatDefendingPlayerRef); ok {
			want = false
		}
		if CastSpellDuringLastTurn(player).checkState(g, ctx) != want {
			t.Fatal("typed player did not select its archive")
		}
		card := NewCreature("Valid", "", 1, 1, WithAbility(Static(CantAttackEachIf(MatchingPermanents(IsCreature), CastSpellDuringLastTurn(player)))))
		if err := ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
}

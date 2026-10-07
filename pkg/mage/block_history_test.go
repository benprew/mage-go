package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestBlockHistoryLifecycle(t *testing.T) {
	for _, change := range []string{"none", "remove-combat", "new-combat", "control", "blocker-leaves", "blocker-returns", "blocker-phase", "attacker-phase", "attacker-phase-cycle", "attacker-returns"} {
		t.Run(change, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			card := NewCreature("Attacker", "", 2, 5)
			card.SetOwner(a.PlayerID())
			attacker := g.PutOnBattlefield(card, a.PlayerID())
			wallCard := NewCreature("Wall", "", 0, 5, WithSubTypes("Wall"))
			wallCard.SetOwner(b.PlayerID())
			wall := g.PutOnBattlefield(wallCard, b.PlayerID())
			ctx := &EffectContext{Game: g, SourceID: wall.ID(), Controller: b.PlayerID(), sourceCaptured: true, sourceIncarnation: wall.incarnationID}
			saved := BindPermanent("attacker")
			g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{saved: {attacker.ID(), attacker.incarnationID}}
			g.GetCombat().AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
			g.ExecuteBlockers([]BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}})
			set := BlockedByThisTurn(This)
			if len(set.resolvePermanents(ctx)) != 1 {
				t.Fatal("legal block missing")
			}
			switch change {
			case "remove-combat":
				g.GetCombat().RemoveFromCombat(wall.ID())
			case "new-combat":
				g.GetCombat().Reset()
				g.FireEvent(core.GameEvent{Type: core.EvtBeginCombat})
			case "control":
				g.AddControlEffect(ControlEffectSpec{SourceID: wall.ID(), TargetID: attacker.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "blocker-leaves", "blocker-returns":
				g.BouncePermanentToHand(g.FindPermanent(wall.ID()))
				if change == "blocker-returns" {
					b.RemoveFromHand(wallCard.ID())
					g.PutOnBattlefield(wallCard, b.PlayerID())
				}
			case "blocker-phase":
				g.PhaseOut(wall.ID())
			case "attacker-phase":
				g.PhaseOut(attacker.ID())
			case "attacker-phase-cycle":
				g.PhaseIn(g.PhaseOut(attacker.ID()))
			case "attacker-returns":
				g.BouncePermanentToHand(g.FindPermanent(attacker.ID()))
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			}
			want := 1
			if change == "attacker-phase" || change == "attacker-returns" {
				want = 0
			}
			if len(set.resolvePermanents(ctx)) != want {
				t.Fatal("history followed another object or current combat")
			}
			player, ok := ControllerWhenLastBlockedThisTurn(From(saved), This).resolvePlayer(ctx)
			if !ok || player != a.PlayerID() {
				t.Fatal("lost event controller")
			}
			clone := g.Clone()
			clone.trackers.Turn.ResetForNewTurn(a.PlayerID())
			clonedCtx := *ctx
			clonedCtx.Game = clone
			if len(set.resolvePermanents(&clonedCtx)) != 0 || len(set.resolvePermanents(ctx)) != want {
				t.Fatal("clone cleanup changed original")
			}
			if _, ok := ControllerWhenLastBlockedThisTurn(From(saved), This).resolvePlayer(&clonedCtx); ok {
				t.Fatal("cleanup kept controller history")
			}
		})
	}
}

func TestBlockHistoryLegalPairsAndLastController(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	wall := g.PutOnBattlefield(NewCreature("Wall", "", 0, 5, WithKeyword(core.CanBlockAny)), b.PlayerID())
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 5), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 5), a.PlayerID())
	for _, p := range []*Permanent{first, second} {
		g.GetCombat().AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
	}
	ctx := &EffectContext{Game: g, SourceID: wall.ID(), sourceCaptured: true, sourceIncarnation: wall.incarnationID}
	set := BlockedByThisTurn(This)
	proposal := []BlockAssignment{{wall.ID(), first.ID()}, {wall.ID(), second.ID()}}
	g.CompleteBlockAssignments(b.PlayerID(), proposal)
	if len(set.resolvePermanents(ctx)) != 0 {
		t.Fatal("pure proposal query recorded history")
	}
	g.ExecuteBlockers(proposal)
	if len(set.resolvePermanents(ctx)) != 2 {
		t.Fatal("multi-block history missing")
	}
	g.GetCombat().Reset()
	g.AddControlEffect(ControlEffectSpec{SourceID: wall.ID(), TargetID: first.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.AddControlEffect(ControlEffectSpec{SourceID: wall.ID(), TargetID: wall.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
	g.GetCombat().AddAttackerIncarnation(first.ID(), first.incarnationID, a.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{wall.ID(), first.ID()}})
	target := BindPermanent("first")
	g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{target: {first.ID(), first.incarnationID}}
	player, ok := ControllerWhenLastBlockedThisTurn(From(target), This).resolvePlayer(ctx)
	if !ok || player != b.PlayerID() {
		t.Fatal("last block controller did not update")
	}
	g.GetCombat().Reset()
	third := g.PutOnBattlefield(NewCreature("Third", "", 2, 5, WithStaticAbility(SourceCantBeBlockedByFewerThan(3))), b.PlayerID())
	g.GetCombat().AddAttackerIncarnation(third.ID(), third.incarnationID, a.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{wall.ID(), third.ID()}})
	if len(set.resolvePermanents(ctx)) != 2 {
		t.Fatal("illegal block entered history")
	}
	entered := g.PutOnBattlefieldBlocking(NewCreature("Entered", "", 0, 4), a.PlayerID(), third.ID())
	enteredCtx := &EffectContext{Game: g, SourceID: entered.ID()}
	if len(BlockedByThisTurn(This).resolvePermanents(enteredCtx)) != 0 {
		t.Fatal("entered blocker counted as having blocked")
	}
}

func TestBlockHistoryDeclarations(t *testing.T) {
	target := DeclarePermanentTarget("wall", AnyPermanent)
	valid := NewInstant("Valid", "", WithAbility(Spell(DestroyEach(BlockedByThisTurn(target)), WithTargets(target))))
	if err := ValidateCardDefinition(valid.Name(), valid); err != nil {
		t.Fatal(err)
	}
	for _, effect := range []ResolvingEffect{DestroyEach(BlockedByThisTurn(nil)), DestroyEach(BlockedByThisTurn(target)), GainLifeFor(ControllerWhenLastBlockedThisTurn(nil, This), Fixed(1)), GainLifeFor(ControllerWhenLastBlockedThisTurn(This, nil), Fixed(1))} {
		card := NewCreature("Bad", "", 1, 1, WithAbility(Activate(Costs(), effect)))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid history declaration accepted")
		}
	}
}

func TestBlockHistoryBandsAndNormalDeclaration(t *testing.T) {
	for _, normal := range []bool{false, true} {
		t.Run(map[bool]string{false: "explicit", true: "normal"}[normal], func(t *testing.T) {
			a := NewBasePlayer("A")
			b := &fixedBlockPlayer{BasePlayer: NewBasePlayer("B")}
			g := NewGame(a, b)
			first := g.PutOnBattlefield(NewCreature("Band member", "", 2, 5, WithKeyword(core.Banding)), a.PlayerID())
			second := g.PutOnBattlefield(NewCreature("Other member", "", 2, 5), a.PlayerID())
			wall := g.PutOnBattlefield(NewCreature("Wall", "", 0, 5), b.PlayerID())
			g.GetCombat().AddAttackerIncarnation(first.ID(), first.incarnationID, b.PlayerID())
			g.GetCombat().AddAttackerIncarnation(second.ID(), second.incarnationID, b.PlayerID())
			g.GetCombat().AddBand([]uuid.UUID{first.ID(), second.ID()})
			b.assignments = []BlockAssignment{{wall.ID(), first.ID()}}
			if normal {
				g.doDeclareBlockers()
			} else {
				g.ExecuteBlockers(b.assignments)
			}
			ctx := &EffectContext{Game: g, SourceID: wall.ID()}
			if len(BlockedByThisTurn(This).resolvePermanents(ctx)) != 2 {
				t.Fatal("band members missing from history")
			}
			clone := g.Clone()
			g.GetCombat().Reset()
			third := g.PutOnBattlefield(NewCreature("Later attacker", "", 2, 5), a.PlayerID())
			g.GetCombat().AddAttackerIncarnation(third.ID(), third.incarnationID, b.PlayerID())
			g.ExecuteBlockers([]BlockAssignment{{wall.ID(), third.ID()}})
			clonedCtx := *ctx
			clonedCtx.Game = clone
			if len(BlockedByThisTurn(This).resolvePermanents(ctx)) != 3 || len(BlockedByThisTurn(This).resolvePermanents(&clonedCtx)) != 2 {
				t.Fatal("new combat shared clone map")
			}
		})
	}
}

func TestBlockHistoryCleanupPriority(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	wall := g.PutOnBattlefield(NewCreature("Wall", "", 0, 5), b.PlayerID())
	attacker := g.PutOnBattlefield(NewCreature("Attacker", "", 2, 5), a.PlayerID())
	g.GetCombat().AddAttackerIncarnation(attacker.ID(), attacker.incarnationID, b.PlayerID())
	g.ExecuteBlockers([]BlockAssignment{{wall.ID(), attacker.ID()}})
	ctx := &EffectContext{Game: g, SourceID: wall.ID()}
	set := BlockedByThisTurn(This)
	g.GetStack().Push(&StackObject{ID: uuid.New(), Controller: a.PlayerID()})
	if !g.doCleanupActions() || len(set.resolvePermanents(ctx)) != 1 {
		t.Fatal("cleanup priority cleared history")
	}
	g.GetStack().Pop()
	if g.doCleanupActions() || len(set.resolvePermanents(ctx)) != 0 {
		t.Fatal("final cleanup retained history")
	}
}

func TestBlockHistorySeparateObjects(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	first := g.PutOnBattlefield(NewCreature("First", "", 2, 5), a.PlayerID())
	second := g.PutOnBattlefield(NewCreature("Second", "", 2, 5), a.PlayerID())
	wallCard := NewCreature("Wall", "", 0, 5)
	wallCard.SetOwner(b.PlayerID())
	wall := g.PutOnBattlefield(wallCard, b.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other wall", "", 0, 5), b.PlayerID())
	for _, p := range []*Permanent{first, second} {
		g.GetCombat().AddAttackerIncarnation(p.ID(), p.incarnationID, b.PlayerID())
	}
	g.ExecuteBlockers([]BlockAssignment{{wall.ID(), first.ID()}, {other.ID(), second.ID()}})
	ctx := &EffectContext{Game: g, SourceID: wall.ID(), sourceCaptured: true, sourceIncarnation: wall.incarnationID}
	original := BlockedByThisTurn(This).resolvePermanents(ctx)
	otherCtx := &EffectContext{Game: g, SourceID: other.ID()}
	if len(original) != 1 || original[0].ID() != first.ID() {
		t.Fatal("first blocker selected other history")
	}
	selected := BlockedByThisTurn(This).resolvePermanents(otherCtx)
	if len(selected) != 1 || selected[0].ID() != second.ID() {
		t.Fatal("second blocker selected other history")
	}
	g.BouncePermanentToHand(wall)
	b.RemoveFromHand(wallCard.ID())
	returned := g.PutOnBattlefield(wallCard, b.PlayerID())
	g.zones.ClearLKI()
	if len(BlockedByThisTurn(This).resolvePermanents(ctx)) != 1 {
		t.Fatal("saved identity required current LKI")
	}
	currentCtx := &EffectContext{Game: g, SourceID: returned.ID()}
	if len(BlockedByThisTurn(This).resolvePermanents(currentCtx)) != 0 {
		t.Fatal("returned wall inherited history")
	}
}

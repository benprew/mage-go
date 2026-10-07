package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONAttackHistoryBerserk(t *testing.T) {
	for _, change := range []string{"attack", "none", "attacking-only", "return", "phase", "phase-cycle", "control", "shroud", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			victim := mage.NewCreature("Victim", "", 3, 9)
			p := g.PutOnBattlefield(victim, a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			spell := repositoryCard(t, "LEA", "Berserk")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, 0); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("unpaid casting changed state")
			}
			a.ManaPool().Add(core.Green, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			p = g.FindPermanent(p.ID())
			if p.CurrentPower(g) != 6 || p.CurrentToughness(g) != 9 || !p.HasKeyword(core.Trample) {
				t.Fatal("wrong resolving bonus")
			}
			if change != "none" && change != "attacking-only" {
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
			} else if change == "attacking-only" {
				g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
			}
			g.GetCombat().RemoveFromCombat(p.ID())
			switch change {
			case "return":
				g.RemoveFromBattlefield(p)
				g.PutOnBattlefield(victim, a.PlayerID())
			case "phase":
				g.PhaseOut(p.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "shroud":
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("next end step did not trigger")
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("trigger copy failed")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if g.FindPermanentIncludingPhased(p.ID()) == nil {
				t.Fatal("clone changed original")
			}
			g.ResolveStack()
			destroyed := change != "none" && change != "attacking-only" && change != "return" && change != "phase"
			for _, game := range []*mage.Game{g, clone} {
				if (game.FindPermanentIncludingPhased(p.ID()) == nil) != destroyed {
					t.Fatalf("destroyed=%v", destroyed)
				}
			}
		})
	}
}

func TestRONAttackHistoryFireAndBrimstone(t *testing.T) {
	for _, change := range []string{"opponent", "self", "copy", "cleanup"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			spell := repositoryCard(t, "DRK", "Fire and Brimstone")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.White, 2)
			a.ManaPool().Add(core.Colorless, 3)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{b.PlayerID()}, 0); err == nil || a.ManaPool().TotalMana() != 5 || g.StackSize() != 0 {
				t.Fatal("player without an attack qualified")
			}
			attackerPlayer := b
			if change == "self" {
				attackerPlayer = a
			}
			p := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 5), attackerPlayer.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(attackerPlayer.PlayerID(), []uuid.UUID{p.ID()})
			g.RemoveFromBattlefield(p)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{attackerPlayer.PlayerID()}, 0); err != nil {
				t.Fatal(err)
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("spell copy failed")
			}
			if change == "cleanup" {
				g.RunStepWithPriority(core.Cleanup)
				if g.PlayerAttackedThisTurn(attackerPlayer.PlayerID()) {
					t.Fatal("native cleanup retained history")
				}
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveStack()
				wantA, wantB := 16, 16
				if change == "self" {
					wantA, wantB = 12, 20
				}
				if change == "copy" {
					wantA, wantB = 16, 8
				}
				if game.GetPlayer(a.PlayerID()).Life() != wantA || game.GetPlayer(b.PlayerID()).Life() != wantB {
					t.Fatalf("life=%d/%d want=%d/%d", game.GetPlayer(a.PlayerID()).Life(), game.GetPlayer(b.PlayerID()).Life(), wantA, wantB)
				}
			}
		})
	}
}

func TestRONAttackHistoryInvalidAndInventory(t *testing.T) {
	for _, body := range []string{
		`Targeted([TargetPlayer(AttackedThisTurn())],ActionList([GainLife(Integer(1))]))`,
		`Targeted([TargetPlayer(DidntAttackThisTurn)],ActionList([GainLife(Integer(1))]))`,
		`ActionList([If(PermanentPassesFilter(Ref_TargetPermanent,AttackedThisTurn),[GainLife(Integer(1))])])`,
		`Targeted([TargetPlayer(AnyPlayer)],ActionList([If(PermanentPassesFilter(Ref_TargetPlayer,AttackedThisTurn),[GainLife(Integer(1))])]))`,
		`Targeted([TargetPermanent(AttackedThisTurn(Integer(1)))],ActionList([TapPermanent(Ref_TargetPermanent)]))`,
		`Targeted([TargetPermanent(AnyPermanent)],ActionList([If(PermanentPassesFilter(Ref_TargetPermanent,DidntAttackThisTurn()),[GainLife(Integer(1))])]))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(` + body + `)])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	src := `Card(Name:"Roles",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPlayer(AttackedThisTurn)],ActionList([PlayerAction(Ref_TargetPlayer,GainLife(Integer(1)))])))])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range report.Signatures {
		if entry.Name == "AttackedThisTurn" {
			found = entry.Role == "player_filter"
		}
	}
	if !found || report.Compiled != 1 {
		t.Fatal("lost typed history inventory")
	}
}

func TestRONAttackHistoryBerserkTimingAndLegality(t *testing.T) {
	for _, change := range []string{"late-cast", "illegal-target", "power-change", "attack-after-trigger", "indestructible", "regenerate"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(mage.NewCreature("Subject", "", 3, 8), a.PlayerID())
			p.RevokeBaseAttr(core.AttrSummonSick)
			spell := repositoryCard(t, "LEA", "Berserk")
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.Green, 1)
			if change == "late-cast" {
				g.SetStep(core.CombatDamage)
			}
			err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, 0)
			if change == "late-cast" {
				if err == nil || g.StackSize() != 0 || a.ManaPool().TotalMana() != 1 {
					t.Fatal("late casting spent resources")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if change == "illegal-target" {
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Shroud)
			}
			if change == "power-change" {
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(2), mage.Fixed(0))), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveStack()
			if change == "illegal-target" {
				g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
				g.PutTriggersOnStack()
				if g.StackSize() != 0 || g.FindPermanent(p.ID()).CurrentPower(g) != 3 {
					t.Fatal("illegal spell made a delayed effect")
				}
				return
			}
			wantPower := 6
			if change == "power-change" {
				wantPower = 10
			}
			if g.FindPermanent(p.ID()).CurrentPower(g) != wantPower {
				t.Fatal("power was not captured at resolution")
			}
			if change != "attack-after-trigger" {
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
			}
			if change == "indestructible" {
				g.MutablePermanent(p.ID()).GrantBaseAttr(core.Indestructible)
			}
			if change == "regenerate" {
				g.AddRegenerationShield(p.ID())
			}
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("condition was checked at trigger time")
			}
			if change == "attack-after-trigger" {
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
			}
			g.ResolveStack()
			if change == "indestructible" || change == "regenerate" {
				if g.FindPermanent(p.ID()) == nil {
					t.Fatal("destruction bypassed protection")
				}
			} else if g.FindPermanent(p.ID()) != nil {
				t.Fatal("delayed destruction failed")
			}
		})
	}
}

func TestRONAttackHistoryBerserkCopyAndExpiry(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 6), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 3, 6), b.PlayerID())
	spell := repositoryCard(t, "LEA", "Berserk")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{first.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.PhaseOut(first.ID())
	copySpell := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if copySpell == nil || len(copySpell.Targets) != 1 || copySpell.Targets[0] != second.ID() {
		t.Fatal("copy did not retarget")
	}
	g.PhaseIn([]uuid.UUID{first.ID()})
	g.ResolveStack()
	if g.FindPermanent(first.ID()).CurrentPower(g) != 4 || g.FindPermanent(second.ID()).CurrentPower(g) != 6 {
		t.Fatal("copy shared its target binding")
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("copies did not create separate delayed triggers")
	}
	g.ResolveStack()
	g.RunStepWithPriority(core.Cleanup)
	for _, p := range []*mage.Permanent{g.FindPermanent(first.ID()), g.FindPermanent(second.ID())} {
		if p.HasKeyword(core.Trample) {
			t.Fatal("trample outlasted cleanup")
		}
	}
	if g.FindPermanent(first.ID()).CurrentPower(g) != 2 || g.FindPermanent(second.ID()).CurrentPower(g) != 3 {
		t.Fatal("bonus outlasted cleanup")
	}
	g.MutablePermanent(first.ID()).RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID()})
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	if g.StackSize() != 0 || g.FindPermanent(first.ID()) == nil {
		t.Fatal("one-shot trigger fired a second time")
	}
}

func TestRONAttackHistoryFireAndBrimstoneBatch(t *testing.T) {
	for _, self := range []bool{false, true} {
		g, a, b := ronGame()
		attackerPlayer := b
		if self {
			attackerPlayer = a
		}
		p := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 4), attackerPlayer.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		g.ExecuteAttackers(attackerPlayer.PlayerID(), []uuid.UUID{p.ID()})
		g.AddPreventionShield(attackerPlayer.PlayerID(), 2)
		spell := repositoryCard(t, "DRK", "Fire and Brimstone")
		spell.SetOwner(a.PlayerID())
		a.AddToHand(spell)
		a.ManaPool().Add(core.White, 2)
		a.ManaPool().Add(core.Colorless, 3)
		if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{attackerPlayer.PlayerID()}, 0); err != nil {
			t.Fatal(err)
		}
		wantA, wantB := 16, 18
		if self {
			wantA, wantB = 14, 20
		}
		callbacks := 0
		g.SetOnDamageDealt(func(_, _ string, _ int, combat bool) {
			callbacks++
			if a.Life() != wantA || b.Life() != wantB || combat {
				t.Error("damage callback observed an incomplete batch")
			}
		})
		g.ResolveStack()
		wantCallbacks := 2
		if self {
			wantCallbacks = 1
		}
		if callbacks != wantCallbacks || a.Life() != wantA || b.Life() != wantB {
			t.Fatal("prevention changed independent packets")
		}
		if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != spell.ID() || a.ManaPool().TotalMana() != 0 {
			t.Fatal("wrong casting payment or spell destination")
		}
	}
}

func TestRONAttackHistoryFilterContextsAndInstances(t *testing.T) {
	for _, filter := range []string{"AttackedThisTurn", "DidntAttackThisTurn"} {
		g, a, b := ronGame()
		attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 4), a.PlayerID())
		idle := g.PutOnBattlefield(mage.NewCreature("Idle", "", 2, 4), b.PlayerID())
		attacker.RevokeBaseAttr(core.AttrSummonSick)
		g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
		g.RemoveFromCombat(attacker.ID())
		rule := `SpellActions(Targeted([TargetPermanent(` + filter + `)],ActionList([If(PermanentPassesFilter(Ref_TargetPermanent,And([IsCardtype(Creature),` + filter + `])),[TapPermanent(Ref_TargetPermanent)])])))`
		factory := permanentScopeFactory(t, "Instant", rule)
		first, second := factory(), factory()
		firstAction := first.Abilities()[0].(*mage.ActionDefinition)
		secondAction := second.Abilities()[0].(*mage.ActionDefinition)
		if firstAction.Targets()[0] == secondAction.Targets()[0] {
			t.Fatal("factories shared a declaration")
		}
		for _, card := range []mage.Card{first, second} {
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
		}
		good, bad := attacker, idle
		if filter == "DidntAttackThisTurn" {
			good, bad = idle, attacker
		}
		first.SetOwner(a.PlayerID())
		a.AddToHand(first)
		if err := g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{bad.ID()}, 0); err == nil {
			t.Fatal("wrong history target accepted")
		}
		if err := g.CastSpellByID(a.PlayerID(), first.ID(), []uuid.UUID{good.ID()}, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if !g.FindPermanent(good.ID()).Tapped {
			t.Fatal("resolving branch lost its history")
		}
		collection := permanentScopeFactory(t, "Instant", `SpellActions(ActionList([UntapEachPermanent(And([IsCardtype(Creature),`+filter+`]))]))`)()
		collection.SetOwner(b.PlayerID())
		b.AddToHand(collection)
		if err := g.CastSpellByID(b.PlayerID(), collection.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if g.FindPermanent(good.ID()).Tapped {
			t.Fatal("collection lost its history filter")
		}
	}
}

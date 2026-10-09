package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNextCombatPreventionForcefield(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			g, base, b := ronGame()
			a := &ronPermanentChoicePlayer{Player: base, choose: func(p []*mage.Permanent) *mage.Permanent { return p[0] }}
			g.SetPlayerAt(0, a)
			source := g.PutOnBattlefield(repositoryCard(t, set, "Forcefield"), a.PlayerID())
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 4, 10, mage.WithKeyword(core.Shroud), mage.WithKeyword(core.DoubleStrike)), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other attacker", "", 0, 10), b.PlayerID())
			other.RevokeBaseAttr(core.AttrSummonSick)
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID(), other.ID()})
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || a.calls != 0 || g.StackSize() != 0 {
				t.Fatal("unpaid activation changed state")
			}
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if a.calls != 0 {
				t.Fatal("attacker was unblocked before blockers declaration")
			}
			g.ExecuteBlockers(nil)
			if !g.IsAttackingIncarnation(attacker.ID(), g.CaptureDamageSource(attacker.ID()).Incarnation()) {
				t.Fatal("fixture did not declare attacker")
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if a.calls != 0 {
				t.Fatal("choice occurred at announcement")
			}
			pending := g.Clone()
			pending.SetPlayerAt(0, &ronPermanentChoicePlayer{Player: pending.GetPlayer(a.PlayerID()), choose: func(p []*mage.Permanent) *mage.Permanent { return p[0] }})
			g.ResolveStack()
			if a.calls != 1 || a.ManaPool().TotalMana() != 0 {
				t.Fatalf("wrong choice or payment: calls=%d mana=%d", a.calls, a.ManaPool().TotalMana())
			}
			g.DealDamageToPlayer(a, 2, attacker.ID())
			g.ExecuteCombatDamage()
			if a.Life() != 13 {
				t.Fatalf("life=%d want=13", a.Life())
			}
			pending.ResolveStack()
			pending.ExecuteCombatDamage()
			if pending.GetPlayer(a.PlayerID()).Life() != 15 {
				t.Fatal("pending clone lost independent prevention")
			}
		})
	}
}

func TestRONNextCombatPreventionInvalid(t *testing.T) {
	const event = `NextTimeCombatDamageWouldBeDealtThisTurnByCreatureToRecipient(TheChosenPermanent,Player(You))`
	for _, body := range []string{
		`CreateFuturePreventDamage(` + event + `,[PreventAllButSomeOfThatDamage(Integer(1))])`,
		`ChooseAPermanent(IsCardtype(Creature)),CreateFuturePreventDamage(` + event + `,[PreventAllButSomeOfThatDamage])`,
		`ChooseAPermanent(IsCardtype(Creature)),CreateFuturePreventDamage(` + event + `,[PreventAllButSomeOfThatDamage(Integer(-1))])`,
		`ChooseAPermanent(IsCardtype(Creature)),CreateFuturePreventDamage(` + event + `,[PreventThatDamage])`,
		`ChooseAPermanent(IsCardtype(Creature)),CreateFuturePreventDamage(` + event + `,[])`,
		`ChooseAPermanent(IsCardtype(Creature)),CreateFuturePreventDamage(` + event + `,[PreventAllButSomeOfThatDamage(Integer(1)),PreventThatDamage])`,
		`ChooseAPermanent(IsCardtype(Creature)),CreateFuturePreventDamage(NextTimeCombatDamageWouldBeDealtThisTurnByCreatureToRecipient(TheChosenPermanent(),Player(You)),[PreventAllButSomeOfThatDamage(Integer(1))])`,
		`ChooseAPermanent(IsCardtype(Creature)),CreateFuturePreventDamage(NextTimeCombatDamageWouldBeDealtThisTurnByCreatureToRecipient(TheChosenPermanent,Permanent(ThisPermanent)),[PreventAllButSomeOfThatDamage(Integer(1))])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([`+body+`]))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONNextCombatPreventionLifecycle(t *testing.T) {
	for _, change := range []string{"phase-cycle", "return", "provider-departure", "blocked-later", "cleanup"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "LEB", "Forcefield"), a.PlayerID())
			card := mage.NewCreature("Attacker", "", 4, 10, mage.WithKeyword(core.Shroud))
			card.SetOwner(b.PlayerID())
			attacker := g.PutOnBattlefield(card, b.PlayerID())
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID()})
			g.ExecuteBlockers(nil)
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			old := g.CaptureDamageSource(attacker.ID())
			switch change {
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(attacker.ID()))
			case "return":
				g.BouncePermanentToHand(attacker)
				b.RemoveFromHand(card.ID())
				attacker = g.PutOnBattlefield(card, b.PlayerID())
				attacker.RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID()})
				g.ExecuteBlockers(nil)
			case "provider-departure":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			case "blocked-later":
				blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 10), a.PlayerID())
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}})
			case "cleanup":
				g.SetStep(core.Cleanup)
				g.RunStepWithPriority(core.Cleanup)
			}
			if change == "blocked-later" {
				if len(g.WaitingPreventionPermanentSources()) != 1 {
					t.Fatal("later blocked status removed saved source")
				}
				g.ExecuteCombatDamage()
				if a.Life() != 20 || len(g.WaitingPreventionPermanentSources()) != 1 {
					t.Fatal("damage to blocker consumed player shield")
				}
				return
			}
			if change == "phase-cycle" {
				g.UntapPermanent(g.FindPermanent(attacker.ID()))
				g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID()})
				g.ExecuteBlockers(nil)
			}
			g.ExecuteCombatDamage()
			want := 19
			if change == "return" || change == "cleanup" {
				want = 16
			}
			if a.Life() != want {
				t.Fatalf("life=%d want=%d old=%v", a.Life(), want, old.ID())
			}
		})
	}
}

func TestRONNextCombatPreventionInventory(t *testing.T) {
	const src = `Card(Name:"Shield",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([ChooseAPermanent(IsCardtype(Creature)),CreateFuturePreventDamage(NextTimeCombatDamageWouldBeDealtThisTurnByCreatureToRecipient(TheChosenPermanent,Player(You)),[PreventAllButSomeOfThatDamage(Integer(1))])]))])`
	report, err := InspectFS(fstest.MapFS{"shield.ron": &fstest.MapFile{Data: []byte(src)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory rejected", err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "TheChosenPermanent" {
			found = true
			if signature.Role != "permanent_reference" {
				t.Fatal("wrong source role", signature.Role)
			}
		}
		if signature.Name == "Integer" && signature.Role != "value" {
			t.Fatal("wrong amount role", signature.Role)
		}
	}
	if !found {
		t.Fatal("missing source signature")
	}
}

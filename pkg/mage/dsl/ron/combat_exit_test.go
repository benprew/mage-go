package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
	"github.com/benprew/mage-go/pkg/mage/gametest"
)

func TestRONCombatExitNative(t *testing.T) {
	for _, name := range []string{"Mijae Djinn", "Ydwen Efreet"} {
		for _, win := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%v", name, win), func(t *testing.T) {
				a, b := gametest.NewTestPlayer("A"), gametest.NewTestPlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(1)
				g.SetStep(core.PrecombatMain)
				card := repositoryCard(t, "ARN", name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				a.ManaPool().Add(core.Red, 2)
				if g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0) == nil || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 2 {
					t.Fatal("unpaid casting or rollback")
				}
				a.ManaPool().Add(core.Red, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				p := g.FindPermanent(card.ID())
				p.RevokeBaseAttr(core.AttrSummonSick)
				power, toughness := 6, 3
				if name == "Ydwen Efreet" {
					power, toughness = 3, 6
				}
				if p.CurrentPower(g) != power || p.CurrentToughness(g) != toughness {
					t.Fatal("wrong metadata")
				}
				attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 10), b.PlayerID())
				attacker.RevokeBaseAttr(core.AttrSummonSick)
				g.SetCoinFlipResults([]bool{win, !win})
				if name == "Mijae Djinn" {
					a.SetAttackers(1, []string{name})
				} else {
					g.SetActivePlayerIndex(1)
					b.SetAttackers(1, []string{"Attacker"})
					a.SetBlockers(1, map[string]string{name: "Attacker"})
				}
				g.RunStepWithPriority(core.DeclareAttackers)
				g.RunStepWithPriority(core.DeclareBlockers)
				if name == "Mijae Djinn" {
					if g.GetCombat().IsAttacking(p.ID()) != win || !g.FindPermanent(p.ID()).Tapped {
						t.Fatal("attack outcome")
					}
				} else if g.GetCombat().IsBlocking(p.ID()) != win || g.GetCombat().GroupFor(attacker.ID()).Blocked != win || g.FindPermanent(p.ID()).Tapped {
					t.Fatal("block outcome")
				}
				g.ExecuteCombatDamage()
				wantA, wantB := 20, 20
				if name == "Mijae Djinn" && win {
					wantB = 14
				}
				if name == "Ydwen Efreet" && !win {
					wantA = 18
				}
				if a.Life() != wantA || b.Life() != wantB {
					t.Fatal("wrong actual combat damage")
				}
				if g.FlipCoin(a.PlayerID()) == win {
					t.Fatal("wrong number of flips")
				}
				if name == "Ydwen Efreet" {
					if mage.CanBlock(g.FindPermanent(p.ID()), g.FindPermanent(attacker.ID()), g) != win {
						t.Fatal("blocking prohibition")
					}
					g.RunStepWithPriority(core.Cleanup)
					if !mage.CanBlock(g.FindPermanent(p.ID()), g.FindPermanent(attacker.ID()), g) {
						t.Fatal("restriction did not expire")
					}
				}
			})
		}
	}
}

func TestRONCombatExitInvalid(t *testing.T) {
	for _, action := range []string{`RemoveCreatureFromCombat`, `RemoveCreatureFromCombat()`, `RemoveCreatureFromCombat(You)`, `RemoveCreatureFromCombat(Ref_TargetPermanent)`, `RemoveCreatureFromCombat(ThisPermanent())`, `RemoveCreatureFromCombatAndUnblockBlockers(ThisPermanent,ThisPermanent)`, `CreatePermanentRuleEffectUntil(ThisPermanent,[CantBlock()],UntilEndOfTurn)`, `CreatePermanentRuleEffectUntil(ThisPermanent,[CantBlock],UntilEndOfCombat)`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[Activated(PayMana([]),ActionList([` + action + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-exit.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONCombatExitPendingIdentity(t *testing.T) {
	for _, name := range []string{"Mijae Djinn", "Ydwen Efreet"} {
		for _, change := range []string{"none", "two-blockers", "other-removed", "depart", "return", "phase", "phase-cycle", "control", "ability-removal", "copy", "clone"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "ARN", name)
				p := g.PutOnBattlefield(card, a.PlayerID())
				attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 10), b.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 10), a.PlayerID())
				event := core.EvtDeclaredAttacker
				if name == "Mijae Djinn" {
					g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
				} else {
					event = core.EvtDeclaredBlocker
					g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
					g.GetCombat().AddBlockerIncarnation(p.ID(), g.CaptureDamageSource(p.ID()).Incarnation(), attacker.ID())
				}
				if name == "Ydwen Efreet" && (change == "two-blockers" || change == "other-removed") {
					g.GetCombat().AddBlockerIncarnation(other.ID(), g.CaptureDamageSource(other.ID()).Incarnation(), attacker.ID())
					if change == "other-removed" {
						g.RemoveFromCombat(other.ID())
					}
				}
				g.FireEvent(core.GameEvent{Type: event, SourceID: p.ID(), PlayerID: a.PlayerID(), Flag: true})
				g.PutTriggersOnStack()
				if g.StackSize() != 1 {
					t.Fatal("missing trigger")
				}
				g.SetCoinFlipResults([]bool{false, false, true})
				switch change {
				case "depart", "return":
					g.BouncePermanentToHand(g.FindPermanent(p.ID()))
					a.RemoveFromHand(card.ID())
					if change == "return" {
						p = g.PutOnBattlefield(card, a.PlayerID())
						if name == "Mijae Djinn" {
							g.GetCombat().AddAttacker(p.ID(), b.PlayerID())
						} else {
							g.GetCombat().AddBlockerIncarnation(p.ID(), g.CaptureDamageSource(p.ID()).Incarnation(), attacker.ID())
						}
					}
				case "phase":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "ability-removal":
					g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				if change == "return" {
					if name == "Mijae Djinn" && (!g.GetCombat().IsAttacking(p.ID()) || g.FindPermanent(p.ID()).Tapped) {
						t.Fatal("followed returned attacker")
					}
					if name == "Ydwen Efreet" && !mage.CanBlock(g.FindPermanent(p.ID()), g.FindPermanent(attacker.ID()), g) {
						t.Fatal("restricted returned object")
					}
				} else {
					if g.GetCombat().IsAttacking(p.ID()) || g.GetCombat().IsBlocking(p.ID()) {
						t.Fatal("failed to remove old object")
					}
					if name == "Ydwen Efreet" {
						wantBlocked := change == "two-blockers" || change == "other-removed" || change == "depart" || change == "phase" || change == "phase-cycle" || change == "control"
						if g.GetCombat().GroupFor(attacker.ID()).Blocked != wantBlocked {
							t.Fatal("lost history or unblocked absent blocker")
						}
					}
				}
				if clone.StackSize() != g.StackSize()+1 && change != "copy" {
					t.Fatal("clone shared stack")
				}
				clone.ResolveStack()
				if change == "none" && name == "Ydwen Efreet" && clone.GetCombat().GroupFor(attacker.ID()).Blocked {
					t.Fatal("clone lost sole-blocker history")
				}
			})
		}
	}
}

func TestRONCombatExitVigilance(t *testing.T) {
	for _, win := range []bool{false, true} {
		g, a, b := ronGame()
		p := g.PutOnBattlefield(repositoryCard(t, "ARN", "Mijae Djinn"), a.PlayerID())
		p.RevokeBaseAttr(core.AttrSummonSick)
		p.GrantBaseAttr(core.Vigilance)
		g.SetCoinFlipResults([]bool{win})
		g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
		g.PutTriggersOnStack()
		if g.FindPermanent(p.ID()).Tapped || !g.GetCombat().IsAttacking(p.ID()) || g.StackSize() != 1 {
			t.Fatal("wrong announcement timing")
		}
		g.ResolveStack()
		if g.FindPermanent(p.ID()).Tapped == win || g.GetCombat().IsAttacking(p.ID()) != win || b.Life() != 20 {
			t.Fatal("loss failed to tap vigilant attacker")
		}
	}
}

func TestRONCombatExitTargetContexts(t *testing.T) {
	for _, action := range []string{"RemoveCreatureFromCombat", "RemoveCreatureFromCombatAndUnblockBlockers"} {
		for _, change := range []string{"none", "illegal", "return", "phase", "phase-cycle", "copy"} {
			t.Run(action+"/"+change, func(t *testing.T) {
				factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+action+`(Ref_TargetPermanent),CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[CantBlock],UntilEndOfTurn)])))`)
				g, a, b := ronGame()
				attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 8), a.PlayerID())
				card := mage.NewCreature("Blocker", "", 1, 8)
				blocker := g.PutOnBattlefield(card, b.PlayerID())
				g.GetCombat().AddAttacker(attacker.ID(), b.PlayerID())
				g.GetCombat().AddBlockerIncarnation(blocker.ID(), g.CaptureDamageSource(blocker.ID()).Incarnation(), attacker.ID())
				spell := factory()
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{blocker.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "illegal":
					g.MutablePermanent(blocker.ID()).GrantBaseAttr(core.Shroud)
				case "return":
					g.BouncePermanentToHand(g.FindPermanent(blocker.ID()))
					b.RemoveFromHand(card.ID())
					blocker = g.PutOnBattlefield(card, b.PlayerID())
					g.GetCombat().AddBlockerIncarnation(blocker.ID(), g.CaptureDamageSource(blocker.ID()).Incarnation(), attacker.ID())
				case "phase":
					g.PhaseOut(blocker.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(blocker.ID()))
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				g.PhaseIn([]uuid.UUID{blocker.ID()})
				legal := change == "none" || change == "copy" || change == "phase-cycle"
				if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(attacker.ID()), g) == legal {
					t.Fatal("wrong target legality or restriction")
				}
				if change == "none" && g.GetCombat().GroupFor(attacker.ID()).Blocked != (action == "RemoveCreatureFromCombat") {
					t.Fatal("ordinary removal changed blocked status")
				}
				clone.ResolveStack()
				if change == "none" && clone.GetCombat().IsBlocking(blocker.ID()) {
					t.Fatal("clone lost target")
				}
			})
		}
	}
}

func TestRONCombatExitInventoryAndReprints(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"ARN", "Mijae Djinn"}, {"3ED", "Mijae Djinn"}, {"ARN", "Ydwen Efreet"}} {
		card := repositoryCard(t, tc.set, tc.name)
		if err := mage.ValidateCardDefinition(tc.name, card); err != nil {
			t.Fatal(err)
		}
	}
	src := `Card(Name:"Exit",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[Activated(PayMana([]),ActionList([RemoveCreatureFromCombat(ThisPermanent),RemoveCreatureFromCombatAndUnblockBlockers(ThisPermanent),CreatePermanentRuleEffectUntil(ThisPermanent,[CantBlock],UntilEndOfTurn)]))])`
	report, err := InspectFS(fstest.MapFS{"exit.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatal(report.Issues)
	}
	for _, entry := range report.Signatures {
		if entry.Role == "unknown" {
			t.Fatalf("lost semantic role: %+v", entry)
		}
	}
}

func TestRONCombatExitLaterCombatsAndInstances(t *testing.T) {
	g, a, b := ronGame()
	g.SetActivePlayerIndex(1)
	first := g.PutOnBattlefield(repositoryCard(t, "ARN", "Ydwen Efreet"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "ARN", "Ydwen Efreet"), a.PlayerID())
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 10), b.PlayerID())
	other := g.PutOnBattlefield(mage.NewCreature("Other attacker", "", 1, 10), b.PlayerID())
	if first.Card.Abilities()[0] == second.Card.Abilities()[0] {
		t.Fatal("factory shared trigger")
	}
	g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
	g.GetCombat().AddAttacker(other.ID(), a.PlayerID())
	for i, p := range []*mage.Permanent{first, second} {
		target := attacker
		if i == 1 {
			target = other
		}
		g.GetCombat().AddBlockerIncarnation(p.ID(), g.CaptureDamageSource(p.ID()).Incarnation(), target.ID())
		g.FireEvent(core.GameEvent{Type: core.EvtDeclaredBlocker, SourceID: p.ID(), TargetID: target.ID(), PlayerID: a.PlayerID(), Flag: true})
		g.PutTriggersOnStack()
	}
	g.SetCoinFlipResults([]bool{true, false})
	g.ResolveStack()
	if mage.CanBlock(g.FindPermanent(first.ID()), g.FindPermanent(attacker.ID()), g) || !mage.CanBlock(g.FindPermanent(second.ID()), g.FindPermanent(attacker.ID()), g) {
		t.Fatal("pending instances lost separate outcomes")
	}
	g.GetCombat().Reset()
	g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
	g.GetCombat().AddAttacker(other.ID(), a.PlayerID())
	g.SetCoinFlipResults([]bool{true})
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: first.ID(), AttackerID: attacker.ID()}, {BlockerID: second.ID(), AttackerID: other.ID()}})
	if g.GetCombat().IsBlocking(first.ID()) || !g.GetCombat().IsBlocking(second.ID()) {
		t.Fatal("native later combat ignored separate restrictions")
	}
}

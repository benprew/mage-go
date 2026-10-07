package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCombatCollectionCards(t *testing.T) {
	for _, name := range []string{"Feint", "Sewers of Estark"} {
		for _, change := range []string{"none", "leave-pair", "blocker-return", "blocker-phase", "target-return", "target-phase", "target-shroud", "copy", "clone", "later-blocker", "second-blocker"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				set := "LEG"
				if name == "Sewers of Estark" {
					set = "PROMOS"
				}
				card := repositoryCard(t, set, name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				attackerCard := mage.NewCreature("Attacker", "", 2, 20)
				blockerCard := mage.NewCreature("Blocker", "", 3, 20)
				attacker := g.PutOnBattlefield(attackerCard, a.PlayerID())
				blocker := g.PutOnBattlefield(blockerCard, b.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 20), b.PlayerID())
				attacker.RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}})
				if name == "Feint" && change == "second-blocker" {
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: other.ID(), AttackerID: attacker.ID()}})
				}
				target := attacker
				if name == "Sewers of Estark" {
					target = blocker
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0); err == nil {
					t.Fatal("unpaid cast accepted")
				}
				if len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("failed cast changed state")
				}
				if name == "Feint" {
					a.ManaPool().Add(core.Red, 1)
				} else {
					a.ManaPool().Add(core.Black, 2)
					a.ManaPool().Add(core.Colorless, 2)
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				if sourceCombatPrevented(g, attacker.ID(), b.PlayerID()) || blocker.Tapped {
					t.Fatal("effect before resolution")
				}
				active := true
				switch change {
				case "leave-pair":
					g.RemoveFromCombat(blocker.ID())
				case "blocker-return":
					g.RemoveFromBattlefield(g.FindPermanent(blocker.ID()))
					g.PutOnBattlefield(blockerCard, b.PlayerID())
				case "blocker-phase":
					g.PhaseIn(g.PhaseOut(blocker.ID()))
				case "target-return":
					g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
					g.PutOnBattlefield(target.Card, target.ControllerID())
					active = false
				case "target-phase":
					g.PhaseIn(g.PhaseOut(target.ID()))
					active = false
				case "target-shroud":
					g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
					active = false
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				if change == "leave-pair" || change == "blocker-return" || change == "blocker-phase" {
					if name == "Sewers of Estark" {
						active = false
					}
				}
				clone := g.Clone()
				if change == "clone" {
					clone.RemoveFromCombat(blocker.ID())
				}
				g.ResolveStack()
				clone.ResolveStack()
				if change == "later-blocker" {
					g.GetCombat().AddBlocker(other.ID(), attacker.ID())
				}
				wantAttacker := active
				wantBlocker := active
				if change == "leave-pair" || change == "blocker-return" || change == "blocker-phase" {
					wantBlocker = false
				}
				if sourceCombatPrevented(g, attacker.ID(), b.PlayerID()) != wantAttacker || sourceCombatPrevented(g, blocker.ID(), a.PlayerID()) != wantBlocker {
					t.Fatal("wrong captured membership or target legality")
				}
				if name == "Feint" && g.FindPermanent(blocker.ID()).Tapped != wantBlocker {
					t.Fatal("tap collection")
				}
				if sourceCombatPrevented(g, other.ID(), a.PlayerID()) != (name == "Feint" && change == "second-blocker") {
					t.Fatal("unrelated or later blocker captured")
				}
				if change == "clone" && sourceCombatPrevented(clone, blocker.ID(), a.PlayerID()) {
					t.Fatal("clone followed original combat")
				}
				if change != "none" {
					return
				}
				g.ExecuteCombatDamage()
				if g.FindPermanent(attacker.ID()).Damage != 0 || g.FindPermanent(blocker.ID()).Damage != 0 {
					t.Fatal("native simultaneous combat damage")
				}
				g.RemoveFromCombat(blocker.ID())
				if !sourceCombatPrevented(g, blocker.ID(), a.PlayerID()) {
					t.Fatal("captured effect followed live combat")
				}
				g.GetEffects().RemoveEndOfCombat()
				want := name == "Feint"
				if sourceCombatPrevented(g, attacker.ID(), b.PlayerID()) != want {
					t.Fatal("wrong combat boundary duration")
				}
				g.RunStepWithPriority(core.Cleanup)
				if sourceCombatPrevented(g, attacker.ID(), b.PlayerID()) {
					t.Fatal("prevention survived cleanup")
				}
			})
		}
	}
}

func TestRONCombatCollectionInvalid(t *testing.T) {
	for _, action := range []string{
		`TapEachPermanent(And([IsCardtype(Creature),IsBlockingAttacker(Ref_TargetPlayer)]))`,
		`TapEachPermanent(IsBlockingAttacker(Ref_TargetPermanent,ThisPermanent))`,
		`TapEachPermanent(And([IsCardtype(Land),IsBlockingAttacker(Ref_TargetPermanent)]))`,
		`TapEachPermanent(And([IsCardtype(Creature),IsBlockingAttacker(Ref_TargetPermanent),IsUntapped]))`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(Or([])),[PreventThatDamage],UntilEndOfTurn)`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(SinglePermanent(Ref_TargetPermanent)),[PreventThatDamage()],UntilEndOfTurn)`,
		`CreatePreventDamageUntil(CombatDamageWouldBeDealtByACreature(SinglePermanent(Ref_TargetPermanent)),[PreventThatDamage],UntilYourNextTurn)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+action+`])))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONCombatCollectionSewersAttacking(t *testing.T) {
	for _, change := range []string{"none", "exit-before", "already-blocked", "copy-retarget"} {
		t.Run(change, func(t *testing.T) {
			a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A")}
			b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B")}
			g := mage.NewGame(a, b)
			g.SetTurn(2)
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "PROMOS", "Sewers of Estark")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			left := g.PutOnBattlefield(mage.NewCreature("Left", "", 2, 20), a.PlayerID())
			right := g.PutOnBattlefield(mage.NewCreature("Right", "", 2, 20), a.PlayerID())
			blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 20), b.PlayerID())
			left.RevokeBaseAttr(core.AttrSummonSick)
			right.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{left.ID(), right.ID()})
			if change == "already-blocked" {
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: left.ID()}})
			}
			a.ManaPool().Add(core.Black, 2)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{left.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if change == "exit-before" {
				g.RemoveFromCombat(left.ID())
			}
			if change == "copy-retarget" {
				b.targets = []uuid.UUID{right.ID()}
				copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
				if copied == nil {
					t.Fatal("copy failed")
				}
				if copied.Targets[0] != right.ID() {
					t.Fatal("copy did not retarget")
				}
			}
			g.ResolveStack()
			if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(left.ID()), g) != (change == "exit-before") {
				t.Fatal("attacking condition or blocking restriction")
			}
			if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(right.ID()), g) == (change == "copy-retarget") {
				t.Fatal("copy target binding")
			}
			if sourceCombatPrevented(g, left.ID(), b.PlayerID()) {
				t.Fatal("attacking branch prevented damage")
			}
			if change == "already-blocked" {
				g.ExecuteCombatDamage()
				if g.FindPermanent(left.ID()).Damage != 1 || g.FindPermanent(blocker.ID()).Damage != 2 {
					t.Fatal("restriction erased an existing block")
				}
			}
			g.RunStepWithPriority(core.Cleanup)
			if !mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(left.ID()), g) {
				t.Fatal("restriction survived cleanup")
			}
		})
	}
}

func TestRONCombatCollectionAnnouncementLegality(t *testing.T) {
	for _, name := range []string{"Feint", "Sewers of Estark"} {
		for _, change := range []string{"land", "not-attacking", "shroud", "phased"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				set := "LEG"
				if name == "Sewers of Estark" {
					set = "PROMOS"
				}
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				p := g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 20), b.PlayerID())
				if change == "land" {
					p = g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
				}
				if change == "shroud" {
					p.GrantBaseAttr(core.Shroud)
				}
				if change == "phased" {
					g.PhaseOut(p.ID())
				}
				a.ManaPool().Add(core.Red, 1)
				a.ManaPool().Add(core.Black, 2)
				a.ManaPool().Add(core.Colorless, 2)
				before := a.ManaPool().TotalMana()
				err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{p.ID()}, 0)
				if name == "Sewers of Estark" && change == "not-attacking" {
					if err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
					if sourceCombatPrevented(g, p.ID(), a.PlayerID()) {
						t.Fatal("idle creature gained prevention")
					}
					return
				}
				if err == nil || a.ManaPool().TotalMana() != before || g.StackSize() != 0 || len(a.Hand()) != 1 {
					t.Fatal("illegal target changed payment or stack")
				}
			})
		}
	}
}

func TestRONCombatCollectionStrictInventory(t *testing.T) {
	files := fstest.MapFS{}
	for _, entry := range []struct{ set, name string }{{"LEG", "Feint"}, {"PROMOS", "Sewers of Estark"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + entry.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		record := hostUntapRecord(t, entry.set, entry.name)
		line := strings.Split(string(data), "\n")[record.Position.Line-1]
		files[entry.name+".ron"] = &fstest.MapFile{Data: []byte(line)}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatalf("complete records: %v", err)
	}
	for _, name := range registry.RegisteredCardNames() {
		first, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		second, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		if first.Abilities()[0].(*mage.SpellAbility).Targets()[0] == second.Abilities()[0].(*mage.SpellAbility).Targets()[0] {
			t.Fatal("factories share target declarations")
		}
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 2 {
		t.Fatalf("inventory: %v", err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "IsBlockedByDefender" {
			found = true
			if signature.Role != "permanent_filter" {
				t.Fatal("nested relation role")
			}
		}
	}
	if !found {
		t.Fatal("nested relation missing")
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Rules:[Unknown])`)}
	if registry, err = LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial registry published")
	}
}

package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTemporaryRuleListCards(t *testing.T) {
	for _, name := range []string{"Blaze of Glory", "Hurr Jackal"} {
		for _, change := range []string{"live", "source-left", "phase", "cycle", "returned", "control", "copy", "abilities"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				set := "LEA"
				if name == "Hurr Jackal" {
					set = "ARN"
				}
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				targetCard := mage.NewCreature("Recipient", "", 1, 8)
				targetCard.SetOwner(b.PlayerID())
				target := g.PutOnBattlefield(targetCard, b.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 8), b.PlayerID())
				attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 8), a.PlayerID())
				second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 8), a.PlayerID())
				for _, p := range []*mage.Permanent{attacker, second} {
					p.RevokeBaseAttr(core.AttrSummonSick)
				}
				var source *mage.Permanent
				if name == "Hurr Jackal" {
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
						t.Fatal("unpaid cast")
					}
					a.ManaPool().Add(core.Red, 1)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
					source = g.FindPermanent(card.ID())
					if source.CurrentPower(g) != 1 || source.CurrentToughness(g) != 1 {
						t.Fatal("metadata")
					}
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil {
						t.Fatal("sick activation")
					}
					if source.Tapped || g.StackSize() != 0 {
						t.Fatal("rollback")
					}
					source.RevokeBaseAttr(core.AttrSummonSick)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
						t.Fatal(err)
					}
					if !g.FindPermanent(source.ID()).Tapped || target.HasKeyword(core.CantRegenerate) {
						t.Fatal("timing")
					}
				} else {
					g.SetStep(core.DeclareAttackers)
					g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID(), second.ID()})
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0); err == nil {
						t.Fatal("unpaid spell")
					}
					a.ManaPool().Add(core.White, 1)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
						t.Fatal(err)
					}
					if target.HasKeyword(core.CanBlockAny) || len(g.CompleteBlockAssignments(b.PlayerID(), nil)) != 0 {
						t.Fatal("early rule")
					}
				}
				pending := g.Clone()
				switch change {
				case "source-left":
					if source != nil {
						g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					}
				case "phase":
					g.PhaseOut(target.ID())
				case "cycle":
					g.PhaseIn(g.PhaseOut(target.ID()))
				case "returned":
					g.BouncePermanentToHand(g.FindPermanent(target.ID()))
					b.RemoveFromHand(targetCard.ID())
					target = g.PutOnBattlefield(targetCard, b.PlayerID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy")
					}
				}
				g.ResolveStack()
				g.PhaseIn([]uuid.UUID{target.ID()})
				active := change != "returned" && change != "phase"
				if name == "Blaze of Glory" && change == "control" {
					active = false
				}
				if change == "abilities" {
					g.AddContinuousEffect(mage.RemoveAllAbilities(target.ID(), core.Indefinite))
				}
				if name == "Hurr Jackal" {
					if g.FindPermanent(target.ID()).HasKeyword(core.CantRegenerate) != active || g.FindPermanent(other.ID()).HasKeyword(core.CantRegenerate) {
						t.Fatal("recipient identity")
					}
					g.AddRegenerationShield(target.ID())
					g.DestroyPermanent(g.FindPermanent(target.ID()))
					if (g.FindPermanent(target.ID()) == nil) != active {
						t.Fatal("regeneration prohibition")
					}
					g.AddRegenerationShield(other.ID())
					g.DestroyPermanent(g.FindPermanent(other.ID()))
					if g.FindPermanent(other.ID()) == nil {
						t.Fatal("other recipient")
					}
				} else {
					blocks := g.CompleteBlockAssignments(b.PlayerID(), nil)
					want := 0
					if active {
						want = 2
					}
					if len(blocks) != want {
						t.Fatalf("required blocks: %v", blocks)
					}
					if active {
						g.ExecuteBlockers(blocks)
						for _, group := range g.GetCombat().Groups {
							if len(group.BlockerIDs) != 1 || group.BlockerIDs[0] != target.ID() {
								t.Fatal("native blockers")
							}
						}
					}
				}
				pending.ResolveStack()
				if name == "Hurr Jackal" {
					if !pending.FindPermanent(targetCard.ID()).HasKeyword(core.CantRegenerate) {
						t.Fatal("pending clone")
					}
				} else if len(pending.CompleteBlockAssignments(b.PlayerID(), nil)) != 2 {
					t.Fatal("pending clone")
				}
				g.RunStepWithPriority(core.Cleanup)
				if p := g.FindPermanent(target.ID()); p != nil && (p.HasKeyword(core.CanBlockAny) || p.HasKeyword(core.CantRegenerate)) {
					t.Fatal("cleanup")
				}
			})
		}
	}
}

func TestRONTemporaryRuleListDeclarations(t *testing.T) {
	for _, ops := range []string{`CantBeRegenerated`, `CanBlockAnyNumberOfCreatures,MustBlockEachAttacker`, `MustBlockEachAttacker`} {
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[`+ops+`],UntilEndOfTurn)])))`)
		if err := mage.ValidateCardDefinition("Rules", factory()); err != nil {
			t.Fatal(err)
		}
	}
	for _, ops := range []string{``, `CantBeRegenerated()`, `MustBlockEachAttacker(You)`, `CanBlockAnyNumberOfCreatures()`, `CantBeRegenerated,Flying`, `DamageDealtToItCantBePreventedOrRedirected()`} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[%s],UntilEndOfTurn)])))])`, ops)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", ops)
		}
	}
}

func TestRONTemporaryRuleListInvalidScope(t *testing.T) {
	for _, action := range []string{
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[MustBlockEachAttacker],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(You,[CantBeRegenerated],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(ThisPermanent,[CanBlockAnyNumberOfCreatures],UntilEndOfCombat)`,
		`CreatePermanentRuleEffectUntil(ThisPermanent,[MustBlockEachAttacker],UntilEndOfTurn())`,
		`CreatePermanentRuleEffectUntil(HostPermanent,[CantBeRegenerated],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(ThisPermanent(),[CantBeRegenerated],UntilEndOfTurn)`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([` + action + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-rule-scope.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", action)
		}
	}
}

func TestRONTemporaryRuleListLegality(t *testing.T) {
	for _, name := range []string{"Blaze of Glory", "Hurr Jackal"} {
		for _, reason := range []string{"noncreature", "shroud", "phase", "control", "late-shroud", "late-phase", "window"} {
			t.Run(name+"/"+reason, func(t *testing.T) {
				g, a, b := ronGame()
				set := "LEA"
				if name == "Hurr Jackal" {
					set = "ARN"
				}
				target := g.PutOnBattlefield(mage.NewCreature("Target", "", 1, 8), b.PlayerID())
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				var source *mage.Permanent
				if name == "Hurr Jackal" {
					source = g.PutOnBattlefield(card, a.PlayerID())
					source.RevokeBaseAttr(core.AttrSummonSick)
				} else {
					g.SetStep(core.BeginCombat)
					a.AddToHand(card)
					a.ManaPool().Add(core.White, 1)
				}
				switch reason {
				case "noncreature":
					target = g.PutOnBattlefield(mage.NewArtifact("Artifact", ""), b.PlayerID())
				case "shroud":
					target.GrantBaseAttr(core.Shroud)
				case "phase":
					g.PhaseOut(target.ID())
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: uuid.New(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "window":
					if name == "Blaze of Glory" {
						g.SetStep(core.DeclareBlockers)
					} else {
						source.Tapped = true
					}
				}
				activate := func() error {
					if source != nil {
						return g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()})
					}
					return g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0)
				}
				err := activate()
				valid := strings.HasPrefix(reason, "late-") || name == "Hurr Jackal" && reason == "control"
				if (err == nil) != valid {
					t.Fatalf("valid=%v err=%v", valid, err)
				}
				if !valid {
					if g.StackSize() != 0 || source != nil && source.Tapped != (reason == "window") || source == nil && (a.ManaPool().Count(core.White) != 1 || len(a.Hand()) != 1) {
						t.Fatal("illegal action spent resources")
					}
					return
				}
				if reason == "late-shroud" {
					g.MutablePermanent(target.ID()).GrantBaseAttr(core.Shroud)
				}
				if reason == "late-phase" {
					g.PhaseOut(target.ID())
				}
				g.ResolveStack()
				if reason == "late-phase" {
					g.PhaseIn([]uuid.UUID{target.ID()})
				}
				if strings.HasPrefix(reason, "late-") && g.FindPermanent(target.ID()).HasKeyword(core.CantRegenerate) {
					t.Fatal("illegal resolution applied rule")
				}
			})
		}
	}
}

func TestRONTemporaryRuleListBindingsAndExpiry(t *testing.T) {
	for _, name := range []string{"Blaze of Glory", "Hurr Jackal"} {
		g, a, b := ronModalGame(0)
		g.SetActivePlayerIndex(0)
		set := "LEA"
		if name == "Hurr Jackal" {
			set = "ARN"
		}
		node := hostUntapRecord(t, set, name)
		factory, err := Compile(node)
		if err != nil {
			t.Fatal(err)
		}
		one, two := factory(), factory()
		if one.Abilities()[0] == two.Abilities()[0] {
			t.Fatal("shared declarations")
		}
		first := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 8), b.PlayerID())
		second := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 8), b.PlayerID())
		if name == "Hurr Jackal" {
			for i, card := range []mage.Card{one, two} {
				source := g.PutOnBattlefield(card, a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				target := first
				if i == 1 {
					target = second
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveTopOfStack()
			if g.FindPermanent(first.ID()).HasKeyword(core.CantRegenerate) || !g.FindPermanent(second.ID()).HasKeyword(core.CantRegenerate) {
				t.Fatal("independent pending target")
			}
			g.ResolveStack()
			clone := g.Clone()
			g.RunStepWithPriority(core.Cleanup)
			for _, id := range []uuid.UUID{first.ID(), second.ID()} {
				if g.FindPermanent(id).HasKeyword(core.CantRegenerate) || !clone.FindPermanent(id).HasKeyword(core.CantRegenerate) {
					t.Fatal("expiry isolation")
				}
				g.AddRegenerationShield(id)
				g.DestroyPermanent(g.FindPermanent(id))
				if g.FindPermanent(id) == nil {
					t.Fatal("regeneration after expiry")
				}
			}
		} else {
			g.SetStep(core.BeginCombat)
			one.SetOwner(a.PlayerID())
			a.AddToHand(one)
			a.ManaPool().Add(core.White, 1)
			if err := g.CastSpellByID(a.PlayerID(), one.ID(), []uuid.UUID{first.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			original := g.StackPeek()
			a.targets = []uuid.UUID{second.ID()}
			copied := g.CopyStackObjectDirect(original, a.PlayerID(), true)
			if copied == nil {
				t.Fatal("copy")
			}
			if copied.Targets[0] != second.ID() || original.Targets[0] != first.ID() {
				t.Fatal("retarget isolation")
			}
			g.ResolveTopOfStack()
			g.ResolveStack()
			for _, p := range []*mage.Permanent{first, second} {
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 1, 8), a.PlayerID())
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
			if len(g.CompleteBlockAssignments(b.PlayerID(), nil)) != 2 {
				t.Fatal("retargeted copy or phase cycle")
			}
			clone := g.Clone()
			g.RunStepWithPriority(core.Cleanup)
			g.MutablePermanent(attacker.ID()).Tapped = false
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
			if len(g.CompleteBlockAssignments(b.PlayerID(), nil)) != 0 || len(clone.CompleteBlockAssignments(b.PlayerID(), nil)) != 2 {
				t.Fatal("cleanup isolation")
			}
		}
	}
}

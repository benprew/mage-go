package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTemporaryCombatRuleCards(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		ability   int
	}{
		{"DRK", "Tower of Coireall", 0}, {"LEG", "Wall of Wonder", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid casting accepted")
			}
			if g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("failed casting changed state")
			}
			a.ManaPool().Add(core.Colorless, 2)
			a.ManaPool().Add(core.Blue, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("entered before resolution")
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			attacker := source
			var targets []uuid.UUID
			if tc.name == "Tower of Coireall" {
				attacker = g.PutOnBattlefield(mage.NewCreature("Attacker", "", 3, 4), a.PlayerID())
				attacker.RevokeBaseAttr(core.AttrSummonSick)
				targets = []uuid.UUID{attacker.ID()}
			}
			wall := g.PutOnBattlefield(mage.NewCreature("Blocker Wall", "", 1, 8, mage.WithSubTypes("Wall")), b.PlayerID())
			plain := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 8), b.PlayerID())
			a.ManaPool().Clear()
			if tc.name == "Wall of Wonder" {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), tc.ability, targets); err == nil {
					t.Fatal("unpaid activation accepted")
				}
				if g.StackSize() != 0 || source.Tapped {
					t.Fatal("payment rollback")
				}
				a.ManaPool().Add(core.Colorless, 2)
				a.ManaPool().Add(core.Blue, 2)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), tc.ability, targets); err != nil {
				t.Fatal(err)
			}
			if tc.name == "Tower of Coireall" {
				if !g.FindPermanent(source.ID()).Tapped || !mage.CanBlock(wall, attacker, g) {
					t.Fatal("tap payment or early rule")
				}
			} else if g.FindPermanent(source.ID()).CanDeclareAsAttacker(g) || g.FindPermanent(source.ID()).CurrentPower(g) != 1 {
				t.Fatal("early source rule")
			}
			clone := g.Clone()
			clone.ResolveStack()
			g.ResolveStack()
			attacker = g.FindPermanent(attacker.ID())
			if tc.name == "Tower of Coireall" {
				if mage.CanBlock(g.FindPermanent(wall.ID()), attacker, g) || !mage.CanBlock(g.FindPermanent(plain.ID()), attacker, g) {
					t.Fatal("blocking direction")
				}
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddSubtypeFamily(mage.This, mage.SubtypeCreature, "Wall")), plain.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if mage.CanBlock(g.FindPermanent(plain.ID()), attacker, g) {
					t.Fatal("filter did not read current subtype")
				}
				if !mage.CanBlock(clone.FindPermanent(plain.ID()), clone.FindPermanent(attacker.ID()), clone) {
					t.Fatal("clone changed")
				}
			} else {
				if !attacker.CanDeclareAsAttacker(g) || attacker.CurrentPower(g) != 5 || attacker.CurrentToughness(g) != 1 || !attacker.HasKeyword(core.Defender) {
					t.Fatal("combined bonus and permission")
				}
				if a.ManaPool().TotalMana() != 0 {
					t.Fatal("activation payment")
				}
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
				if !g.GetCombat().IsAttacking(attacker.ID()) || !g.FindPermanent(attacker.ID()).Tapped {
					t.Fatal("native attack rejected")
				}
				g.ExecuteCombatDamage()
				if b.Life() != 15 {
					t.Fatal("native damage", b.Life())
				}
			}
			g.RunStepWithPriority(core.Cleanup)
			if tc.name == "Tower of Coireall" {
				if !mage.CanBlock(g.FindPermanent(wall.ID()), g.FindPermanent(attacker.ID()), g) {
					t.Fatal("blocking cleanup")
				}
				if mage.CanBlock(clone.FindPermanent(wall.ID()), clone.FindPermanent(attacker.ID()), clone) {
					t.Fatal("clone cleanup isolation")
				}
			} else {
				checkConditionalPT(t, g, attacker.ID(), 1, 5)
				g.MutablePermanent(attacker.ID()).Tapped = false
				if g.FindPermanent(attacker.ID()).CanDeclareAsAttacker(g) {
					t.Fatal("permission survived cleanup")
				}
				checkConditionalPT(t, clone, attacker.ID(), 5, 1)
			}
		})
	}
}

func TestRONTemporaryCombatRuleLifecycle(t *testing.T) {
	for _, name := range []string{"Tower of Coireall", "Wall of Wonder"} {
		for _, change := range []string{"phase-before", "phase-cycle", "return-before", "return-after", "control", "source-left", "abilities", "cleanup-phased"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				set, index := "DRK", 0
				if name == "Wall of Wonder" {
					set, index = "LEG", 0
				}
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				targetCard := card
				target := source
				var targets []uuid.UUID
				if name == "Tower of Coireall" {
					targetCard = mage.NewCreature("Target", "", 2, 4)
					targetCard.SetOwner(a.PlayerID())
					target = g.PutOnBattlefield(targetCard, a.PlayerID())
					targets = []uuid.UUID{target.ID()}
				}
				target.RevokeBaseAttr(core.AttrSummonSick)
				blocker := g.PutOnBattlefield(mage.NewCreature("Wall", "", 1, 8, mage.WithSubTypes("Wall")), b.PlayerID())
				a.ManaPool().Add(core.Blue, 4)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, targets); err != nil {
					t.Fatal(err)
				}
				changeObject := func() {
					switch change {
					case "phase-before", "cleanup-phased":
						g.PhaseOut(target.ID())
					case "phase-cycle":
						g.PhaseIn(g.PhaseOut(target.ID()))
					case "return-before", "return-after":
						g.BouncePermanentToHand(g.FindPermanent(target.ID()))
						a.RemoveFromHand(targetCard.ID())
						target = g.PutOnBattlefield(targetCard, a.PlayerID())
						target.RevokeBaseAttr(core.AttrSummonSick)
					case "control":
						g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: target.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					case "source-left":
						g.BouncePermanentToHand(g.FindPermanent(source.ID()))
					case "abilities":
						g.AddContinuousEffect(mage.RemoveAllAbilities(target.ID(), core.EndOfTurn))
					}
				}
				after := change == "return-after" || change == "abilities" || change == "cleanup-phased"
				if !after {
					changeObject()
				}
				g.ResolveStack()
				if after {
					changeObject()
				}
				if change == "cleanup-phased" {
					g.RunStepWithPriority(core.Cleanup)
				}
				if change == "phase-before" || change == "cleanup-phased" {
					g.PhaseIn([]uuid.UUID{target.ID()})
				}
				g.ApplyContinuousEffects()
				if name == "Wall of Wonder" && change == "source-left" {
					a.RemoveFromHand(card.ID())
					target = g.PutOnBattlefield(card, a.PlayerID())
					target.RevokeBaseAttr(core.AttrSummonSick)
				}
				want := change != "phase-before" && change != "return-before" && change != "return-after" && change != "cleanup-phased" && (name != "Wall of Wonder" || change != "source-left")
				target = g.FindPermanent(target.ID())
				if name == "Tower of Coireall" {
					if mage.CanBlock(g.FindPermanent(blocker.ID()), target, g) == want {
						t.Fatal("captured blocking lifetime")
					}
				} else {
					target.Tapped = false
					target.RevokeBaseAttr(core.AttrSummonSick)
					target.GrantBaseAttr(core.Defender)
					g.ApplyContinuousEffects()
					if g.FindPermanent(target.ID()).CanDeclareAsAttacker(g) != want {
						t.Fatal("captured attack lifetime")
					}
					power, toughness := 1, 5
					if want {
						power, toughness = 5, 1
					}
					checkConditionalPT(t, g, target.ID(), power, toughness)
				}
			})
		}
	}
}

func TestRONTemporaryCombatRuleLegality(t *testing.T) {
	for _, change := range []string{"land", "shroud", "phase-before", "return-before", "noncreature-before"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Tower of Coireall"), a.PlayerID())
			card := mage.Card(mage.NewCreature("Target", "", 2, 4))
			if change == "land" {
				card = mage.NewLand("Land")
			}
			card.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(card, b.PlayerID())
			if change == "shroud" {
				g.AddContinuousEffect(mage.TemporaryKeyword(target.ID(), core.Shroud))
			}
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()})
			if change == "land" || change == "shroud" {
				if err == nil || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 {
					t.Fatal("illegal declaration or rollback")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "phase-before":
				g.PhaseOut(target.ID())
			case "return-before":
				g.BouncePermanentToHand(g.FindPermanent(target.ID()))
				b.RemoveFromHand(card.ID())
				target = g.PutOnBattlefield(card, b.PlayerID())
			case "noncreature-before":
				g.MutablePermanent(target.ID()).RevokeBaseAttr(core.AttrIsCreature)
			}
			g.ResolveStack()
			if change == "phase-before" {
				g.PhaseIn([]uuid.UUID{target.ID()})
			}
			if change == "noncreature-before" {
				g.MutablePermanent(target.ID()).GrantBaseAttr(core.AttrIsCreature)
			}
			g.ApplyContinuousEffects()
			wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 1, 8, mage.WithSubTypes("Wall")), a.PlayerID())
			if !mage.CanBlock(wall, g.FindPermanent(target.ID()), g) {
				t.Fatal("illegal target received rule")
			}
		})
	}
}

func TestRONTemporaryCombatRuleCopy(t *testing.T) {
	g, a, b := ronModalGame(0)
	card := repositoryCard(t, "DRK", "Tower of Coireall")
	other := repositoryCard(t, "DRK", "Tower of Coireall")
	if card.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == other.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("shared targets")
	}
	source := g.PutOnBattlefield(card, a.PlayerID())
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 4), a.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 4), a.PlayerID())
	wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 1, 8, mage.WithSubTypes("Wall")), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
		t.Fatal(err)
	}
	a.targets = []uuid.UUID{second.ID()}
	if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), true) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveTopOfStack()
	if mage.CanBlock(g.FindPermanent(wall.ID()), g.FindPermanent(second.ID()), g) || !mage.CanBlock(g.FindPermanent(wall.ID()), g.FindPermanent(first.ID()), g) {
		t.Fatal("copy bindings")
	}
	g.ResolveStack()
	if mage.CanBlock(g.FindPermanent(wall.ID()), g.FindPermanent(first.ID()), g) {
		t.Fatal("original binding")
	}
}

func TestRONTemporaryCombatRuleReprints(t *testing.T) {
	paths, err := os.ReadDir("../../../../cards/mtgish")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	count := 0
	for _, path := range paths {
		if !strings.HasSuffix(path.Name(), ".ron") {
			continue
		}
		data, err := os.ReadFile("../../../../cards/mtgish/" + path.Name())
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), path.Name())
		if err != nil {
			t.Fatal(err)
		}
		var lines []string
		for _, node := range nodes {
			name := node.Fields["Name"].String
			if name != "Tower of Coireall" && name != "Wall of Wonder" {
				continue
			}
			count++
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			first, second := factory(), factory()
			if first.ID() == second.ID() {
				t.Fatal("shared identity")
			}
			for _, card := range []mage.Card{first, second} {
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
			}
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "Tower of Coireall"`) || strings.Contains(line, `Name: "Wall of Wonder"`) {
				lines = append(lines, line)
			}
		}
		if len(lines) > 0 {
			files[path.Name()] = &fstest.MapFile{Data: []byte(strings.Join(lines, "\n"))}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 || len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("collection counts", count)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Unsupported])`)}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("partial registry")
	}
}

func TestRONTemporaryCombatRuleInvalid(t *testing.T) {
	for _, body := range []string{
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[CanAttackAsThoughItHadHaste()],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[CanAttackAsThoughItDidntHaveDefender(AnyPermanent)],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[CantBeBlockedByDefenders],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[CantBeBlockedByDefenders(AnyPermanent,AnyPermanent)],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[CantBeBlockedByDefenders(And(AnyPermanent))],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[CantBeBlockedByDefenders(IsCreatureType(1))],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[CanAttackAsThoughItHadHaste,CantBeBlocked],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[CanAttackAsThoughItHadHaste],UntilEndOfCombat)`,
		`CreatePermanentRuleEffectUntil(Ref_TargetPlayer,[CanAttackAsThoughItHadHaste],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(ThisPermanent,[CanAttackAsThoughItHadHaste],UntilEndOfTurn)`,
		`CreatePermanentRuleEffectUntil(HostPermanent,[CanAttackAsThoughItHadHaste],UntilEndOfTurn)`,
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([%s])))])`, body)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", body)
		}
	}
}

func TestRONTemporaryCombatRuleAttackContexts(t *testing.T) {
	for _, operation := range []string{"CanAttackAsThoughItHadHaste", "CanAttackAsThoughItDidntHaveDefender"} {
		for _, context := range []string{"source", "spell", "mode", "trigger"} {
			t.Run(operation+"/"+context, func(t *testing.T) {
				g, a, _ := ronModalGame(0)
				body := `Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[` + operation + `],UntilEndOfTurn)]))`
				kind, rule := "Instant", `SpellActions(`+body+`)`
				if context == "mode" {
					rule = `SpellActions(Modal_ChooseOne([` + body + `,ActionList([PlayerAction(You,GainLife(Integer(1)))] )]))`
				}
				if context == "trigger" {
					kind, rule = "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),`+body+`)`
				}
				if context == "source" {
					kind, rule = "Creature", `Activated(PayMana([ManaCostGeneric(0)]),ActionList([CreatePermanentRuleEffectUntil(ThisPermanent,[`+operation+`],UntilEndOfTurn)]))`
				}
				pt := ""
				if kind == "Creature" {
					pt = ",CardPT:(Power:2,Toughness:4)"
				}
				src := fmt.Sprintf(`Card(Name:"Temporary",Typeline:(Supertypes:[],Cardtypes:[%s],Subtypes:[])%s,Rules:[%s])`, kind, pt, rule)
				nodes, err := Parse(strings.NewReader(src), "temporary.ron")
				if err != nil {
					t.Fatal(err)
				}
				factory, err := Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
				card := factory()
				card.SetOwner(a.PlayerID())
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				var target *mage.Permanent
				if context == "source" {
					target = g.PutOnBattlefield(card, a.PlayerID())
				} else {
					target = g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 4), a.PlayerID())
				}
				if operation == "CanAttackAsThoughItDidntHaveDefender" {
					target.GrantBaseAttr(core.Defender)
					target.RevokeBaseAttr(core.AttrSummonSick)
				}
				if target.CanDeclareAsAttacker(g) {
					t.Fatal("missing initial restriction")
				}
				switch context {
				case "source":
					if err := g.ActivateAbilityByIndex(a.PlayerID(), target.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
				case "trigger":
					a.targets = []uuid.UUID{target.ID()}
					g.PutOnBattlefield(card, a.PlayerID())
					g.PutTriggersOnStack()
				default:
					a.AddToHand(card)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
						t.Fatal(err)
					}
				}
				if g.FindPermanent(target.ID()).CanDeclareAsAttacker(g) {
					t.Fatal("permission before resolution")
				}
				g.ResolveStack()
				target = g.FindPermanent(target.ID())
				if !target.CanDeclareAsAttacker(g) {
					t.Fatal("missing attack permission")
				}
				if operation == "CanAttackAsThoughItHadHaste" {
					if target.HasKeyword(core.Haste) || target.CanTapForEffect(g) {
						t.Fatal("granted full haste")
					}
					target.GrantBaseAttr(core.Defender)
					g.ApplyContinuousEffects()
					if g.FindPermanent(target.ID()).CanDeclareAsAttacker(g) {
						t.Fatal("haste permission bypassed defender")
					}
					g.MutablePermanent(target.ID()).RevokeBaseAttr(core.Defender)
				} else {
					g.MutablePermanent(target.ID()).GrantBaseAttr(core.AttrSummonSick)
					g.ApplyContinuousEffects()
					if g.FindPermanent(target.ID()).CanDeclareAsAttacker(g) {
						t.Fatal("defender permission bypassed control history")
					}
					g.MutablePermanent(target.ID()).RevokeBaseAttr(core.AttrSummonSick)
				}
				g.AddContinuousEffect(mage.RemoveAllAbilities(target.ID(), core.EndOfTurn))
				g.ApplyContinuousEffects()
				if !g.FindPermanent(target.ID()).CanDeclareAsAttacker(g) {
					t.Fatal("ability removal erased rule permission")
				}
				clone := g.Clone()
				g.PhaseIn(g.PhaseOut(target.ID()))
				g.ApplyContinuousEffects()
				if !g.FindPermanent(target.ID()).CanDeclareAsAttacker(g) {
					t.Fatal("phase cycle lost permission")
				}
				g.RunStepWithPriority(core.Cleanup)
				attr := core.AttrCanAttack
				if operation == "CanAttackAsThoughItHadHaste" {
					attr = core.AttrAttackAsThoughHaste
				}
				if operation == "CanAttackAsThoughItDidntHaveDefender" {
					for _, game := range []*mage.Game{g, clone} {
						game.MutablePermanent(target.ID()).GrantBaseAttr(core.Defender)
						game.MutablePermanent(target.ID()).RevokeBaseAttr(core.AttrSummonSick)
						game.ApplyContinuousEffects()
					}
					if g.FindPermanent(target.ID()).CanDeclareAsAttacker(g) || !clone.FindPermanent(target.ID()).CanDeclareAsAttacker(clone) {
						t.Fatal("defender permission cleanup or clone isolation")
					}
				} else if g.FindPermanent(target.ID()).HasAttr(attr) || !clone.FindPermanent(target.ID()).HasAttr(attr) {
					t.Fatal("cleanup or clone isolation")
				}
			})
		}
	}
}

func TestRONTemporaryCombatRuleNativeBlocking(t *testing.T) {
	for _, alreadyBlocked := range []bool{false, true} {
		t.Run(fmt.Sprint(alreadyBlocked), func(t *testing.T) {
			g, a, b := ronGame()
			tower := g.PutOnBattlefield(repositoryCard(t, "DRK", "Tower of Coireall"), a.PlayerID())
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 3, 4), a.PlayerID())
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 1, 8, mage.WithSubTypes("Wall")), b.PlayerID())
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{attacker.ID()})
			assignments := []mage.BlockAssignment{{BlockerID: wall.ID(), AttackerID: attacker.ID()}}
			if alreadyBlocked {
				g.ExecuteBlockers(assignments)
				if !g.GetCombat().IsBlocking(wall.ID()) {
					t.Fatal("initial block rejected")
				}
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), tower.ID(), 0, []uuid.UUID{attacker.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if !alreadyBlocked {
				g.ExecuteBlockers(assignments)
			}
			if g.GetCombat().IsBlocking(wall.ID()) != alreadyBlocked {
				t.Fatal("native block legality or declared block lifetime")
			}
			g.ExecuteCombatDamage()
			want := 17
			if alreadyBlocked {
				want = 20
			}
			if b.Life() != want {
				t.Fatal("native damage", b.Life())
			}
		})
	}
}

func TestRONTemporaryCombatRuleFilteredContexts(t *testing.T) {
	for _, context := range []string{"source", "spell", "mode", "trigger"} {
		t.Run(context, func(t *testing.T) {
			g, a, b := ronModalGame(0)
			body := `Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentRuleEffectUntil(Ref_TargetPermanent,[CantBeBlockedByDefenders(IsCreatureType(Wall))],UntilEndOfTurn)]))`
			kind, rule := "Instant", `SpellActions(`+body+`)`
			if context == "mode" {
				rule = `SpellActions(Modal_ChooseOne([` + body + `,ActionList([PlayerAction(You,GainLife(Integer(1)))] )]))`
			}
			if context == "trigger" {
				kind, rule = "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),`+body+`)`
			}
			if context == "source" {
				kind, rule = "Creature", `Activated(PayMana([ManaCostGeneric(0)]),ActionList([CreatePermanentRuleEffectUntil(ThisPermanent,[CantBeBlockedByDefenders(IsCreatureType(Wall))],UntilEndOfTurn)]))`
			}
			pt := ""
			if kind == "Creature" {
				pt = ",CardPT:(Power:2,Toughness:4)"
			}
			src := fmt.Sprintf(`Card(Name:"Filtered",Typeline:(Supertypes:[],Cardtypes:[%s],Subtypes:[])%s,Rules:[%s])`, kind, pt, rule)
			nodes, err := Parse(strings.NewReader(src), "filtered.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			card.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			var target *mage.Permanent
			if context == "source" {
				target = g.PutOnBattlefield(card, a.PlayerID())
			} else {
				target = g.PutOnBattlefield(mage.NewCreature("Target", "", 2, 4), a.PlayerID())
			}
			wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 1, 8, mage.WithSubTypes("Wall")), b.PlayerID())
			switch context {
			case "source":
				if err := g.ActivateAbilityByIndex(a.PlayerID(), target.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
			case "trigger":
				a.targets = []uuid.UUID{target.ID()}
				g.PutOnBattlefield(card, a.PlayerID())
				g.PutTriggersOnStack()
			default:
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
					t.Fatal(err)
				}
			}
			if !mage.CanBlock(wall, g.FindPermanent(target.ID()), g) {
				t.Fatal("rule before resolution")
			}
			g.ResolveStack()
			if mage.CanBlock(g.FindPermanent(wall.ID()), g.FindPermanent(target.ID()), g) {
				t.Fatal("missing filtered restriction")
			}
			g.RunStepWithPriority(core.Cleanup)
			if !mage.CanBlock(g.FindPermanent(wall.ID()), g.FindPermanent(target.ID()), g) {
				t.Fatal("filter cleanup")
			}
		})
	}
}

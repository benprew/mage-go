package ron

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

var blockingProhibitionCards = []struct {
	set, name        string
	power, toughness int
	blocking         bool
}{
	{"LEA", "Ironclaw Orcs", 2, 2, true},
	{"FEM", "Brassclaw Orcs", 3, 2, true},
	{"FEM", "Orcish Veteran", 2, 2, true},
	{"LEG", "Amrou Kithkin", 1, 1, false},
	{"DRK", "Bog Rats", 1, 1, false},
}

func TestRONBlockingProhibitionCards(t *testing.T) {
	for _, tc := range blockingProhibitionCards {
		for _, power := range []int{-1, 0, 1, 2, 3, 4} {
			for _, white := range []bool{false, true} {
				for _, wall := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%d/%v/%v", tc.name, power, white, wall), func(t *testing.T) {
						g, a, b := ronGame()
						card := repositoryCard(t, tc.set, tc.name)
						if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
							t.Fatal(err)
						}
						source := g.PutOnBattlefield(card, a.PlayerID())
						if source.CurrentPower(g) != tc.power || source.CurrentToughness(g) != tc.toughness {
							t.Fatal("wrong source stats")
						}
						otherCard := mage.NewCreature("Other", "", power, 8)
						if white {
							otherCard.SetColorOverride([]core.Color{core.White, core.Blue})
						}
						if wall {
							otherCard.AddSubType("Wall")
						}
						other := g.PutOnBattlefield(otherCard, b.PlayerID())
						blocker, attacker := other, source
						prohibited := (tc.name == "Bog Rats" && wall) || (tc.name == "Amrou Kithkin" && power >= 3)
						if tc.blocking {
							blocker, attacker = source, other
							prohibited = power >= 2 && (tc.name != "Orcish Veteran" || white)
						}
						if mage.CanBlock(blocker, attacker, g) == prohibited {
							t.Fatal("wrong filter or direction")
						}
						clone := g.Clone()
						if mage.CanBlock(clone.FindPermanent(blocker.ID()), clone.FindPermanent(attacker.ID()), clone) == prohibited {
							t.Fatal("immediate clone lost rule")
						}
						g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
						if !mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(attacker.ID()), g) {
							t.Fatal("removed ability still restricts blocks")
						}
						if mage.CanBlock(clone.FindPermanent(blocker.ID()), clone.FindPermanent(attacker.ID()), clone) == prohibited {
							t.Fatal("clone changed with original")
						}
						g.RunStepWithPriority(core.Cleanup)
						if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(attacker.ID()), g) == prohibited {
							t.Fatal("cleanup did not restore rule")
						}
					})
				}
			}
		}
	}
}

func TestRONBlockingProhibitionInvalid(t *testing.T) {
	for _, operation := range []string{"CantBlockAttackers", "CantBeBlockedByDefenders"} {
		for _, declaration := range []string{operation, operation + "()", operation + "(AnyPermanent,AnyPermanent)", operation + "(IsColor(NoColor))", operation + "(PowerIs(GreaterThanOrEqualTo(ValueX)))", operation + "(PowerIs(GreaterThanOrEqualTo(Integer(2,3))))", operation + "(IsCardtype(Creature)),CantAttack"} {
			for _, subject := range []string{"ThisPermanent", "Ref_TargetPermanent", "Trigger_ThatPermanent", "HostPermanent"} {
				src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:2),Rules:[PermanentRuleEffect(%s,[%s])])`, subject, declaration)
				nodes, err := Parse(strings.NewReader(src), "invalid.ron")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := Compile(nodes[0]); err == nil {
					t.Fatalf("accepted %s", src)
				}
			}
		}
	}
	for _, rule := range []string{`PermanentRuleEffect(HostPermanent,[CantBlockAttackers(AnyPermanent)])`, `SpellActions(ActionList([CantBlockAttackers(AnyPermanent)]))`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted invalid context", rule)
		}
	}
}

func TestRONBlockingProhibitionInventory(t *testing.T) {
	for _, operation := range []string{"CantBlockAttackers", "CantBeBlockedByDefenders"} {
		src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:2),Rules:[PermanentRuleEffect(ThisPermanent,[` + operation + `(And([IsCardtype(Creature),PowerIs(GreaterThanOrEqualTo(Integer(2)))]))])])`
		report, err := InspectFS(fstest.MapFS{"r.ron": {Data: []byte(src)}}, "*.ron")
		if err != nil {
			t.Fatal(err)
		}
		for name, role := range map[string]string{"And": "permanent_filter", "PowerIs": "permanent_filter", "GreaterThanOrEqualTo": "power_comparison", "Integer": "value"} {
			if !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool { return s.Name == name && s.Role == role }) {
				t.Fatalf("missing %s role %s", name, role)
			}
		}
	}
}

func TestRONBlockingProhibitionStrictCollection(t *testing.T) {
	files := fstest.MapFS{}
	paths, err := filepath.Glob("../../../../cards/mtgish/*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), path)
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			name := node.Fields["Name"].String
			if !slices.ContainsFunc(blockingProhibitionCards, func(c struct {
				set, name        string
				power, toughness int
				blocking         bool
			}) bool {
				return c.name == name
			}) {
				continue
			}
			factory, err := Compile(node)
			if err != nil {
				t.Fatal(err)
			}
			first, second := factory(), factory()
			if err := mage.ValidateCardDefinition(name, first); err != nil {
				t.Fatal(err)
			}
			if first.Abilities()[0] == second.Abilities()[0] {
				t.Fatal("factory shared ability")
			}
			lines := strings.Split(string(data), "\n")
			files[filepath.Base(path)+name+".ron"] = &fstest.MapFile{Data: []byte(lines[node.Position.Line-1])}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(files) != 9 || len(registry.RegisteredCardNames()) != 5 {
		t.Fatalf("strict collection: %d records, %v", len(files), err)
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[PermanentRuleEffect(ThisPermanent,[CantBlockAttackers()])])`)}
	if partial, err := LoadFS(files, "*.ron"); err == nil || partial != nil {
		t.Fatal("published partial registry")
	}
}

func TestRONBlockingProhibitionNativeCombat(t *testing.T) {
	for _, tc := range blockingProhibitionCards {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast succeeded")
			}
			if g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("unpaid cast changed state")
			}
			a.ManaPool().Add(core.Red, 8)
			a.ManaPool().Add(core.White, 8)
			a.ManaPool().Add(core.Black, 8)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("entered before resolution")
			}
			g.ResolveStack()
			source := g.FindPermanent(card.ID())
			if tc.name == "Orcish Veteran" {
				a.ManaPool().Clear()
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err == nil {
					t.Fatal("unpaid activation succeeded")
				}
				if g.StackSize() != 0 || g.FindPermanent(source.ID()).HasKeyword(core.FirstStrike) {
					t.Fatal("unpaid activation changed state")
				}
				a.ManaPool().Add(core.Red, 1)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().TotalMana() != 0 {
					t.Fatal("activation did not pay red mana")
				}
				if source.HasKeyword(core.FirstStrike) {
					t.Fatal("grant before resolution")
				}
				g.ResolveStack()
				if !g.FindPermanent(source.ID()).HasKeyword(core.FirstStrike) {
					t.Fatal("missing first strike")
				}
			}
			otherCard := mage.NewCreature("Other", "", 3, 8)
			otherCard.SetColorOverride([]core.Color{core.White})
			otherCard.AddSubType("Wall")
			other := g.PutOnBattlefield(otherCard, b.PlayerID())
			attacker, blocker, active := source, other, a.PlayerID()
			if tc.blocking {
				attacker, blocker, active = other, source, b.PlayerID()
			}
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(active, []uuid.UUID{attacker.ID()})
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}})
			if g.GetCombat().IsBlocking(blocker.ID()) {
				t.Fatal("illegal block accepted")
			}
			g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}})
			if !g.GetCombat().IsBlocking(blocker.ID()) {
				t.Fatal("legal block rejected")
			}
			g.RunStepWithPriority(core.Cleanup)
			if tc.name == "Orcish Veteran" && g.FindPermanent(source.ID()).HasKeyword(core.FirstStrike) {
				t.Fatal("first strike did not expire")
			}
		})
	}
}

func TestRONBlockingProhibitionCurrentCharacteristics(t *testing.T) {
	for _, tc := range blockingProhibitionCards {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 8), b.PlayerID())
			blocker, attacker := other, source
			if tc.blocking {
				blocker, attacker = source, other
			}
			canBlock := func(game *mage.Game) bool {
				return mage.CanBlock(game.FindPermanent(blocker.ID()), game.FindPermanent(attacker.ID()), game)
			}
			if !canBlock(g) {
				t.Fatal("initial block illegal")
			}
			clone := g.Clone()
			var modification mage.ContinuousModification
			switch tc.name {
			case "Bog Rats":
				modification = mage.AddSubtypeFamily(mage.This, mage.SubtypeCreature, "Wall")
			default:
				modification = mage.ModifyPT(mage.This, mage.Fixed(3), mage.Fixed(0))
			}
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, modification), other.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if tc.name == "Orcish Veteran" {
				if !canBlock(g) {
					t.Fatal("nonwhite high power illegal")
				}
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.White, core.Blue)), other.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if canBlock(g) {
				t.Fatal("filter ignored current layered characteristics")
			}
			if !canBlock(clone) {
				t.Fatal("clone shared characteristic change")
			}
			g.RunStepWithPriority(core.Cleanup)
			if !canBlock(g) {
				t.Fatal("filter retained expired characteristic change")
			}
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Flying)), attacker.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if canBlock(g) {
				t.Fatal("filter bypassed flying")
			}
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Reach)), blocker.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if !canBlock(g) {
				t.Fatal("legal reach blocker rejected")
			}
		})
	}
}

func TestRONBlockingProhibitionLifecycle(t *testing.T) {
	for _, tc := range blockingProhibitionCards {
		for _, change := range []string{"phase-cycle", "control", "return", "instances"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				otherCard := mage.NewCreature("Other", "", 4, 8, mage.WithSubTypes("Wall"))
				otherCard.SetColorOverride([]core.Color{core.White})
				other := g.PutOnBattlefield(otherCard, b.PlayerID())
				canBlock := func(game *mage.Game, s *mage.Permanent) bool {
					if tc.blocking {
						return mage.CanBlock(s, game.FindPermanent(other.ID()), game)
					}
					return mage.CanBlock(game.FindPermanent(other.ID()), s, game)
				}
				if canBlock(g, source) {
					t.Fatal("initial prohibition absent")
				}
				clone := g.Clone()
				switch change {
				case "phase-cycle":
					ids := g.PhaseOut(source.ID())
					g.ApplyContinuousEffects()
					g.PhaseIn(ids)
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
				case "return":
					g.RemoveFromBattlefield(source)
					unrelated := g.PutOnBattlefield(mage.NewCreature("Unrelated", "", 1, 8), a.PlayerID())
					if !canBlock(g, unrelated) {
						t.Fatal("departed source left a rule on another object")
					}
					source = g.PutOnBattlefield(card, b.PlayerID())
				case "instances":
					second := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), b.PlayerID())
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.EndOfTurn))
					if !canBlock(g, g.FindPermanent(source.ID())) || canBlock(g, second) {
						t.Fatal("instances shared source rule")
					}
				}
				g.ApplyContinuousEffects()
				if change != "instances" && canBlock(g, g.FindPermanent(source.ID())) {
					t.Fatal("source lost its rule")
				}
				if canBlock(clone, clone.FindPermanent(card.ID())) {
					t.Fatal("original change altered clone")
				}
			})
		}
	}
}

func TestRONBlockingProhibitionHost(t *testing.T) {
	src := `Card(Name:"Host rule",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),PermanentRuleEffect(HostPermanent,[CantBeBlockedByDefenders(IsCreatureType(Wall))])])`
	nodes, err := Parse(strings.NewReader(src), "host.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []string{"phase", "phase-cycle", "source-left", "source-return", "reattach", "host-return", "host-ability-removal", "source-ability-removal"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			auraCard := factory()
			auraCard.SetOwner(a.PlayerID())
			aura := g.PutOnBattlefield(auraCard, a.PlayerID())
			hostCard := mage.NewCreature("Host", "", 3, 6)
			hostCard.SetOwner(a.PlayerID())
			host := g.PutOnBattlefield(hostCard, a.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 3, 6), a.PlayerID())
			wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 8, mage.WithSubTypes("Wall")), b.PlayerID())
			g.Attach(aura.ID(), host.ID())
			if mage.CanBlock(wall, host, g) {
				t.Fatal("host rule absent")
			}
			clone := g.Clone()
			switch change {
			case "phase":
				g.PhaseOut(aura.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "source-left":
				g.RemoveFromBattlefield(aura)
			case "source-return":
				g.RemoveFromBattlefield(aura)
				aura = g.PutOnBattlefield(auraCard, a.PlayerID())
				g.Attach(aura.ID(), other.ID())
			case "reattach":
				g.Attach(aura.ID(), other.ID())
			case "host-return":
				g.RemoveFromBattlefield(host)
				host = g.PutOnBattlefield(hostCard, a.PlayerID())
			case "host-ability-removal":
				g.AddContinuousEffect(mage.RemoveAllAbilities(host.ID(), core.Indefinite))
			case "source-ability-removal":
				g.AddContinuousEffect(mage.RemoveAllAbilities(aura.ID(), core.Indefinite))
			}
			g.ApplyContinuousEffects()
			allowed := change != "phase-cycle" && change != "host-ability-removal"
			if mage.CanBlock(g.FindPermanent(wall.ID()), g.FindPermanent(host.ID()), g) != allowed {
				t.Fatal("wrong host lifecycle rule")
			}
			if mage.CanBlock(clone.FindPermanent(wall.ID()), clone.FindPermanent(host.ID()), clone) {
				t.Fatal("clone lost host rule")
			}
			if (change == "source-return" || change == "reattach") && mage.CanBlock(g.FindPermanent(wall.ID()), g.FindPermanent(other.ID()), g) {
				t.Fatal("new attachment lacks rule")
			}
		})
	}
}

func TestRONBlockingProhibitionDeclaredBlock(t *testing.T) {
	for _, tc := range blockingProhibitionCards {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
			otherCard := mage.NewCreature("Other", "", 1, 8)
			otherCard.SetColorOverride([]core.Color{core.White})
			other := g.PutOnBattlefield(otherCard, b.PlayerID())
			attacker, blocker, active := source, other, a.PlayerID()
			if tc.blocking {
				attacker, blocker, active = other, source, b.PlayerID()
			}
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(active, []uuid.UUID{attacker.ID()})
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: attacker.ID()}})
			if !g.GetCombat().IsBlocking(blocker.ID()) {
				t.Fatal("legal initial block rejected")
			}
			var modification = mage.ModifyPT(mage.This, mage.Fixed(3), mage.Fixed(0))
			if tc.name == "Bog Rats" {
				modification = mage.AddSubtypeFamily(mage.This, mage.SubtypeCreature, "Wall")
			}
			if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, modification), other.ID(), b.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(attacker.ID()), g) {
				t.Fatal("later filter change did not affect legality query")
			}
			if !g.GetCombat().IsBlocking(blocker.ID()) {
				t.Fatal("later filter change removed declared block")
			}
			g.ExecuteCombatDamage()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("declared block did not prevent player damage")
			}
		})
	}
}

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

func TestRONPlayerDamageHistoryCards(t *testing.T) {
	for _, name := range []string{"Simulacrum", "Reverse Polarity"} {
		for _, scenario := range []string{"zero", "normal", "late", "prevented", "departed", "returned", "phased", "copy", "clone", "illegal", "target-control", "target-phased", "target-returned", "outgoing-prevented"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				set := "LEB"
				if name == "Reverse Polarity" {
					set = "ATQ"
				}
				card := repositoryCard(t, set, name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				victimCard := mage.NewCreature("Victim", "", 1, 30)
				victimCard.SetOwner(a.PlayerID())
				victim := g.PutOnBattlefield(victimCard, a.PlayerID())
				sourceCard := mage.NewCreature("Artifact source", "", 1, 30, mage.WithCardType(core.TypeArtifact))
				sourceCard.SetOwner(b.PlayerID())
				source := g.PutOnBattlefield(sourceCard, b.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Other source", "", 1, 30), b.PlayerID())
				total, artifacts := 0, 0
				if scenario != "zero" {
					if scenario == "prevented" {
						if err := mage.ApplyEffect(g, mage.PreventNextDamage(mage.You, mage.Fixed(2)), source.ID(), a.PlayerID(), nil); err != nil {
							t.Fatal(err)
						}
					}
					captured := g.CaptureDamageSource(source.ID())
					switch scenario {
					case "departed", "returned":
						g.RemoveFromBattlefield(source)
						if scenario == "returned" {
							g.PutOnBattlefield(sourceCard, a.PlayerID())
						}
					case "phased":
						g.PhaseOut(source.ID())
					}
					g.DealSimultaneousDamage(mage.PlayerDamageAssignment(a, 3, captured), mage.PlayerDamageAssignment(a, 2, g.CaptureDamageSource(other.ID())), mage.PlayerDamageAssignment(b, 1, g.CaptureDamageSource(other.ID())))
					total, artifacts = 5, 3
					if scenario == "prevented" {
						total, artifacts = 3, 1
					}
				}
				if scenario == "outgoing-prevented" {
					binding := mage.DeclarePermanentTarget("victim", mage.IsCreature)
					shield := mage.NewInstant("Shield", "", mage.Spell(mage.PreventDamageThisTurn(binding, mage.PermanentFilter{}), mage.WithTargets(binding)))
					castRONSpell(t, g, a, shield, victim.ID())
					g.ResolveStack()
					a.ManaPool().Clear()
				}
				before := a.Life()
				var targets []uuid.UUID
				if name == "Simulacrum" {
					targets = []uuid.UUID{victim.ID()}
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err == nil {
					t.Fatal("unpaid cast accepted")
				}
				if len(a.Hand()) != 1 || g.StackSize() != 0 || a.Life() != before {
					t.Fatal("failed casting changed state")
				}
				if name == "Simulacrum" {
					a.ManaPool().Add(core.Black, 1)
					a.ManaPool().Add(core.Colorless, 1)
				} else {
					a.ManaPool().Add(core.White, 2)
				}
				if name == "Simulacrum" {
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{other.ID()}, 0); err == nil {
						t.Fatal("opponent creature accepted")
					}
					if a.ManaPool().TotalMana() != 2 || len(a.Hand()) != 1 {
						t.Fatal("illegal target paid cost")
					}
				}
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
					t.Fatal(err)
				}
				if a.Life() != before || g.FindPermanent(victim.ID()).Damage != 0 {
					t.Fatal("early resolution")
				}
				if scenario == "late" {
					g.DealDamageToPlayer(a, 1, other.ID())
					total++
					before--
				}
				if name == "Simulacrum" {
					switch scenario {
					case "illegal":
						g.MutablePermanent(victim.ID()).GrantBaseAttr(core.Shroud)
					case "target-control":
						if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ChangeController(mage.This, mage.You)), victim.ID(), b.PlayerID(), nil); err != nil {
							t.Fatal(err)
						}
					case "target-phased":
						g.PhaseOut(victim.ID())
					case "target-returned":
						g.RemoveFromBattlefield(victim)
						g.PutOnBattlefield(victimCard, a.PlayerID())
					}
				}
				copies := 1
				if scenario == "copy" {
					if g.CopySpellOnStack(card.ID(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
					copies++
				}
				clone := g.Clone()
				gain := total
				if name == "Reverse Polarity" {
					gain = 2 * artifacts
				}
				if name == "Simulacrum" && (scenario == "illegal" || strings.HasPrefix(scenario, "target-")) {
					gain = 0
				}
				g.ResolveStack()
				if a.Life() != before+copies*gain {
					t.Fatalf("life=%d want=%d", a.Life(), before+copies*gain)
				}
				live := g.FindPermanent(victim.ID())
				wantDamage := copies * gain
				if scenario == "outgoing-prevented" {
					wantDamage = 0
				}
				if name == "Simulacrum" && live != nil && live.Damage != wantDamage {
					t.Fatalf("damage=%d want=%d", live.Damage, wantDamage)
				}
				if clone.GetPlayer(a.PlayerID()).Life() != before {
					t.Fatal("resolution changed clone")
				}
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != before+copies*gain {
					t.Fatal("clone lost history or binding")
				}
			})
		}
	}
}

func TestRONPlayerDamageHistoryInvalid(t *testing.T) {
	for _, value := range []string{
		`DamageDealtToPlayerThisTurn`, `DamageDealtToPlayerThisTurn()`, `DamageDealtToPlayerThisTurn(You, You)`,
		`DamageDealtToPlayerThisTurn(You())`, `DamageDealtToPlayerThisTurn(Ref_TargetPlayer)`, `DamageDealtToPlayerThisTurn(ThisPermanent)`,
		`DamageDealtToPlayerThisTurnByPermanents(You)`, `DamageDealtToPlayerThisTurnByPermanents(You, IsCardtype(Creature))`,
		`DamageDealtToPlayerThisTurnByPermanents(You, IsCardtype(Artifact()))`, `DamageDealtToPlayerThisTurnByPermanents(Trigger_ThatPlayer, IsCardtype(Artifact))`,
	} {
		source := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([GainLife(` + value + `)]))])`
		nodes, err := Parse(strings.NewReader(source), "invalid-history.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", value)
		}
	}
}

func TestRONPlayerDamageHistoryBindings(t *testing.T) {
	for _, action := range []string{
		`PlayerAction(Ref_TargetPlayer,GainLife(DamageDealtToPlayerThisTurn(Ref_TargetPlayer)))`,
		`PlayerActions(Ref_TargetPlayer,[GainLife(DamageDealtToPlayerThisTurn(ActionPlayer))])`,
		`PlayerAction(Ref_TargetPlayer,GainLife(DamageDealtToPlayerThisTurnByPermanents(Ref_TargetPlayer,IsCardtype(Artifact))))`,
	} {
		t.Run(action, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([`+action+`])))`)
			g, a, b := ronGame()
			source := g.PutOnBattlefield(mage.NewArtifact("Source", ""), a.PlayerID())
			g.DealDamageToPlayer(a, 1, source.ID())
			g.DealDamageToPlayer(b, 3, source.ID())
			cards := []mage.Card{factory(), factory()}
			if cards[0].ID() == cards[1].ID() {
				t.Fatal("shared factory identity")
			}
			for i, player := range []*mage.BasePlayer{a, b} {
				card := cards[i]
				card.SetOwner(player.PlayerID())
				player.AddToHand(card)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				g.SetActivePlayerIndex(i)
				if err := g.CastSpellByID(player.PlayerID(), card.ID(), []uuid.UUID{player.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
			}
			g.ResolveTopOfStack()
			if a.Life() != 19 || b.Life() != 20 {
				t.Fatal("second factory read another target")
			}
			g.ResolveTopOfStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("first factory lost target binding")
			}
		})
	}
}

func TestRONPlayerDamageHistoryCopyController(t *testing.T) {
	for _, tc := range []struct {
		set, name    string
		wantA, wantB int
	}{
		{"ATQ", "Reverse Polarity", 22, 24},
		{"LEB", "Simulacrum", 20, 16},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(mage.NewArtifact("Source", ""), a.PlayerID())
			g.DealDamageToPlayer(a, 2, source.ID())
			g.DealDamageToPlayer(b, 4, source.ID())
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 20), a.PlayerID())
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.White, 2)
			a.ManaPool().Add(core.Black, 1)
			a.ManaPool().Add(core.Colorless, 1)
			var targets []uuid.UUID
			if tc.name == "Simulacrum" {
				targets = []uuid.UUID{victim.ID()}
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
				t.Fatal(err)
			}
			if g.CopySpellOnStack(card.ID(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			g.ResolveStack()
			if a.Life() != tc.wantA || b.Life() != tc.wantB {
				t.Fatalf("life=%d/%d want=%d/%d", a.Life(), b.Life(), tc.wantA, tc.wantB)
			}
		})
	}
}

func TestRONPlayerDamageHistoryRecordsAndInventory(t *testing.T) {
	files, err := os.ReadDir("../../../../cards/mtgish")
	if err != nil {
		t.Fatal(err)
	}
	selected := fstest.MapFS{}
	canonical := map[string]Node{}
	counts := map[string]int{}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".ron") {
			continue
		}
		data, err := os.ReadFile("../../../../cards/mtgish/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), file.Name())
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range nodes {
			name := n.Fields["Name"].String
			if name != "Simulacrum" && name != "Reverse Polarity" {
				continue
			}
			if old, ok := canonical[name]; ok {
				if definitionKey(old) != definitionKey(n) {
					t.Fatalf("reprints differ: %s", name)
				}
			} else {
				canonical[name] = n
			}
			factory, err := Compile(n)
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
			counts[name]++
			for line := range strings.SplitSeq(string(data), "\n") {
				if strings.Contains(line, `Name: "`+name+`"`) {
					selected[file.Name()+name+".ron"] = &fstest.MapFile{Data: []byte(line)}
				}
			}
		}
	}
	if counts["Simulacrum"] != 6 || counts["Reverse Polarity"] != 2 {
		t.Fatalf("reprints=%v", counts)
	}
	registry, err := LoadFS(selected, "*.ron")
	if err != nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatalf("strict loading=%v", err)
	}
	inventory, err := InspectFS(selected, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range inventory.Signatures {
		if entry.Role == "unknown" {
			t.Fatalf("unknown role in history records: %s", entry.Name)
		}
	}
	selected["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([GainLife(DamageDealtToPlayerThisTurn(Ref_TargetPlayer))]))])`)}
	if registry, err := LoadFS(selected, "*.ron"); err == nil || registry != nil {
		t.Fatal("invalid collection produced registry")
	}
}

func TestRONPlayerDamageHistoryDelayedBinding(t *testing.T) {
	factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([PlayerAction(Ref_TargetPlayer,GainLife(DamageDealtToPlayerThisTurn(Ref_TargetPlayer)))]))])))`)
	g, a, b := ronGame()
	source := g.PutOnBattlefield(mage.NewArtifact("Source", ""), a.PlayerID())
	g.DealDamageToPlayer(b, 2, source.ID())
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.DealDamageToPlayer(b, 1, source.ID())
	g.RunStepWithPriority(core.EndStep)
	if b.Life() != 20 || a.Life() != 20 {
		t.Fatal("delayed history lost bound player or read too early")
	}
}

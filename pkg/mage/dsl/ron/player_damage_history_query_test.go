package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPlayerDamageHistoryQueryCards(t *testing.T) {
	for _, name := range []string{"Simulacrum", "Reverse Polarity"} {
		sets := []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"}
		if name == "Reverse Polarity" {
			sets = []string{"ATQ", "3ED"}
		}
		for _, set := range sets {
			for _, state := range []string{"normal", "late", "prevented", "phase", "phase cycle", "returned", "shroud", "copy", "clone"} {
				t.Run(name+"/"+set+"/"+state, func(t *testing.T) {
					a := mage.NewBasePlayer("A")
					b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B")}
					g := mage.NewGame(a, b)
					g.SetStep(core.PrecombatMain)
					g.SetTurn(1)
					p := g.PutOnBattlefield(mage.NewCreature("target", "", 1, 30), a.PlayerID())
					q := g.PutOnBattlefield(mage.NewCreature("copy target", "", 1, 30), b.PlayerID())
					source := g.PutOnBattlefield(mage.NewCreature("source", "", 1, 30, mage.WithCardType(core.TypeArtifact)), b.PlayerID())
					if state == "prevented" {
						g.AddPreventionShield(a.PlayerID(), 2)
					}
					g.DealDamageToPlayer(a, 4, source.ID())
					g.DealDamageToPlayer(b, 1, source.ID())
					total := 4
					if state == "prevented" {
						total = 2
					}
					card := repositoryCard(t, set, name)
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					targets := []uuid.UUID(nil)
					if name == "Simulacrum" {
						targets = []uuid.UUID{p.ID()}
					}
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
						t.Fatal("unpaid cast changed state")
					}
					if name == "Simulacrum" {
						a.ManaPool().Add(core.Black, 1)
						a.ManaPool().Add(core.Colorless, 1)
					} else {
						a.ManaPool().Add(core.White, 2)
					}
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
						t.Fatal(err)
					}
					if state == "late" {
						g.DealDamageToPlayer(a, 1, source.ID())
						total++
					}
					if name == "Simulacrum" {
						switch state {
						case "phase":
							g.PhaseOut(p.ID())
						case "phase cycle":
							g.PhaseIn(g.PhaseOut(p.ID()))
						case "returned":
							g.RemoveFromBattlefield(p)
							g.PutOnBattlefield(p.Card, a.PlayerID())
						case "shroud":
							p.GrantBaseAttr(core.Shroud)
						}
					}
					obj := g.StackPeek()
					receiver := a.PlayerID()
					victim := p
					gain := total
					if name == "Reverse Polarity" {
						gain *= 2
					}
					if state == "copy" {
						b.targets = []uuid.UUID{q.ID()}
						obj = g.CopyStackObjectDirect(obj, b.PlayerID(), name == "Simulacrum")
						if obj == nil {
							t.Fatal("copy failed")
						}
						receiver = b.PlayerID()
						victim = q
						gain = 1
						if name == "Reverse Polarity" {
							gain = 2
						}
					}
					if name == "Simulacrum" && (state == "phase" || state == "returned" || state == "shroud") {
						gain = 0
					}
					predicate, ok := mage.InspectContextualPermanentCollection(mage.MatchingPermanents(mage.AnyPermanent))
					if !ok {
						t.Fatal("missing collection")
					}
					watched := predicate.Select(&mage.EffectContext{Game: g, Controller: a.PlayerID()})
					before := g.GetPlayer(receiver).Life()
					for _, game := range []*mage.Game{g, g.Clone()} {
						for range 2 {
							if certain, known := game.InspectSpellDestruction(game.StackPeek(), watched); certain || !known {
								t.Fatalf("history query=%v/%v", certain, known)
							}
						}
						if game.GetPlayer(receiver).Life() != before || game.FindPermanent(victim.ID()) != nil && game.FindPermanent(victim.ID()).Damage != 0 {
							t.Fatal("query executed")
						}
						if game != g {
							game.ResolveTopOfStack()
							if game.GetPlayer(receiver).Life() != before+gain || g.GetPlayer(receiver).Life() != before {
								t.Fatal("clone crossed state")
							}
						}
					}
					g.ResolveTopOfStack()
					if g.GetPlayer(receiver).Life() != before+gain {
						t.Fatalf("life=%d want=%d", g.GetPlayer(receiver).Life(), before+gain)
					}
					if name == "Simulacrum" && g.FindPermanent(victim.ID()) != nil && g.FindPermanent(victim.ID()).Damage != gain {
						t.Fatal("wrong history damage")
					}
				})
			}
		}
	}
}

func TestRONPlayerDamageHistoryQueryInvalid(t *testing.T) {
	for _, value := range []string{`DamageDealtToPlayerThisTurn()`, `DamageDealtToPlayerThisTurn(You())`, `DamageDealtToPlayerThisTurn(Ref_TargetPlayer)`, `DamageDealtToPlayerThisTurnByPermanents(You,IsCardtype(Creature))`, `Twice(DamageDealtToPlayerThisTurn(You),Integer(1))`, `Twice(DamageDealtToPlayerThisTurnByPermanents(You,IsCardtype(Artifact())))`} {
		for _, action := range []string{`GainLife(` + value + `)`, `SpellDealsDamage(ThisSpell,` + value + `,Player(You))`} {
			source := `Card(Name:"Invalid history query",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + action + `]))])`
			if _, err := Compile(nameScopeNode(t, source)); err == nil {
				t.Fatalf("accepted %s", action)
			}
		}
	}
}

func TestRONPlayerDamageHistoryQueryAdapter(t *testing.T) {
	n := nameScopeNode(t, `UnknownHistory(You)`)
	if _, err := compilePlayerDamageHistory(n, scope{}); err == nil {
		t.Fatal("unknown history constructor accepted")
	}
}

func TestRONPlayerDamageHistoryQueryBindings(t *testing.T) {
	for _, value := range []string{`DamageDealtToPlayerThisTurn(Ref_TargetPlayer)`, `Twice(DamageDealtToPlayerThisTurnByPermanents(Ref_TargetPlayer,IsCardtype(Artifact)))`, `DamageDealtToPlayerThisTurn(TheActivePlayer)`} {
		factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([GainLife(`+value+`)])))`)
		g, a, b := ronGame()
		source := g.PutOnBattlefield(mage.NewArtifact("source", ""), a.PlayerID())
		g.DealDamageToPlayer(a, 1, source.ID())
		g.DealDamageToPlayer(b, 3, source.ID())
		for _, target := range []uuid.UUID{a.PlayerID(), b.PlayerID()} {
			castRONSpell(t, g, a, factory(), target)
		}
		g.SetActivePlayerIndex(1)
		before := a.Life()
		gain := 4
		if value == `Twice(DamageDealtToPlayerThisTurnByPermanents(Ref_TargetPlayer,IsCardtype(Artifact)))` {
			gain = 8
		}
		if value == `DamageDealtToPlayerThisTurn(TheActivePlayer)` {
			gain = 6
		}
		for _, game := range []*mage.Game{g, g.Clone()} {
			for range 2 {
				if certain, known := game.InspectSpellDestruction(game.StackPeek(), nil); certain || !known {
					t.Fatal("bound history query failed")
				}
			}
			if game.GetPlayer(a.PlayerID()).Life() != before {
				t.Fatal("query changed life")
			}
			if game != g {
				game.ResolveStack()
				if game.GetPlayer(a.PlayerID()).Life() != before+gain || a.Life() != before {
					t.Fatal("clone binding crossed")
				}
			}
		}
		g.ResolveStack()
		if a.Life() != before+gain {
			t.Fatal("history factories crossed bindings")
		}
	}
}

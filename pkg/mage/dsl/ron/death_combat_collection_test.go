package ron

import (
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDeathCombatCollectionAbu(t *testing.T) {
	for _, role := range []string{"attacker", "blocker"} {
		for _, change := range []string{"normal", "source-return", "opponent-return", "phase", "phase-cycle", "control", "removed", "regenerate", "indestructible", "exile-replacement", "source-exile", "source-regenerate", "copy", "clone"} {
			t.Run(role+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "ARN", "Abu Ja'far")
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
					t.Fatal("unpaid cast accepted")
				}
				if len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("failed cast changed state")
				}
				a.ManaPool().Add(core.White, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				source := g.FindPermanent(card.ID())
				if source == nil || source.CurrentPower(g) != 0 || source.CurrentToughness(g) != 1 || a.ManaPool().TotalMana() != 0 {
					t.Fatal("paid cast failed")
				}
				opponentCard := mage.NewCreature("Opponent", "", 1, 10, mage.WithKeyword(core.Shroud))
				opponentCard.SetOwner(b.PlayerID())
				opponent := g.PutOnBattlefield(opponentCard, b.PlayerID())
				idle := g.PutOnBattlefield(mage.NewCreature("Idle", "", 1, 10), b.PlayerID())
				if role == "attacker" {
					source.RevokeBaseAttr(core.AttrSummonSick)
					g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{source.ID()})
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: opponent.ID(), AttackerID: source.ID()}})
				} else {
					opponent.RevokeBaseAttr(core.AttrSummonSick)
					g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{opponent.ID()})
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: source.ID(), AttackerID: opponent.ID()}})
				}
				if change == "source-exile" {
					g.AddExileIfWouldGoToGraveyardThisTurn(source.ID(), source.ID())
				}
				if change == "source-regenerate" {
					g.AddRegenerationShield(source.ID())
				}
				g.DestroyPermanent(source)
				g.PutTriggersOnStack()
				if change == "source-exile" || change == "source-regenerate" {
					if g.StackSize() != 0 || g.FindPermanent(opponent.ID()) == nil {
						t.Fatal("triggered without death")
					}
					return
				}
				if g.StackSize() != 1 || g.FindPermanent(opponent.ID()) == nil {
					t.Fatal("death timing or combat declaration")
				}
				switch change {
				case "source-return":
					a.RemoveFromGraveyard(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				case "opponent-return":
					g.BouncePermanentToHand(opponent)
					b.RemoveFromHand(opponentCard.ID())
					g.PutOnBattlefield(opponentCard, b.PlayerID())
				case "phase":
					g.PhaseOut(opponent.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(opponent.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: opponent.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				case "removed":
					g.RemoveFromCombat(opponent.ID())
				case "regenerate":
					g.AddRegenerationShield(opponent.ID())
				case "indestructible":
					g.MutablePermanent(opponent.ID()).GrantBaseAttr(core.Indestructible)
				case "exile-replacement":
					g.AddExileIfWouldGoToGraveyardThisTurn(opponent.ID(), source.ID())
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				pending := g.Clone()
				g.ResolveStack()
				if pending.FindPermanentIncludingPhased(opponent.ID()) == nil {
					t.Fatal("live resolution changed clone")
				}
				pending.ResolveStack()
				for _, game := range []*mage.Game{g, pending} {
					survives := change == "opponent-return" || change == "phase" || change == "indestructible"
					if (game.FindPermanentIncludingPhased(opponent.ID()) != nil) != survives {
						t.Fatal("wrong exact opponent destroyed")
					}
					if game.FindPermanent(idle.ID()) == nil {
						t.Fatal("unrelated creature destroyed")
					}
					if change == "source-return" && game.FindPermanent(card.ID()) == nil {
						t.Fatal("returned source destroyed")
					}
					if change == "exile-replacement" && len(game.GetExile()) == 0 {
						t.Fatal("destruction bypassed replacement")
					}
				}
			})
		}
	}
}

func TestRONDeathCombatCollectionInvalid(t *testing.T) {
	pair := `Or([BlockedDeadAttacker,WasBlockingDeadDefender])`
	for _, filter := range []string{`BlockedDeadAttacker`, `WasBlockingDeadDefender`, `Or([BlockedDeadAttacker(),WasBlockingDeadDefender])`, `Or([BlockedDeadAttacker,BlockedDeadAttacker])`, `Or([BlockedDeadAttacker,WasBlockingDeadDefender,AnyPermanent])`} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[TriggerA(WhenAPermanentDies(SinglePermanent(ThisPermanent)),ActionList([DestroyEachPermanentNoRegen(` + filter + `)]))])`
		if r, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil || r != nil {
			t.Fatalf("accepted %s", filter)
		}
	}
	for _, event := range []string{`WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent))`, `AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You))`, `WhenAPermanentDies(IsCardtype(Creature))`} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[TriggerA(` + event + `,ActionList([DestroyEachPermanentNoRegen(` + pair + `)]))])`
		if r, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil || r != nil {
			t.Fatalf("accepted event %s", event)
		}
	}
	reversed := strings.Replace(pair, `BlockedDeadAttacker,WasBlockingDeadDefender`, `WasBlockingDeadDefender,BlockedDeadAttacker`, 1)
	for _, filter := range []string{pair, reversed} {
		if _, err := compileContextualCollection(Node{Name: "BlockedDeadAttacker", Kind: Symbol}, scope{}); err == nil {
			t.Fatal("unbound relation accepted")
		}
		src := `Card(Name:"Good",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[TriggerA(WhenAPermanentDies(SinglePermanent(ThisPermanent)),ActionList([DestroyEachPermanentNoRegen(` + filter + `)]))])`
		if _, err := LoadFS(fstest.MapFS{"good.ron": {Data: []byte(src)}}, "*.ron"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRONDeathCombatCollectionMultiple(t *testing.T) {
	factory, err := Compile(hostUntapRecord(t, "ARN", "Abu Ja'far"))
	if err != nil {
		t.Fatal(err)
	}
	for _, cause := range []string{"combat", "sacrifice", "outside-combat", "no-longer-creature", "two-sources"} {
		t.Run(cause, func(t *testing.T) {
			g, a, b := ronGame()
			firstCard, secondCard := factory(), factory()
			if firstCard.ID() == secondCard.ID() || firstCard.Abilities()[0].AbilityID() == secondCard.Abilities()[0].AbilityID() {
				t.Fatal("factory shared card or ability identity")
			}
			first := g.PutOnBattlefield(firstCard, a.PlayerID())
			second := g.PutOnBattlefield(secondCard, a.PlayerID())
			one := g.PutOnBattlefield(mage.NewCreature("One", "", 1, 5), b.PlayerID())
			two := g.PutOnBattlefield(mage.NewCreature("Two", "", 1, 5), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 1, 5), b.PlayerID())
			first.RevokeBaseAttr(core.AttrSummonSick)
			second.RevokeBaseAttr(core.AttrSummonSick)
			if cause != "outside-combat" {
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: one.ID(), AttackerID: first.ID()}, {BlockerID: two.ID(), AttackerID: first.ID()}, {BlockerID: other.ID(), AttackerID: second.ID()}})
			}
			switch cause {
			case "combat":
				g.ExecuteCombatDamage()
			case "no-longer-creature":
				g.DoSacrifice(first)
				g.AddContinuousEffect(mage.TargetEffect(core.LayerType, core.Indefinite, one.ID(), func(game *mage.Game, p *mage.Permanent) error {
					game.GetEffects().RevokeAttr(p.ID(), core.AttrIsCreature)
					return nil
				}))
			default:
				g.DoSacrifice(first)
			}
			g.PutTriggersOnStack()
			if g.StackSize() == 0 || g.FindPermanent(one.ID()) == nil || g.FindPermanent(two.ID()) == nil {
				t.Fatal("death resolution timing")
			}
			if cause == "two-sources" {
				g.DoSacrifice(second)
				g.PutTriggersOnStack()
				if g.StackSize() != 2 {
					t.Fatal("separate source bindings lost")
				}
			}
			g.ResolveStack()
			if cause == "outside-combat" {
				if g.FindPermanent(one.ID()) == nil || g.FindPermanent(two.ID()) == nil {
					t.Fatal("invented combat opponents")
				}
				return
			}
			if (g.FindPermanent(one.ID()) != nil) != (cause == "no-longer-creature") || g.FindPermanent(two.ID()) != nil {
				t.Fatal("wrong multiple opponents")
			}
			if cause == "two-sources" || cause == "combat" {
				if g.FindPermanent(other.ID()) != nil {
					t.Fatal("second source lost its opponents")
				}
			} else if g.FindPermanent(other.ID()) == nil || g.FindPermanent(second.ID()) == nil {
				t.Fatal("one source used another source binding")
			}
		})
	}
}

func TestRONDeathCombatCollectionInventory(t *testing.T) {
	report, err := InspectFS(os.DirFS("../../../../cards/mtgish"), "ARN.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, entry := range report.Signatures {
		if slices.Contains(entry.Cards, "Abu Ja'far") && (entry.Name == "BlockedDeadAttacker" || entry.Name == "WasBlockingDeadDefender") && entry.Role == "death_combat_relation" && entry.CompiledOccurrences == 1 {
			seen[entry.Name] = true
		}
	}
	if len(seen) != 2 {
		t.Fatal("missing death combat relation roles", seen)
	}
}

package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDrawLifeNativeDomain(t *testing.T) {
	for _, rule := range []struct{ set, name string }{{"LEA", "Lich"}, {"LEA", "Island Sanctuary"}, {"LEG", "Chains of Mephistopheles"}, {"ARN", "Aladdin's Lamp"}, {"ARN", "Ring of Ma'rûf"}} {
		for _, destroy := range []string{"Stone Rain", "Wrath of God"} {
			t.Run(rule.name+"/"+destroy, func(t *testing.T) {
				a := &ronResolutionChooser{BasePlayer: mage.NewBasePlayer("A"), choice: 1}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetTurn(3)
				g.SetStep(core.PrecombatMain)
				seedRONDrawLibrary(a.BasePlayer)
				card := repositoryCard(t, rule.set, rule.name)
				p := g.PutOnBattlefield(card, a.PlayerID())
				outside := mage.NewLand("Outside")
				outside.SetOwner(a.PlayerID())
				if rule.name == "Ring of Ma'rûf" {
					if err := g.SetOutsideGameCards(a.PlayerID(), []mage.Card{outside}); err != nil {
						t.Fatal(err)
					}
				}
				if rule.name == "Aladdin's Lamp" || rule.name == "Ring of Ma'rûf" {
					a.ManaPool().Add(core.Colorless, 5)
					g.SetXValue(3)
					if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
				}
				victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 2, 3), b.PlayerID())
				if destroy == "Stone Rain" {
					victim = g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID())
				}
				var targets []uuid.UUID
				if destroy == "Stone Rain" {
					targets = []uuid.UUID{victim.ID()}
				}
				spell := castRONSpell(t, g, a.BasePlayer, repositoryCard(t, "LEA", destroy), targets...)
				instructions, ok := mage.InspectDirectDestruction(spell.Effects[0])
				if !ok || len(instructions) != 1 {
					t.Fatal("missing instruction")
				}
				var visits []mage.PermanentVisit
				if instructions[0].Subject != nil {
					query, ok := mage.InspectPermanentTarget(instructions[0].Subject)
					if !ok {
						t.Fatal("missing target")
					}
					visits, ok = query.Select(g, spell)
					if !ok {
						t.Fatal("unknown target")
					}
				} else {
					query, ok := mage.InspectPermanentCollection(instructions[0].Subjects)
					if !ok {
						t.Fatal("missing collection")
					}
					visits = query.Select(g)
				}
				if len(visits) != 1 {
					t.Fatal("wrong subjects", len(visits))
				}
				for _, game := range []*mage.Game{g.Clone(), g} {
					for range 2 {
						for _, visit := range visits {
							if got, known := game.InspectDestructionDestinations(visit, instructions[0].CannotRegenerate); !known || got != mage.DestructionToGraveyard {
								t.Fatalf("destination %v known %v", got, known)
							}
							if proceeds, known := game.InspectDestructionProposal(visit, instructions[0].CannotRegenerate); !known || !proceeds {
								t.Fatal("unknown proposal")
							}
						}
					}
					if a.prompts != 0 || game.StackSize() != 1 || len(game.GetPlayer(a.PlayerID()).Hand()) != 0 {
						t.Fatal("inspection executed draw body")
					}
					player := game.GetPlayer(a.PlayerID())
					switch rule.name {
					case "Lich":
						game.PlayerGainLife(player, 2)
						if player.Life() != 0 || len(player.Hand()) != 2 {
							t.Fatal("lost life gain replacement")
						}
					case "Island Sanctuary":
						game.SetStep(core.Draw)
						game.PlayerDrawCard(player)
						if game == g && (a.prompts != 1 || len(player.Hand()) != 1) {
							t.Fatal("lost draw-step choice")
						}
						game.SetStep(core.PrecombatMain)
					case "Chains of Mephistopheles":
						game.PlayerDrawCard(player)
						if len(player.Hand()) != 0 || len(player.Graveyard()) != 1 {
							t.Fatal("lost empty-hand mill replacement")
						}
					case "Aladdin's Lamp":
						game.PlayerDrawCard(player)
						if len(player.Hand()) != 1 || len(game.LibraryLooks(a.PlayerID())) != 3 {
							t.Fatal("inspection consumed future draw or lost X")
						}
					case "Ring of Ma'rûf":
						game.PlayerDrawCard(player)
						if len(player.Hand()) != 1 || player.Hand()[0].ID() != outside.ID() || len(game.OutsideGameCards(a.PlayerID())) != 0 {
							t.Fatal("lost outside-game replacement")
						}
					}
					game.ResolveStack()
					if game.FindPermanent(victim.ID()) != nil {
						t.Fatal("native destruction disagrees")
					}
				}
			})
		}
	}
}

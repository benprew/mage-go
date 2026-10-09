package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONAnteExchangeDarkpact(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := staticBaseRecord(t, set, "Darkpact")
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			for _, state := range []string{"normal", "empty", "foreign", "returned", "departed", "clone", "copy"} {
				t.Run(state, func(t *testing.T) {
					a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
					g, _ := mage.NewGameWithAnte(a, b, nil, nil)
					g.SetStep(core.PrecombatMain)
					g.SetTurn(1)
					ante := mage.NewArtifact("ante", "")
					ante.SetOwner(b.PlayerID())
					b.AddToAnte(ante)
					top, next := mage.NewArtifact("top", ""), mage.NewArtifact("next", "")
					top.SetOwner(a.PlayerID())
					next.SetOwner(a.PlayerID())
					a.SetLibrary([]mage.Card{top, next})
					if state == "empty" {
						a.SetLibrary(nil)
					}
					if state == "foreign" {
						top.SetOwner(b.PlayerID())
					}
					card := factory()
					card.SetOwner(a.PlayerID())
					a.AddToHand(card)
					a.ManaPool().Add(core.Black, 2)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{ante.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 2 || g.StackSize() != 0 || b.Ante()[0].Owner() != b.PlayerID() {
						t.Fatal("failed payment changed state")
					}
					a.ManaPool().Add(core.Black, 1)
					if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{ante.ID()}, 0); err != nil {
						t.Fatal(err)
					}
					if b.Ante()[0].Owner() != b.PlayerID() || a.ManaPool().TotalMana() != 0 {
						t.Fatal("wrong announcement timing")
					}
					if state == "returned" || state == "departed" {
						removed, err := g.RemoveFromAnte(ante.ID())
						if err != nil {
							t.Fatal(err)
						}
						if state == "returned" {
							b.AddToAnte(removed)
						} else {
							b.AddToHand(removed)
						}
					}
					if state == "clone" {
						clone := g.Clone()
						clone.ResolveStack()
						if clone.GetPlayer(a.PlayerID()).Library()[0].ID() != ante.ID() || a.Library()[0].ID() != top.ID() || b.Ante()[0].Owner() != b.PlayerID() {
							t.Fatal("clone resolution leaked")
						}
					}
					if state == "copy" {
						otherTop := mage.NewArtifact("B top", "")
						otherTop.SetOwner(b.PlayerID())
						b.SetLibrary([]mage.Card{otherTop})
						if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
							t.Fatal("copy failed")
						}
					}
					g.ResolveStack()
					if state == "normal" || state == "empty" || state == "clone" {
						if a.Library()[0].ID() != ante.ID() || a.Library()[0].Owner() != a.PlayerID() || len(b.Ante()) != 0 {
							t.Fatal("exchange failed")
						}
						if state != "empty" && (len(a.Ante()) != 1 || a.Ante()[0].ID() != top.ID() || a.Library()[1].ID() != next.ID()) {
							t.Fatal("other half failed")
						}
					} else if state == "foreign" {
						if b.Ante()[0].Owner() != a.PlayerID() || a.Library()[0].ID() != top.ID() {
							t.Fatal("ownership depended on exchange")
						}
					} else if state == "copy" {
						if b.Library()[0].ID() != ante.ID() || a.Library()[0].ID() != top.ID() || len(b.Ante()) != 1 {
							t.Fatal("copy reused original controller or original followed departed target")
						}
					} else if g.FindCardAnywhere(ante.ID()).Owner() != b.PlayerID() || a.Library()[0].ID() != top.ID() {
						t.Fatal("illegal target changed state")
					}
					if len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != card.ID() {
						t.Fatal("spell destination")
					}
				})
			}
		})
	}
}

func TestRONAnteExchangeDeclarations(t *testing.T) {
	for _, action := range []string{
		`AcquireOwnershipOfCard`, `AcquireOwnershipOfCard(Ref_TargetAnteCard)`, `AcquireOwnershipOfCard(Ref_TargetAnteCard,You,You)`,
		`AcquireOwnershipOfCard(Ref_TargetPermanent,You)`, `AcquireOwnershipOfCard(Ref_TargetAnteCard,Ref_TargetPlayer)`,
		`ExchangeAnteCardWithTopCardOfPlayersLibrary(Ref_TargetAnteCard)`, `ExchangeAnteCardWithTopCardOfPlayersLibrary(Ref_TargetAnteCard,You,You)`,
		`ExchangeAnteCardWithTopCardOfPlayersLibrary(Ref_TargetGraveyardCard,You)`, `ExchangeAnteCardWithTopCardOfPlayersLibrary(Ref_TargetAnteCard,ThisPermanent)`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(Targeted([TargetAnteCard(AnyCard)],ActionList([`+action+`])))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", action)
		}
	}
}

func TestRONAnteExchangeContextsAndInstances(t *testing.T) {
	body := `Targeted([TargetAnteCard(AnyCard)],ActionList([AcquireOwnershipOfCard(Ref_TargetAnteCard,You),ExchangeAnteCardWithTopCardOfPlayersLibrary(Ref_TargetAnteCard,You)]))`
	for _, state := range []string{"activation", "phase", "returned-source", "trigger"} {
		t.Run(state, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g, _ := mage.NewGameWithAnte(a, b, nil, nil)
			first, second := mage.NewArtifact("first", ""), mage.NewArtifact("second", "")
			first.SetOwner(b.PlayerID())
			second.SetOwner(a.PlayerID())
			a.AddToAnte(first)
			b.AddToAnte(second)
			rule := `Activated(PayMana([]),` + body + `)`
			if state == "trigger" {
				rule = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),` + body + `)`
			}
			factory := permanentScopeFactory(t, "Artifact", rule)
			one, two := factory(), factory()
			one.SetOwner(a.PlayerID())
			two.SetOwner(b.PlayerID())
			p := g.PutOnBattlefield(one, a.PlayerID())
			q := g.PutOnBattlefield(two, b.PlayerID())
			if state == "trigger" {
				g.PutTriggersOnStack()
				if g.StackSize() != 2 {
					t.Fatal("entry triggers missing")
				}
				g.ResolveStack()
				if len(a.Library())+len(b.Library()) != 1 {
					t.Fatal("independent triggers followed moved target")
				}
				return
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(b.PlayerID(), q.ID(), 0, []uuid.UUID{second.ID()}); err != nil {
				t.Fatal(err)
			}
			if state == "phase" {
				g.PhaseOut(p.ID())
				g.PhaseOut(q.ID())
			}
			if state == "returned-source" {
				g.BouncePermanentToHand(p)
				a.RemoveFromHand(one.ID())
				g.PutOnBattlefield(one, a.PlayerID())
			}
			g.ResolveStack()
			if len(a.Library()) != 1 || a.Library()[0].ID() != first.ID() || len(b.Library()) != 1 || b.Library()[0].ID() != second.ID() {
				t.Fatal("instances shared targets or lost pending source")
			}
		})
	}
}

func TestRONAnteExchangeRolesAndInventory(t *testing.T) {
	action := `AcquireOwnershipOfCard(Ref_TargetAnteCard,You)`
	exchange := `ExchangeAnteCardWithTopCardOfPlayersLibrary(Ref_TargetAnteCard,You)`
	for _, rule := range []string{
		`SpellActions(ActionList([` + action + `]))`,
		`SpellActions(Targeted([TargetGraveyardCard(IsCardtype(Creature))],ActionList([AcquireOwnershipOfCard(Ref_TargetGraveyardCard,You)])))`,
		`Activated(` + exchange + `,ActionList([DrawACard]))`,
		`AsPermanentEnters(ThisPermanent,[` + action + `])`,
		`PermanentLayerEffect(ThisPermanent,[` + exchange + `])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	entries := map[string]*InventorySignature{}
	walkInventory(staticBaseRecord(t, "LEA", "Darkpact"), "Darkpact", "Card", "card", "collection", true, entries)
	roles := map[string]string{}
	for _, entry := range entries {
		roles[entry.Name] = entry.Role
	}
	if roles["AcquireOwnershipOfCard"] != "action" || roles["Ref_TargetAnteCard"] != "card_reference" || roles["You"] != "player_reference" {
		t.Fatal("wrong roles", roles)
	}
}

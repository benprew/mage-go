package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONContextualCollectionQuery(t *testing.T) {
	for _, syntax := range []string{
		`ControlledByAPlayer(SinglePlayer(You))`, `OwnedByAPlayer(SinglePlayer(You))`, `ControlledByPlayer(TheActivePlayer)`,
		`And([IsCardtype(Land),ControlledByAPlayer(SinglePlayer(You))])`,
		`Or([And([IsColor(Black),ControlledByAPlayer(SinglePlayer(You))]),OwnedByAPlayer(SinglePlayer(TheActivePlayer))])`,
	} {
		build, err := compileDestructionCollection(nameScopeNode(t, syntax), scope{resolving: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := mage.InspectContextualPermanentCollection(build(instanceBindings{})); !ok {
			t.Fatalf("lost declaration: %s", syntax)
		}
	}
	for _, syntax := range []string{
		`ControlledByAPlayer()`, `ControlledByAPlayer(SinglePlayer())`, `ControlledByAPlayer(SinglePlayer(You,You))`,
		`ControlledByAPlayer(SinglePlayer(ActionPlayer))`, `OwnedByAPlayer(SinglePlayer(Ref_TargetPlayer))`,
		`ControlledByPlayer(You)`, `ControlledByPlayer(TheActivePlayer())`, `And([])`, `Or([IsCardtype(Land),And([])])`,
	} {
		if _, err := compileDestructionCollection(nameScopeNode(t, syntax), scope{resolving: true}); err == nil {
			t.Fatalf("accepted %s", syntax)
		}
	}
	record := repositoryRecord(t, "LEG", "Equinox", "LEG.ron")
	query := contextualQueryNode(t, record, "WouldDestroyAPermanent").Args[0]
	build, err := compileDestructionCollection(query, scope{resolving: true})
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	g.PutOnBattlefield(mage.NewLand("A"), a.PlayerID())
	target := g.PutOnBattlefield(mage.NewLand("B"), b.PlayerID())
	plan, ok := mage.InspectContextualPermanentCollection(build(instanceBindings{}))
	if !ok {
		t.Fatal("Equinox lost its collection")
	}
	visits := plan.Select(&mage.EffectContext{Game: g, Controller: b.PlayerID()})
	if len(visits) != 1 || visits[0].ID != target.ID() {
		t.Fatal("Equinox used the wrong controller")
	}
	remove := repositoryRecord(t, "LEG", "Remove Enchantments", "LEG.ron")
	full := contextualQueryNode(t, remove, "DestroyEachPermanent").Args[0]
	fullBuild, err := compileDestructionCollection(full, scope{resolving: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := mage.InspectContextualPermanentCollection(fullBuild(instanceBindings{})); !ok {
		t.Fatal("complete attachment branch lost its query")
	}
	if _, err := Compile(record); err == nil {
		t.Fatal("incomplete prospective query accepted")
	}
}

func TestRONContextualCollectionQueryGameplay(t *testing.T) {
	for _, name := range []string{"Reset", "Remove Enchantments"} {
		for _, state := range []string{"normal", "phase", "return", "control"} {
			t.Run(name+"/"+state, func(t *testing.T) {
				g, a, b := ronGame()
				g.SetActivePlayerIndex(0)
				g.SetStep(core.Upkeep)
				g.SetStep(core.Draw)
				var card mage.Card = mage.NewLand("Member")
				if name == "Remove Enchantments" {
					card = mage.NewEnchantment("Member", "")
				}
				card.SetOwner(a.PlayerID())
				member := g.PutOnBattlefield(card, b.PlayerID())
				g.TapPermanent(member)
				var otherCard mage.Card = mage.NewLand("Other")
				if name == "Remove Enchantments" {
					otherCard = mage.NewEnchantment("Other", "")
				}
				otherCard.SetOwner(a.PlayerID())
				other := g.PutOnBattlefield(otherCard, a.PlayerID())
				g.TapPermanent(other)
				spell := castRONSpell(t, g, b, repositoryCard(t, "LEG", name))
				if state == "phase" {
					g.PhaseOut(member.ID())
				}
				if state == "return" {
					g.RemoveFromBattlefield(member)
					member = g.PutOnBattlefield(card, b.PlayerID())
					g.TapPermanent(member)
				}
				if state == "control" {
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: member.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				}
				record := repositoryRecord(t, "LEG", name, "LEG.ron")
				actionName := "UntapEachPermanent"
				if name == "Remove Enchantments" {
					actionName = "DestroyEachPermanent"
				}
				collection := contextualQueryNode(t, record, actionName).Args[0]
				if name == "Remove Enchantments" {
					collection = collection.Args[0].Args[0]
				}
				build, err := compileDestructionCollection(collection, scope{resolving: true})
				if err != nil {
					t.Fatal(err)
				}
				plan, ok := mage.InspectContextualPermanentCollection(build(instanceBindings{}))
				if !ok {
					t.Fatal("lost real-record controller collection")
				}
				visits := plan.Select(&mage.EffectContext{Game: g, Controller: spell.Controller})
				selected := state != "phase" && state != "control"
				if selected && (len(visits) != 1 || visits[0].ID != member.ID()) || !selected && len(visits) != 0 {
					t.Fatalf("wrong visits %v", visits)
				}
				clone := g.Clone()
				clone.ResolveStack()
				g.ResolveStack()
				for _, game := range []*mage.Game{g, clone} {
					p := game.FindPermanentIncludingPhased(member.ID())
					if name == "Reset" {
						if p == nil || p.Tapped == selected {
							t.Fatal("native untap disagrees")
						}
					} else if (p == nil) != selected {
						t.Fatal("native destruction disagrees")
					}
					if game.FindPermanent(other.ID()) == nil || !game.FindPermanent(other.ID()).Tapped {
						t.Fatal("wrong controller affected")
					}
				}
			})
		}
	}
}

func contextualQueryNode(t *testing.T, root Node, name string) Node {
	t.Helper()
	var find func(Node) (Node, bool)
	find = func(n Node) (Node, bool) {
		if n.Name == name {
			return n, true
		}
		for _, child := range n.Args {
			if got, ok := find(child); ok {
				return got, true
			}
		}
		for _, child := range n.Fields {
			if got, ok := find(child); ok {
				return got, true
			}
		}
		return Node{}, false
	}
	if got, ok := find(root); ok {
		return got
	}
	t.Fatalf("missing %s", name)
	return Node{}
}

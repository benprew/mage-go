package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONAttachmentCollectionQuery(t *testing.T) {
	for _, syntax := range []string{
		`AttachedToAPermanent(IsCardtype(Land))`,
		`AttachedToAPermanent(ControlledByAPlayer(SinglePlayer(You)))`,
		`And([IsEnchantmentType(Aura),AttachedToAPermanent(And([IsAttacking,IsCardtype(Creature),ControlledByAPlayer(Opponent)]))])`,
		`AttachedToAPermanent(AttachedToAPermanent(Or([IsCardtype(Land),ControlledByPlayer(TheActivePlayer)])))`,
	} {
		build, err := compileDestructionCollection(nameScopeNode(t, syntax), scope{resolving: true})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := mage.InspectContextualPermanentCollection(build(instanceBindings{})); !ok {
			t.Fatalf("lost attachment declaration: %s", syntax)
		}
	}
	for _, syntax := range []string{
		`AttachedToAPermanent`, `AttachedToAPermanent()`, `AttachedToAPermanent(IsCardtype(Land),IsAttacking)`,
		`AttachedToAPermanent(You)`, `AttachedToAPermanent(And([]))`,
		`AttachedToAPermanent(Or([IsCardtype(Land),Unknown]))`,
		`AttachedToAPermanent(SinglePermanent(Ref_TargetPermanent))`,
		`AttachedToAPermanent(ControlledByAPlayer(SinglePlayer(ActionPlayer)))`,
	} {
		if _, err := compileDestructionCollection(nameScopeNode(t, syntax), scope{resolving: true}); err == nil {
			t.Fatalf("accepted invalid attachment collection %s", syntax)
		}
	}
	build, err := compileDestructionCollection(nameScopeNode(t, `AttachedToAPermanent(SinglePermanent(ThisPermanent))`), scope{permanent: true, resolving: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := mage.InspectContextualPermanentCollection(build(instanceBindings{})); ok {
		t.Fatal("source-bound host crossed pure boundary")
	}
}

func TestRONAttachmentCollectionQueryGameplay(t *testing.T) {
	record := repositoryRecord(t, "LEG", "Remove Enchantments", "LEG.ron")
	collection := contextualQueryNode(t, record, "DestroyEachPermanent").Args[0]
	build, err := compileDestructionCollection(collection, scope{resolving: true})
	if err != nil {
		t.Fatal(err)
	}
	plan, ok := mage.InspectContextualPermanentCollection(build(instanceBindings{}))
	if !ok {
		t.Fatal("complete real destruction collection lost attachment declaration")
	}
	for _, state := range []string{"normal", "move", "phased-host", "phased-aura", "cycle", "removed", "control"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetStep(core.DeclareBlockers)
			host := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 4), b.PlayerID())
			idle := g.PutOnBattlefield(mage.NewCreature("Idle", "", 2, 4), b.PlayerID())
			g.GetCombat().AddAttacker(host.ID(), a.PlayerID())
			auraCard := mage.NewAura("Foreign aura", "")
			auraCard.SetOwner(b.PlayerID())
			aura := g.PutOnBattlefield(auraCard, b.PlayerID())
			g.Attach(aura.ID(), host.ID())
			ownedCard := mage.NewAura("Owned aura", "")
			ownedCard.SetOwner(a.PlayerID())
			owned := g.PutOnBattlefield(ownedCard, b.PlayerID())
			g.Attach(owned.ID(), host.ID())
			castRONSpell(t, g, a, repositoryCard(t, "LEG", "Remove Enchantments"))
			switch state {
			case "move":
				g.Attach(aura.ID(), idle.ID())
				g.Attach(owned.ID(), idle.ID())
			case "phased-host":
				g.PhaseOut(host.ID())
			case "phased-aura":
				g.PhaseOut(aura.ID())
				g.PhaseOut(owned.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "removed":
				g.RemoveFromCombat(host.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{TargetID: host.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			selected := state == "normal" || state == "control"
			for _, game := range []*mage.Game{g, g.Clone()} {
				game.ApplyContinuousEffects()
				ctx := &mage.EffectContext{Game: game, Controller: a.PlayerID()}
				visits := plan.Select(ctx)
				if selected && (len(visits) != 2 || visits[0].ID != aura.ID() || visits[1].ID != owned.ID()) || !selected && len(visits) != 0 {
					t.Fatalf("wrong real-record attachment visits %v", visits)
				}
				game.ResolveStack()
				if (game.FindPermanentIncludingPhased(aura.ID()) == nil) != selected {
					t.Fatal("native destruction disagrees with attachment selection")
				}
				if (game.FindPermanentIncludingPhased(owned.ID()) == nil) != selected {
					t.Fatal("owned Aura return disagrees with host collection")
				}
				if selected {
					if len(game.GetPlayer(b.PlayerID()).Graveyard()) != 1 || len(game.GetPlayer(a.PlayerID()).Hand()) != 1 {
						t.Fatal("return and destruction order changed")
					}
					if len(plan.Select(ctx)) != 0 {
						t.Fatal("query retained departed attachments")
					}
				}
				if game.FindPermanentIncludingPhased(host.ID()) == nil || game.FindPermanent(idle.ID()) == nil {
					t.Fatal("attachment selection affected hosts")
				}
			}
		})
	}
}

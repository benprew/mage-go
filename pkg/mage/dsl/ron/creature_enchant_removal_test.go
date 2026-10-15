package ron

import (
	"testing"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCreatureEnchantRemoval(t *testing.T) {
	factory, err := Compile(repositoryRecord(t, "LEG", "Takklemaggot", "enchant-removal.ron"))
	if err != nil {
		t.Fatal(err)
	}
	for _, choice := range []bool{false, true} {
		t.Run(map[bool]string{false: "non-aura", true: "attached"}[choice], func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := mage.NewCreature("Host", "", 2, 2)
			hostCard.SetOwner(a.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			card := factory()
			card.SetOwner(a.PlayerID())
			aura := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if host.CurrentToughness(g) != 1 {
				t.Fatal("host upkeep counter missing")
			}
			var next *mage.Permanent
			if choice {
				next = g.PutOnBattlefield(mage.NewCreature("Next", "", 3, 3), a.PlayerID())
				next.GrantBaseAttr(core.Shroud)
			}
			g.DestroyPermanent(host)
			g.CheckStateBasedActions()
			g.PutTriggersOnStack()
			g.ResolveStack()
			returned := g.FindPermanent(card.ID())
			if returned == nil {
				t.Fatal("return failed")
			}
			if returned.HasCreatureEnchantAbility() != choice || returned.HasSubType("Aura") != choice {
				t.Fatal("wrong returned enchant declaration")
			}
			if choice && returned.AttachedTo != next.ID() {
				t.Fatal("required untargeted choice failed")
			}
			g.CheckStateBasedActions()
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if choice {
				if next.CurrentToughness(g) != 3 || b.Life() != 20 {
					t.Fatal("attached branch gained fallback grant")
				}
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
				g.PutTriggersOnStack()
				g.ResolveStack()
				if g.FindPermanent(next.ID()).CurrentToughness(g) != 2 {
					t.Fatal("reattached upkeep counter missing")
				}
			} else if b.Life() != 19 {
				t.Fatal("fallback upkeep damage missing")
			}
		})
	}
}

func TestRONCreatureEnchantRemovalInvalid(t *testing.T) {
	for _, form := range []string{`LosesAbility(Enchant)`, `LosesAbility(Enchant())`, `LosesAbility(Enchant(IsCardtype(Land)))`, `LosesAbility(Enchant(IsCardtype(Creature())))`, `LosesAbility(Enchant(IsCardtype(Creature)),Flying)`} {
		n := nameScopeNode(t, `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffect(ThisPermanent,[`+form+`])]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatalf("invalid declaration accepted: %s", form)
		}
	}
}

func TestRONCreatureEnchantRemovalLifecycle(t *testing.T) {
	for _, state := range []string{"normal", "control", "phase", "cycle", "return", "clone", "copy"} {
		t.Run(state, func(t *testing.T) {
			factory, err := Compile(repositoryRecord(t, "LEG", "Takklemaggot", "complete-enchant.ron"))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			hostCard := mage.NewCreature("Host", "", 2, 2)
			hostCard.SetOwner(a.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			card := factory()
			card.SetOwner(a.PlayerID())
			aura := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			g.DestroyPermanent(host)
			g.CheckStateBasedActions()
			g.PutTriggersOnStack()
			if state == "copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("stack copy failed")
			}
			if state == "clone" {
				original := g
				g = g.Clone()
				g.ResolveStack()
				if original.FindPermanent(card.ID()) != nil {
					t.Fatal("pending clone changed original")
				}
			} else {
				g.ResolveStack()
			}
			p := g.FindPermanent(card.ID())
			if p == nil || p.HasSubType("Aura") || p.HasCreatureEnchantAbility() {
				t.Fatal("fallback return failed")
			}
			want := 19
			switch state {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: p.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(p.ID())
				want = 20
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "return":
				g.BouncePermanentToHand(p)
				owner := g.GetPlayer(a.PlayerID())
				c, _ := owner.RemoveFromHand(card.ID())
				g.PutOnBattlefield(c, a.PlayerID())
				want = 20
			}
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
			g.PutTriggersOnStack()
			g.ResolveStack()
			if g.GetPlayer(a.PlayerID()).Life() != 20 || g.GetPlayer(b.PlayerID()).Life() != want {
				t.Fatalf("grant followed wrong player: A=%d B=%d want B=%d", g.GetPlayer(a.PlayerID()).Life(), g.GetPlayer(b.PlayerID()).Life(), want)
			}
		})
	}
}

func TestRONCreatureEnchantRemovalContexts(t *testing.T) {
	n := nameScopeNode(t, `CreatePermanentLayerEffect(ThisPermanent,[LosesAbility(Enchant(IsCardtype(Creature)))])`)
	for _, ctx := range []scope{{permanent: true, enteringSource: true}, {permanent: true, drawReplacementBody: true}} {
		if _, err := compileIndefiniteType(n, ctx); err == nil {
			t.Fatal("replacement context accepted")
		}
	}
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreatePermanentLayerEffect(ThisPermanent,[LosesAbility(Enchant(IsCardtype(Creature)))])]))`)
	g, a, _ := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if p.HasCreatureEnchantAbility() {
		t.Fatal("removal granted an absent ability")
	}
}

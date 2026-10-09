package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONAnteReferenceGameplay(t *testing.T) {
	for _, state := range []string{"live", "owner", "departed", "returned", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
			g, _ := mage.NewGameWithAnte(a, b, nil, nil)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			ante := mage.NewArtifact("ante", "")
			ante.SetOwner(b.PlayerID())
			b.AddToAnte(ante)
			factory := permanentScopeFactory(t, "Sorcery", `SpellActions(Targeted([TargetAnteCard(AnyCard)],ActionList([GainLife(Integer(3))])))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{ante.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 20 {
				t.Fatal("effect before resolution")
			}
			switch state {
			case "owner":
				if err := g.ChangeOwner(ante.ID(), a.PlayerID()); err != nil {
					t.Fatal(err)
				}
			case "departed", "returned":
				removed, err := g.RemoveFromAnte(ante.ID(), core.ZoneHand)
				if err != nil {
					t.Fatal(err)
				}
				if state == "returned" {
					b.AddToAnte(removed)
				} else {
					b.AddToHand(removed)
				}
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "clone":
				clone := g.Clone()
				clone.ResolveStack()
				if clone.GetPlayer(a.PlayerID()).Life() != 23 || a.Life() != 20 || g.StackSize() != 1 {
					t.Fatal("clone resolution leaked")
				}
			}
			g.ResolveStack()
			want := 23
			if state == "departed" || state == "returned" {
				want = 20
			}
			if a.Life() != want {
				t.Fatalf("life=%d want=%d", a.Life(), want)
			}
			if state == "copy" && b.Life() != 23 {
				t.Fatal("copy lost ante target")
			}
		})
	}
}

func TestRONAnteReferenceDeclarations(t *testing.T) {
	for _, target := range []string{`TargetAnteCard`, `TargetAnteCard()`, `TargetAnteCard(AnyCard())`, `TargetAnteCard(AnyCard,AnyCard)`, `TargetAnteCard(OwnedByAPlayer(You))`, `TargetAnteCard(AnyPermanent)`} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(Targeted([`+target+`],ActionList([DrawACard])))])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", target)
		}
	}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		record := staticBaseRecord(t, set, "Darkpact")
		if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "ExchangeAnteCardWithTopCardOfPlayersLibrary") {
			t.Fatalf("wrong Darkpact blocker: %v", err)
		}
		target := record.Fields["Rules"].Args[1].Args[0].Args[0].Args[0]
		_, binding, err := compileTarget(target, scope{})
		if err != nil || binding != "Ref_TargetAnteCard" {
			t.Fatalf("real target rejected: %v", err)
		}
		ctx := scope{cardTarget: &cardTargetPlan{name: binding, ante: true}}
		ref := record.Fields["Rules"].Args[1].Args[0].Args[1].Args[0].Args[0].Args[0]
		if _, err := compileCardRef(ref, ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := compileCardRef(ref, scope{}); err == nil {
			t.Fatal("unbound reference accepted")
		}
	}
}

func TestRONAnteReferenceZoneBoundaries(t *testing.T) {
	for _, action := range []string{
		`PutGraveyardCardIntoHand(Ref_TargetAnteCard)`,
		`Exile([CardInGraveyards(Ref_TargetAnteCard)],[])`,
		`TapPermanent(Ref_TargetAnteCard)`,
		`PlayerAction(Ref_TargetAnteCard,DrawACard)`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(Targeted([TargetAnteCard(AnyCard)],ActionList([` + action + `])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted incompatible action %s", action)
		}
	}
}

func TestRONAnteReferenceDarkpactTarget(t *testing.T) {
	record := staticBaseRecord(t, "LEB", "Darkpact")
	rules := record.Fields["Rules"]
	targeted := rules.Args[1].Args[0]
	nodes, err := Parse(strings.NewReader(`ActionList([GainLife(Integer(3))])`), "test-body.ron")
	if err != nil {
		t.Fatal(err)
	}
	targeted.Args[1] = nodes[0]
	rules.Args[1].Args[0] = targeted
	record.Fields["Rules"] = rules
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
	g, _ := mage.NewGameWithAnte(a, b, nil, nil)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	ante := mage.NewArtifact("opponent ante", "")
	ante.SetOwner(b.PlayerID())
	b.AddToAnte(ante)
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Black, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{ante.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 2 || g.StackSize() != 0 {
		t.Fatal("failed payment changed state")
	}
	a.ManaPool().Add(core.Black, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{ante.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 23 || b.Ante()[0].ID() != ante.ID() || len(a.Graveyard()) != 1 {
		t.Fatal("real target subrule did not resolve")
	}
}

func TestRONAnteReferenceInventory(t *testing.T) {
	record := staticBaseRecord(t, "LEB", "Darkpact")
	entries := map[string]*InventorySignature{}
	walkInventory(record, "Darkpact", "Card", "card", "collection", true, entries)
	roles := map[string]string{}
	for _, entry := range entries {
		roles[entry.Name] = entry.Role
	}
	if roles["TargetAnteCard"] != "target" || roles["AnyCard"] != "card_filter" || roles["Ref_TargetAnteCard"] != "card_reference" || roles["You"] != "player_reference" {
		t.Fatal("incorrect ante semantic roles", roles)
	}
}

func TestRONAnteReferenceContextsAndInstances(t *testing.T) {
	body := `Targeted([TargetAnteCard(AnyCard)],ActionList([GainLife(Integer(3))]))`
	for _, trigger := range []bool{false, true} {
		a, b := mage.NewBasePlayer("A"), mage.NewBasePlayer("B")
		g, _ := mage.NewGameWithAnte(a, b, nil, nil)
		first, second := mage.NewArtifact("first", ""), mage.NewArtifact("second", "")
		first.SetOwner(a.PlayerID())
		second.SetOwner(b.PlayerID())
		a.AddToAnte(first)
		b.AddToAnte(second)
		rule := `Activated(PayMana([]),` + body + `)`
		if trigger {
			rule = `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)),` + body + `)`
		}
		factory := permanentScopeFactory(t, "Artifact", rule)
		one, two := factory(), factory()
		one.SetOwner(a.PlayerID())
		two.SetOwner(b.PlayerID())
		p := g.PutOnBattlefield(one, a.PlayerID())
		q := g.PutOnBattlefield(two, b.PlayerID())
		if trigger {
			g.PutTriggersOnStack()
			if g.StackSize() != 2 {
				t.Fatal("entry targets were not placed")
			}
		} else {
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(b.PlayerID(), q.ID(), 0, []uuid.UUID{second.ID()}); err != nil {
				t.Fatal(err)
			}
			b.RemoveFromAnte(second.ID())
			b.AddToAnte(second)
		}
		g.ResolveStack()
		if a.Life() != 23 {
			t.Fatal("first instance lost its target")
		}
		want := 20
		if trigger {
			want = 23
		}
		if b.Life() != want {
			t.Fatal("second instance shared target state")
		}
	}
}

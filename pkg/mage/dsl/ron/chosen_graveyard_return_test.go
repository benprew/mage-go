package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func chosenGraveyardRON(t *testing.T, action string) mage.CardFactory {
	t.Helper()
	nodes, err := Parse(strings.NewReader(`Card(Name:"Return contract",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([`+action+`])))])`), "return.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONChosenGraveyardReturnGameplay(t *testing.T) {
	record := hostUntapRecord(t, "LEG", "Glyph of Reincarnation")
	targeted := record.Fields["Rules"].Args[1].Args[0]
	clause := targeted.Args[1].Args[0].Args[1].Args[0].Args[0]
	if clause.Name != "ReturnACardFromPlayersGraveyardToBattlefield" || clause.Args[2].Args[0].Name != "EntersUnderOwnersControl" {
		t.Fatal("real clause changed")
	}
	for _, entry := range []string{`EntersUnderOwnersControl`, `EntersUnderPlayersControl(You)`} {
		for _, change := range []string{"normal", "empty", "copy", "clone"} {
			t.Run(entry+"/"+change, func(t *testing.T) {
				factory := chosenGraveyardRON(t, `ReturnACardFromPlayersGraveyardToBattlefield(IsCardtype(Creature),Ref_TargetPlayer,[`+entry+`])`)
				g, a, b := ronGame()
				target := mage.NewCreature("Candidate", "", 2, 3)
				target.SetOwner(b.PlayerID())
				if change != "empty" {
					b.AddToGraveyard(target)
				}
				spell := factory()
				spell.SetOwner(a.PlayerID())
				a.AddToHand(spell)
				if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
				clone := g.Clone()
				if change == "copy" && g.CopySpellOnStack(spell.ID(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveStack()
				p := g.FindPermanent(target.ID())
				if change == "empty" {
					if p != nil {
						t.Fatal("empty graveyard returned card")
					}
					return
				}
				want := a.PlayerID()
				if entry == "EntersUnderOwnersControl" {
					want = b.PlayerID()
				}
				if p == nil || p.ControllerID() != want || p.Card.Owner() != b.PlayerID() {
					t.Fatal("wrong return")
				}
				if clone.FindPermanent(target.ID()) != nil {
					t.Fatal("original changed clone")
				}
				clone.ResolveStack()
				if clone.FindPermanent(target.ID()) == nil {
					t.Fatal("clone lost return")
				}
			})
		}
	}
}

func TestRONChosenGraveyardReturnInvalidAndNative(t *testing.T) {
	valid := `ReturnACardFromPlayersGraveyardToBattlefield(IsCardtype(Creature),You,[EntersUnderOwnersControl])`
	for _, action := range []string{
		`ReturnACardFromPlayersGraveyardToBattlefield`,
		`ReturnACardFromPlayersGraveyardToBattlefield()`,
		`ReturnACardFromPlayersGraveyardToBattlefield(IsCardtype(Creature),You)`,
		strings.Replace(valid, "IsCardtype(Creature)", "IsCardtype(Instant)", 1),
		strings.Replace(valid, "IsCardtype(Creature)", "IsCardtype(Creature())", 1),
		strings.Replace(valid, "You", "Trigger_ThatPermanent", 1),
		strings.Replace(valid, "[EntersUnderOwnersControl]", "[]", 1),
		strings.Replace(valid, "EntersUnderOwnersControl", "EntersUnderOwnersControl()", 1),
		strings.Replace(valid, "EntersUnderOwnersControl", "EntersUnderPlayersControl", 1),
		strings.Replace(valid, "EntersUnderOwnersControl", "EntersUnderPlayersControl(Ref_TargetPlayer)", 1),
		strings.Replace(valid, "EntersUnderOwnersControl", "EntersUnderOwnersControl,EntersUnderOwnersControl", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+action+`]))])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", action)
		}
	}
	for _, name := range []string{"Glyph of Reincarnation"} {
		if _, err := Compile(hostUntapRecord(t, "LEG", name)); err == nil {
			t.Fatal("published incomplete record", name)
		}
	}
	files := fstest.MapFS{"valid.ron": {Data: []byte(`Card(Name:"Return",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + valid + `]))])`)}}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([ReturnACardFromPlayersGraveyardToBattlefield()]))])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection changed registry")
	}
}

func TestRONChosenGraveyardReturnInventory(t *testing.T) {
	entries := map[string]*InventorySignature{}
	for _, name := range []string{"Reincarnation", "Glyph of Reincarnation"} {
		walkInventory(hostUntapRecord(t, "LEG", name), name, "Card", "card", "collection", false, entries)
	}
	found := map[string]bool{}
	for _, e := range entries {
		if strings.Contains(e.Context, "ReturnACardFromPlayersGraveyardToBattlefield") {
			if e.Name == "IsCardtype" && e.Role == "graveyard_card_filter" {
				found["filter"] = true
			}
			if e.Name == "Creature" && e.Role == "card_type" {
				found["type"] = true
			}
			if e.Name == "OwnerOfPermanent" && e.Role == "player_reference" {
				found["player"] = true
			}
			if e.Name == "EntersUnderOwnersControl" && e.Role == "entry_operation" {
				found["entry"] = true
			}
		}
	}
	for _, key := range []string{"filter", "type", "player", "entry"} {
		if !found[key] {
			t.Error("missing inventory role", key)
		}
	}
}

func TestRONChosenGraveyardReturnDelayedTiming(t *testing.T) {
	for _, change := range []string{"normal", "source-left", "source-phase"} {
		t.Run(change, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([ReturnACardFromPlayersGraveyardToBattlefield(IsCardtype(Creature),ControllerOfPermanent(ThisPermanent),[EntersUnderPlayersControl(You)])]))]))`)
			g, a, b := ronGame()
			source := g.PutOnBattlefield(factory(), a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			candidate := mage.NewCreature("Later candidate", "", 2, 3)
			candidate.SetOwner(a.PlayerID())
			a.AddToGraveyard(candidate)
			if change == "source-left" {
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
			}
			if change == "source-phase" {
				g.PhaseOut(source.ID())
			}
			clone := g.Clone()
			g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			if g.FindPermanent(candidate.ID()) != nil {
				t.Fatal("entry before delayed resolution")
			}
			g.ResolveStack()
			if g.FindPermanent(candidate.ID()) == nil || g.FindPermanent(candidate.ID()).ControllerID() != a.PlayerID() {
				t.Fatal("delayed return lost saved source information")
			}
			if clone.FindPermanent(candidate.ID()) != nil {
				t.Fatal("resolution changed clone")
			}
			clone.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: b.PlayerID()})
			clone.ResolveStack()
			if clone.FindPermanent(candidate.ID()) == nil {
				t.Fatal("cloned delayed return failed")
			}
		})
	}
}

func TestRONChosenGraveyardReturnRealClauses(t *testing.T) {
	for _, name := range []string{"Reincarnation", "Glyph of Reincarnation"} {
		t.Run(name, func(t *testing.T) {
			record := hostUntapRecord(t, "LEG", name)
			var clause Node
			var find func(Node)
			find = func(n Node) {
				if n.Name == "ReturnACardFromPlayersGraveyardToBattlefield" {
					clause = n
				}
				for _, child := range n.Args {
					find(child)
				}
			}
			find(record.Fields["Rules"])
			if clause.Name == "" {
				t.Fatal("source clause missing")
			}
			// The fixture supplies the player while event and loop bindings remain unsupported.
			clause.Args[1] = Node{Kind: Symbol, Name: "You"}
			entry := &clause.Args[2].Args[0]
			if entry.Name == "EntersUnderPlayersControl" {
				entry.Args[0] = Node{Kind: Symbol, Name: "You"}
			}
			build, err := compileChosenGraveyardReturn(clause, scope{})
			if err != nil {
				t.Fatal(err)
			}
			g, a, _ := ronGame()
			card := mage.NewCreature("Chosen creature", "", 2, 3)
			card.SetOwner(a.PlayerID())
			a.AddToGraveyard(card)
			if err := mage.ApplyEffect(g, build(instanceBindings{}), uuid.Nil, a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) == nil {
				t.Fatal("real filter and entry clause did not return creature")
			}
		})
	}
}

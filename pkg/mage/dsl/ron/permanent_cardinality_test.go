package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONPermanentCardinalityBalanceInstructions(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		for _, index := range []int{0, 2} {
			record := staticBaseRecord(t, set, "Balance")
			action := record.Fields["Rules"].Args[0].Args[0].Args[0].Args[index]
			record.Fields["Rules"] = Node{Kind: List, Args: []Node{{Kind: Tuple, Name: "SpellActions", Args: []Node{{Kind: Tuple, Name: "ActionList", Args: []Node{{Kind: List, Args: []Node{action}}}}}}}}
			factory, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			g.SetStep(core.PrecombatMain)
			for i, player := range []*mage.BasePlayer{a, b} {
				for range 3 - i*2 {
					card := mage.NewLand("Land")
					if index == 2 {
						card = mage.NewCreature("Creature", "", 1, 1)
					}
					g.PutOnBattlefield(card, player.PlayerID())
				}
			}
			spell := factory()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			a.ManaPool().Add(core.White, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err == nil || len(a.Graveyard()) != 0 || g.StackSize() != 0 {
				t.Fatal("unpaid instruction ran")
			}
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			g.ResolveStack()
			if len(a.Graveyard()) != 3 || len(b.Graveyard()) != 0 || len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 0 {
				t.Fatal("resolution or clone isolation")
			}
			clone.ResolveStack()
			if len(clone.GetPlayer(a.PlayerID()).Graveyard()) != 3 {
				t.Fatal("clone lost instruction")
			}
		}
		if _, err := Compile(staticBaseRecord(t, set, "Balance")); err == nil || !strings.Contains(err.Error(), "DiscardAllButNumberCards") {
			t.Fatal("complete Balance must retain its hand-operation blocker", err)
		}
	}
}

func TestRONPermanentCardinalityInvalid(t *testing.T) {
	for _, action := range []string{
		`SacrificeAllPermanentsExceptForNum`,
		`SacrificeAllPermanentsExceptForNum(IsCardtype(Land))`,
		`SacrificeAllPermanentsExceptForNum(IsCardtype(Land),Integer(-1))`,
		`SacrificeAllPermanentsExceptForNum(IsCardtype(Land),TheLowestNumberOfPermanentsControlledAmongPlayers(IsCardtype(Land),You))`,
		`SacrificeAllPermanentsExceptForNum(IsCardtype(Land),TheLowestNumberOfPermanentsControlledAmongPlayers(IsCardtype(Land)))`,
		`SacrificeAllPermanentsExceptForNum(SinglePermanent(TheChosenPermanent),Integer(1))`,
	} {
		record := staticBaseRecord(t, "LEA", "Balance")
		record.Fields["Rules"] = nameScopeNode(t, `[SpellActions(ActionList([EachPlayerAction(AnyPlayer,`+action+`)]))]`)
		if _, err := Compile(record); err == nil {
			t.Fatal("invalid declaration accepted", action)
		}
	}
}

func TestRONPermanentCardinalityTargetsCopiesAndDelay(t *testing.T) {
	for _, state := range []string{"present", "returned", "phased", "cycle", "control", "copy", "clone"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([EachPlayerAction(AnyPlayer,SacrificeAllPermanentsExceptForNum(SinglePermanent(Ref_TargetPermanent),Integer(0)))])))`)
			c := factory()
			c.SetOwner(a.PlayerID())
			a.AddToHand(c)
			p := g.PutOnBattlefield(mage.NewCreature("Subject", "", 1, 1), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			switch state {
			case "returned":
				g.BouncePermanentToHand(p)
				b.RemoveFromHand(p.ID())
				g.PutOnBattlefield(p.Card, b.PlayerID())
			case "phased":
				g.PhaseOut(p.ID())
			case "cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: c.ID(), TargetID: p.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			if state == "clone" {
				clone.PhaseOut(p.ID())
			}
			g.ResolveStack()
			wantPresent := state == "returned" || state == "phased"
			if (g.FindPermanentIncludingPhased(p.ID()) != nil) != wantPresent || clone.FindPermanentIncludingPhased(p.ID()) == nil {
				t.Fatal("target identity, current controller or clone isolation")
			}
			clone.ResolveStack()
			if (clone.FindPermanentIncludingPhased(p.ID()) != nil) != (wantPresent || state == "clone") {
				t.Fatal("cloned target legality")
			}
		})
	}
	g, a, b := ronGame()
	f := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([EachPlayerAction(AnyPlayer,SacrificeAllPermanentsExceptForNum(IsCardtype(Creature),TheLowestNumberOfPermanentsControlledAmongPlayers(IsCardtype(Creature),AnyPlayer)))]))]))`)
	source := g.PutOnBattlefield(f(), a.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	g.BouncePermanentToHand(source)
	for range 2 {
		g.PutOnBattlefield(mage.NewCreature("Late", "", 1, 1), b.PlayerID())
	}
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if len(b.Graveyard()) != 2 {
		t.Fatal("delayed operation failed to read current counts after source departure")
	}
}

func TestRONPermanentCardinalityRolesAndInventory(t *testing.T) {
	body := `EachPlayerAction(AnyPlayer,SacrificeAllPermanentsExceptForNum(IsCardtype(Land),TheLowestNumberOfPermanentsControlledAmongPlayers(IsCardtype(Land),AnyPlayer)))`
	for _, rule := range []string{
		`Activated(` + body + `,ActionList([]))`,
		`EachPermanentLayerEffect(AnyPermanent,[` + body + `])`,
		`SpellActions(ActionList([PlayerAction(You,SacrificeAllPermanentsExceptForNum(IsCardtype(Land),Integer(1)))]))`,
	} {
		record := staticBaseRecord(t, "LEA", "Balance")
		record.Fields["Rules"] = nameScopeNode(t, `[`+rule+`]`)
		if _, err := Compile(record); err == nil {
			t.Fatal("invalid context accepted", rule)
		}
	}
	text := `Card(Name:"Probe",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([` + body + `]))])`
	fs := fstest.MapFS{"a.ron": {Data: []byte(text)}}
	report, err := InspectFS(fs, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	found := make(map[string]bool)
	for _, sig := range report.Signatures {
		if sig.Name == "TheLowestNumberOfPermanentsControlledAmongPlayers" && sig.Role == "value" || sig.Name == "IsCardtype" && sig.Role == "permanent_filter" {
			found[sig.Name] = true
		}
	}
	if len(found) != 2 {
		t.Fatal("cardinality inventory roles", found)
	}
	fs["b.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([EachPlayerAction(AnyPlayer,SacrificeAllPermanentsExceptForNum(IsCardtype(Land),Integer(-1)))]))])`)}
	if registry, err := LoadFS(fs, "*.ron"); err == nil || registry != nil {
		t.Fatal("invalid cardinality published a partial registry")
	}
}

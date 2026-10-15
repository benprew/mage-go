package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLandPlayEventFastbond(t *testing.T) {
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		t.Run(set, func(t *testing.T) {
			record := staticBaseRecord(t, set, "Fastbond")
			condition := record.Fields["Rules"].Args[1].Args[1]
			if condition.Args[1].Name != "Not" {
				t.Fatal("Fastbond must exclude the first land")
			}
			compiled, err := Compile(record)
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			spell := compiled()
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast")
			}
			a.ManaPool().Add(core.Green, 1)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			g.PutOnBattlefield(mage.NewLand("Direct"), a.PlayerID())
			for i := range 4 {
				land := mage.NewLand("Land")
				a.AddToHand(land)
				if err := g.PlayLand(a.PlayerID(), land.ID()); err != nil {
					t.Fatal(err)
				}
				want := 20 - i
				if a.Life() != want || b.Life() != 20 {
					t.Fatalf("land %d life=%d/%d", i+1, a.Life(), b.Life())
				}
			}
			g.SetActivePlayerIndex(1)
			g.SetLandsPlayedThisTurn(0)
			land := mage.NewLand("Opponent land")
			b.AddToHand(land)
			if err := g.PlayLand(b.PlayerID(), land.ID()); err != nil {
				t.Fatal(err)
			}
			if a.Life() != 17 || b.Life() != 20 {
				t.Fatal("opponent play triggered")
			}
		})
	}
}

const landPlayEvent = `WhenAPlayerPlaysALand(SinglePlayer(You),IsCardtype(Land))`
const laterLand = `PermanentPassesFilter(Trigger_ThatLand,Not(IsFirstLandPlayedByPlayerThisTurn))`

func TestRONLandPlayEventBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Enchantment", `TriggerI(`+landPlayEvent+`,`+laterLand+`,ActionList([TapPermanent(Trigger_ThatLand),PlayerAction(Trigger_ThatPlayer,GainLife(Integer(2)))]))`)
	g, a, _ := ronGame()
	g.PutOnBattlefield(factory(), a.PlayerID())
	g.GrantExtraLandPlay(a.PlayerID(), 1)
	for i := range 2 {
		land := mage.NewLand("Land")
		a.AddToHand(land)
		if err := g.PlayLand(a.PlayerID(), land.ID()); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(land.ID()).Tapped != (i == 1) {
			t.Fatal("wrong event land")
		}
	}
	if a.Life() != 22 {
		t.Fatal("event player not bound")
	}
}

func TestRONLandPlayEventInvalid(t *testing.T) {
	valid := `TriggerI(` + landPlayEvent + `,` + laterLand + `,ActionList([TapPermanent(Trigger_ThatLand)]))`
	for _, rule := range []string{
		strings.Replace(valid, "SinglePlayer(You)", "AnyPlayer", 1),
		strings.Replace(valid, "You)", "You())", 1),
		strings.Replace(valid, "IsCardtype(Land)", "IsCardtype(Creature)", 1),
		strings.Replace(valid, "Trigger_ThatLand,Not", "ThisPermanent,Not", 1),
		strings.Replace(valid, "IsFirstLandPlayedByPlayerThisTurn)", "IsFirstLandPlayedByPlayerThisTurn())", 1),
		strings.Replace(valid, "Not(IsFirstLandPlayedByPlayerThisTurn)", "Not(IsTapped)", 1),
		strings.Replace(valid, "TapPermanent(Trigger_ThatLand)", "TapPermanent(Trigger_ThatPermanent)", 1),
		valid + `,Activated(PayMana([]),ActionList([TapPermanent(Trigger_ThatLand)]))`,
		`TriggerI(` + landPlayEvent + `,` + laterLand + `,ActionList([PlayerAction(Trigger_ThatLand,GainLife(Integer(1)))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+rule+`])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("invalid declaration accepted: %s", rule)
		}
	}
}

func TestRONLandPlayEventLifecycle(t *testing.T) {
	for _, change := range []string{"departure", "return", "phase", "control", "prevention", "two sources"} {
		t.Run(change, func(t *testing.T) {
			factory, err := Compile(staticBaseRecord(t, "LEA", "Fastbond"))
			if err != nil {
				t.Fatal(err)
			}
			g, a, b := ronGame()
			card := factory()
			source := g.PutOnBattlefield(card, a.PlayerID())
			if change == "two sources" {
				g.PutOnBattlefield(factory(), a.PlayerID())
			}
			g.FireEvent(core.GameEvent{Type: core.EvtLandPlayed, SourceID: uuid.New(), PlayerID: a.PlayerID(), Amount: 2})
			g.PutTriggersOnStack()
			g.SetLandsPlayedThisTurn(0)
			switch change {
			case "departure", "return":
				g.RemoveFromBattlefield(source)
				if change == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "prevention":
				g.AddPreventionShield(a.PlayerID(), 1)
			}
			clone := g.Clone()
			g.ResolveStack()
			want := 19
			if change == "prevention" {
				want = 20
			}
			if change == "two sources" {
				want = 18
			}
			if a.Life() != want || b.Life() != 20 {
				t.Fatalf("saved source/controller damage: %d/%d", a.Life(), b.Life())
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone changed")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != want {
				t.Fatal("clone lost pending damage")
			}
		})
	}
}

func TestRONLandPlayEventAtomicLoadAndInventory(t *testing.T) {
	source := `Card(Name:"Land event",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[TriggerI(` + landPlayEvent + `,` + laterLand + `,ActionList([TapPermanent(Trigger_ThatLand),PlayerAction(Trigger_ThatPlayer,GainLife(Integer(1)))]))])`
	files := fstest.MapFS{"good.ron": {Data: []byte(source)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v %+v", err, report)
	}
	for name, role := range map[string]string{"WhenAPlayerPlaysALand": "event", "SinglePlayer": "player_filter", "Trigger_ThatLand": "permanent_reference", "Not": "permanent_filter", "IsFirstLandPlayedByPlayerThisTurn": "permanent_filter"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("missing inventory role for %s: %s", name, role)
		}
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.Replace(source, `Name:"Land event"`, `Name:"Bad"`, 1), "Trigger_ThatLand", "Trigger_ThatLand()"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection changed registry")
	}
}

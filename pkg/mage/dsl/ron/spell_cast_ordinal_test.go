package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const laterInstantEvent = `WhenAPlayerCastsTheirNthSpellInATurn(Opponent,NotEqualTo(Integer(1)),IsCardtype(Instant))`

func ronOrdinalCast(t *testing.T, g *mage.Game, p *mage.BasePlayer, instant bool) *mage.StackObject {
	t.Helper()
	var spell mage.Card = mage.NewSorcery("Other spell", "")
	if instant {
		spell = mage.NewInstant("Instant", "")
	}
	spell.SetOwner(p.PlayerID())
	p.AddToHand(spell)
	if err := g.CastSpellByID(p.PlayerID(), spell.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	return g.StackPeek()
}

func TestRONSpellCastOrdinalDruid(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Ichneumon Druid")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast succeeded")
	}
	a.ManaPool().Add(core.Green, 2)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	source := g.FindPermanent(card.ID())
	if source == nil || source.CurrentPower(g) != 1 || source.CurrentToughness(g) != 1 {
		t.Fatal("Druid did not enter with printed stats")
	}
	g.SetActivePlayerIndex(1)
	ronOrdinalCast(t, g, b, false)
	g.ResolveStack()
	for i := range 3 {
		spell := ronOrdinalCast(t, g, b, true)
		g.PutTriggersOnStack()
		wantStack := 1
		if i > 0 {
			wantStack++
		}
		if g.StackSize() != wantStack {
			t.Fatalf("instant %d: stack=%d", i+1, g.StackSize())
		}
		if i == 0 && g.CopyStackObjectDirect(spell, b.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		g.ResolveStack()
		if b.Life() != 20-4*i || a.Life() != 20 {
			t.Fatalf("instant %d: life=%d/%d", i+1, a.Life(), b.Life())
		}
	}
	for range 2 {
		ronOrdinalCast(t, g, a, true)
		g.ResolveStack()
	}
	if a.Life() != 20 || b.Life() != 12 {
		t.Fatal("Druid damaged its controller")
	}
}

func TestRONSpellCastOrdinalLifecycle(t *testing.T) {
	for _, change := range []string{"none", "leave", "return", "phase", "control", "counter spell", "copy trigger", "prevention", "two sources"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			ronOrdinalCast(t, g, b, true)
			g.ResolveStack()
			factory, err := Compile(staticBaseRecord(t, "LEG", "Ichneumon Druid"))
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			source := g.PutOnBattlefield(card, a.PlayerID())
			if change == "two sources" {
				g.PutOnBattlefield(factory(), a.PlayerID())
			}
			spell := ronOrdinalCast(t, g, b, true)
			g.PutTriggersOnStack()
			want := 16
			switch change {
			case "leave", "return":
				g.RemoveFromBattlefield(source)
				if change == "return" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "counter spell":
				g.GetStack().RemoveByID(spell.ID)
			case "copy trigger":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				want = 12
			case "prevention":
				g.AddPreventionShield(b.PlayerID(), 2)
				want = 18
			case "two sources":
				want = 12
			}
			clone := g.Clone()
			g.ResolveStack()
			if a.Life() != 20 || b.Life() != want {
				t.Fatalf("life=%d/%d want %d", a.Life(), b.Life(), want)
			}
			if clone.GetPlayer(b.PlayerID()).Life() != 20 {
				t.Fatal("clone changed")
			}
			clone.ResolveStack()
			if clone.GetPlayer(b.PlayerID()).Life() != want {
				t.Fatal("clone lost trigger")
			}
		})
	}
}

func TestRONSpellCastOrdinalBindings(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(`+laterInstantEvent+`,ActionList([CounterSpell(Trigger_ThatSpell),PermanentDealsDamage(ThisPermanent,Integer(2),Player(Trigger_ThatPlayer))]))`)
	g, a, b := ronGame()
	g.PutOnBattlefield(factory(), a.PlayerID())
	ronOrdinalCast(t, g, b, true)
	g.ResolveStack()
	second := ronOrdinalCast(t, g, b, true)
	g.PutTriggersOnStack()
	third := ronOrdinalCast(t, g, b, true)
	g.PutTriggersOnStack()
	if len(g.StackPeek().Targets) != 0 {
		t.Fatal("event operands became targets")
	}
	g.ResolveTopOfStack()
	if g.GetStack().FindByID(third.ID) != nil || g.GetStack().FindByID(second.ID) == nil || b.Life() != 18 {
		t.Fatal("third event read another cast")
	}
	g.ResolveStack()
	if b.Life() != 16 || a.Life() != 20 {
		t.Fatal("independent pending bindings lost")
	}
}

func TestRONSpellCastOrdinalInvalid(t *testing.T) {
	valid := `TriggerA(` + laterInstantEvent + `,ActionList([PermanentDealsDamage(ThisPermanent,Integer(1),Player(Trigger_ThatPlayer))]))`
	for _, rule := range []string{
		strings.Replace(valid, "Opponent,", "You,", 1),
		strings.Replace(valid, "Opponent,", "AnyPlayer,", 1),
		strings.Replace(valid, "Opponent,", "SinglePlayer(You),", 1),
		strings.Replace(valid, "NotEqualTo(Integer(1))", "NotEqualTo()", 1),
		strings.Replace(valid, "Integer(1)", "Integer(1,2)", 1),
		strings.Replace(valid, "IsCardtype(Instant)", "IsCardtype()", 1),
		strings.Replace(valid, ",IsCardtype(Instant)", "", 1),
		strings.Replace(valid, "Opponent,", "Opponent(),", 1),
		strings.Replace(valid, "NotEqualTo(Integer(1))", "Integer(1)", 1),
		strings.Replace(valid, "NotEqualTo(Integer(1))", "NotEqualTo(Integer(2))", 1),
		strings.Replace(valid, "NotEqualTo(Integer(1))", "NotEqualTo(Integer(-1))", 1),
		strings.Replace(valid, "NotEqualTo(Integer(1))", "NotEqualTo(ValueX)", 1),
		strings.Replace(valid, "IsCardtype(Instant)", "IsCardtype(Sorcery)", 1),
		strings.Replace(valid, "IsCardtype(Instant)", "IsCardtype(Instant())", 1),
		strings.Replace(valid, "Player(Trigger_ThatPlayer)", "Player(Trigger_ThatSpell)", 1),
		strings.Replace(valid, "TriggerA(", "TriggerI(", 1),
		valid + `,Activated(PayMana([]),ActionList([PermanentDealsDamage(ThisPermanent,Integer(1),Player(Trigger_ThatPlayer))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("invalid declaration accepted: %s", rule)
		}
	}
}

func TestRONSpellCastOrdinalInventory(t *testing.T) {
	source := `Card(Name:"Ordinal",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[TriggerA(` + laterInstantEvent + `,ActionList([PermanentDealsDamage(ThisPermanent,Integer(1),Player(Trigger_ThatPlayer))]))])`
	files := fstest.MapFS{"good.ron": {Data: []byte(source)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v %+v", err, report)
	}
	for name, role := range map[string]string{"WhenAPlayerCastsTheirNthSpellInATurn": "event", "Opponent": "player_filter", "NotEqualTo": "cast_ordinal_comparison", "Integer": "cast_ordinal_value", "IsCardtype": "spell_filter", "Trigger_ThatPlayer": "player_reference"} {
		found := false
		for _, sig := range report.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("missing role %s for %s", role, name)
		}
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.Replace(source, `Name:"Ordinal"`, `Name:"Bad"`, 1), "NotEqualTo(Integer(1))", "NotEqualTo(Integer(2))"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection changed registry")
	}
}

func TestRONSpellCastOrdinalPhasing(t *testing.T) {
	g, a, b := ronGame()
	source := g.PutOnBattlefield(repositoryCard(t, "LEG", "Ichneumon Druid"), a.PlayerID())
	g.PhaseOut(source.ID())
	for range 2 {
		ronOrdinalCast(t, g, b, true)
		g.ResolveStack()
	}
	if b.Life() != 20 {
		t.Fatal("phased source triggered")
	}
	g.PhaseIn([]uuid.UUID{source.ID()})
	ronOrdinalCast(t, g, b, true)
	g.ResolveStack()
	if b.Life() != 16 {
		t.Fatal("phase cycle lost cast history")
	}
}

package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONExileControllerCountsMartyrsCry(t *testing.T) {
	for _, copySpell := range []bool{false, true} {
		g, a, b := ronGame()
		for range 8 {
			a.AddToLibrary(mage.NewInstant("A draw", ""))
			b.AddToLibrary(mage.NewInstant("B draw", ""))
		}
		stolen := mage.NewCreature("Stolen", "{W}", 2, 2)
		stolen.SetOwner(a.PlayerID())
		first := g.PutOnBattlefield(stolen, b.PlayerID())
		token := g.PutOnBattlefield(mage.NewCreature("Token", "{W}", 2, 2), b.PlayerID())
		token.IsToken = true
		own := g.PutOnBattlefield(mage.NewCreature("Own", "{W}", 2, 2, mage.WithKeyword(core.Shroud), mage.WithKeyword(core.Indestructible)), a.PlayerID())
		survivor := g.PutOnBattlefield(mage.NewCreature("Black", "{B}", 2, 2), b.PlayerID())
		phased := g.PutOnBattlefield(mage.NewCreature("Phased", "{W}", 2, 2), b.PlayerID())
		g.PhaseOut(phased.ID())
		g.AddRegenerationShield(first.ID())
		card := repositoryCard(t, "DRK", "Martyr's Cry")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.White, 1)
		if g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0) == nil || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 1 {
			t.Fatal("failed payment mutated state")
		}
		a.ManaPool().Add(core.White, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
		if g.FindPermanent(own.ID()) == nil || len(b.Hand()) != 0 {
			t.Fatal("action ran at announcement")
		}
		if copySpell && g.CopySpellOnStack(card.ID(), b.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		clone := g.Clone()
		g.ResolveStack()
		if len(a.Hand()) != 1 || len(b.Hand()) != 2 || g.FindPermanent(own.ID()) != nil || g.FindPermanent(first.ID()) != nil || g.FindPermanent(token.ID()) != nil || g.FindPermanent(survivor.ID()) == nil || g.FindPermanentIncludingPhased(phased.ID()) == nil {
			t.Fatalf("hands %d/%d or exile selection", len(a.Hand()), len(b.Hand()))
		}
		if len(clone.GetPlayer(a.PlayerID()).Hand()) != 0 {
			t.Fatal("clone changed before resolution")
		}
		clone.ResolveStack()
		if len(clone.GetPlayer(a.PlayerID()).Hand()) != 1 || len(clone.GetPlayer(b.PlayerID()).Hand()) != 2 {
			t.Fatal("clone lost counts")
		}
	}
}

const exileCountProducer = `Exile([EachPermanent(And([IsColor(White),IsCardtype(Creature)]))],[])`
const exileCountDraw = `EachPlayerAction(AnyPlayer,DrawNumberCards(NumPermanentsOfTypeExiledThisWay(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(ActionPlayer))]))))`

func TestRONExileControllerCountsInvalid(t *testing.T) {
	for _, body := range []string{
		exileCountDraw, exileCountDraw + `,` + exileCountProducer,
		`FlipACoin_OnLose([` + exileCountProducer + `]),` + exileCountDraw,
		exileCountProducer + `,` + strings.Replace(exileCountDraw, `ActionPlayer`, `You`, 1),
		exileCountProducer + `,` + strings.Replace(exileCountDraw, `AnyPlayer`, `Opponent`, 1),
		exileCountProducer + `,` + strings.Replace(exileCountDraw, `IsCardtype(Creature)`, `IsColor(White)`, 1),
		exileCountProducer + `,` + strings.Replace(exileCountDraw, `NumPermanentsOfTypeExiledThisWay(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(ActionPlayer))]))`, `NumPermanentsOfTypeExiledThisWay`, 1),
		strings.Replace(exileCountProducer, `IsCardtype(Creature)`, `AnyPermanent`, 1) + `,` + exileCountDraw,
		strings.Replace(exileCountProducer, `],[])`, `],[ExilesFaceDown])`, 1) + `,` + exileCountDraw,
		exileCountProducer + `,CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([` + exileCountDraw + `]))`,
		exileCountProducer + `,GainLife(NumPermanentsOfTypeExiledThisWay(And([IsCardtype(Creature),ControlledByAPlayer(SinglePlayer(ActionPlayer))])))`,
	} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([` + body + `]))])`
		nodes, err := Parse(strings.NewReader(src), "invalid-exile-count.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONExileControllerCountsLoadAndInventory(t *testing.T) {
	source := `Card(Name:"Saved exile",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(ActionList([` + exileCountProducer + `,` + exileCountDraw + `]))])`
	files := fstest.MapFS{"valid.ron": {Data: []byte(source)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Saved exile")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Saved exile")
	if err != nil {
		t.Fatal(err)
	}
	if first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared factory action")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "And" && signature.Role == "permanent_filter" && strings.Contains(signature.Context, "NumPermanentsOfTypeExiledThisWay") {
			found = true
		}
	}
	if !found {
		t.Fatal("saved result filter role absent")
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(source, `ActionPlayer`, `Trigger_ThatPlayer`, 1))}
	if LoadInto(registry, files, "*.ron") == nil || !registry.CardRegistered("Saved exile") {
		t.Fatal("partial registry published")
	}
}

func TestRONExileControllerCountsIndependentProducers(t *testing.T) {
	g, a, b := ronGame()
	for range 8 {
		a.AddToLibrary(mage.NewInstant("A draw", ""))
		b.AddToLibrary(mage.NewInstant("B draw", ""))
	}
	g.PutOnBattlefield(mage.NewCreature("A white", "{W}", 1, 1), a.PlayerID())
	g.PutOnBattlefield(mage.NewCreature("B white", "{W}", 1, 1), b.PlayerID())
	factory := permanentScopeFactory(t, "Sorcery", `SpellActions(ActionList([`+exileCountProducer+`,`+exileCountDraw+`,`+exileCountProducer+`,`+exileCountDraw+`]))`)
	first, second := factory(), factory()
	castRONSpell(t, g, a, first)
	g.ResolveStack()
	castRONSpell(t, g, a, second)
	g.ResolveStack()
	if len(a.Hand()) != 1 || len(b.Hand()) != 1 {
		t.Fatal("a later empty producer used another result")
	}
}

func TestRONExileControllerCountsSourceDeparture(t *testing.T) {
	source := `Card(Name:"Source",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),ManaCost:[ManaCostW],Rules:[Activated(PayMana([]),ActionList([` + exileCountProducer + `,` + exileCountDraw + `]))],CardPT:(Power:1,Toughness:1))`
	nodes, err := Parse(strings.NewReader(source), "source-exile-count.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	a.AddToLibrary(mage.NewInstant("Draw", ""))
	b.AddToLibrary(mage.NewInstant("Draw", ""))
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	g.PutOnBattlefield(mage.NewCreature("Victim", "{W}", 1, 1), b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()) != nil || len(a.Hand()) != 1 || len(b.Hand()) != 1 {
		t.Fatal("source departure lost continuation or counts")
	}
}

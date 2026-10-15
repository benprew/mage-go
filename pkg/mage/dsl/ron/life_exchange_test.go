package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONLifeExchangeRecord(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Mirror Universe")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast succeeded")
	}
	a.ManaPool().Add(core.Colorless, 6)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	a.SetLife(5)
	b.SetLife(18)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, []uuid.UUID{b.PlayerID()}); err == nil {
		t.Fatal("activation outside upkeep succeeded")
	}
	g.SetStep(core.Upkeep)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, []uuid.UUID{a.PlayerID()}); err == nil {
		t.Fatal("self target succeeded")
	}
	if g.FindPermanent(card.ID()) == nil {
		t.Fatal("failed activation paid sacrifice")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil || a.Life() != 5 || b.Life() != 18 {
		t.Fatal("cost or timing wrong")
	}
	clone := g.Clone()
	g.ResolveStack()
	if a.Life() != 18 || b.Life() != 5 || clone.GetPlayer(a.PlayerID()).Life() != 5 {
		t.Fatal("exchange failed or clone changed")
	}
	clone.ResolveStack()
	if clone.GetPlayer(a.PlayerID()).Life() != 18 || clone.GetPlayer(b.PlayerID()).Life() != 5 {
		t.Fatal("pending clone lost exchange")
	}
}

func TestRONLifeExchangeInvalid(t *testing.T) {
	for _, body := range []string{
		`ExchangeLifeTotalWithPlayer`,
		`ExchangeLifeTotalWithPlayer()`,
		`ExchangeLifeTotalWithPlayer(You,You)`,
		`ExchangeLifeTotalWithPlayer(ThisPermanent)`,
		`ExchangeLifeTotalWithPlayer(Trigger_ThatPlayer)`,
		`ExchangeLifeTotalWithPlayer(Ref_TargetPlayer)`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+body+`]))])`)); err == nil {
			t.Fatal("accepted", body)
		}
	}
	for _, rule := range []string{
		`AsPermanentEnters(ThisPermanent,[ExchangeLifeTotalWithPlayer(You)])`,
		`ReplaceWouldDraw(APlayerWouldDrawExceptFirstDrawStepDraw(AnyPlayer),[ExchangeLifeTotalWithPlayer(You)])`,
		`Activated(ExchangeLifeTotalWithPlayer(You),ActionList([GainLife(Integer(1))]))`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[`+rule+`])`)); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONLifeExchangeScopeAndCopies(t *testing.T) {
	for _, body := range []string{
		`ExchangeLifeTotalWithPlayer(Ref_TargetPlayer)`,
		`PlayerAction(You,ExchangeLifeTotalWithPlayer(Ref_TargetPlayer))`,
		`PlayerActions(You,[ExchangeLifeTotalWithPlayer(Ref_TargetPlayer)])`,
	} {
		g, a, b := ronGame()
		p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Activated(SacrificePermanent(ThisPermanent),Targeted([TargetPlayer(AnyPlayer)],ActionList([`+body+`])))`)(), a.PlayerID())
		a.SetLife(5)
		b.SetLife(18)
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
			t.Fatal(err)
		}
		if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
			t.Fatal("copy failed")
		}
		g.ResolveStack()
		if a.Life() != 5 || b.Life() != 18 {
			t.Fatal("independent copies did not read current totals")
		}
	}
}

func TestRONLifeExchangeCollection(t *testing.T) {
	record := staticBaseRecord(t, "LEG", "Mirror Universe")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Mirror Universe", factory()); err != nil {
		t.Fatal(err)
	}
	source := `Card(Name:"Exchange",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([ExchangeLifeTotalWithPlayer(Ref_TargetPlayer)])))])`
	files := fstest.MapFS{"valid.ron": {Data: []byte(source)}}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "Ref_TargetPlayer" && signature.Role == "player_reference" {
			found = true
		}
	}
	if !found || report.Compiled != 1 {
		t.Fatal("missing typed operand or valid definition")
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(strings.ReplaceAll(source, "Exchange\"", "Invalid\""), "ExchangeLifeTotalWithPlayer(Ref_TargetPlayer)", "ExchangeLifeTotalWithPlayer(ThisPermanent)"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Exchange") || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection changed registry")
	}
}

func TestRONLifeExchangeDelayed(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPlayer(AnyPlayer)],ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([ExchangeLifeTotalWithPlayer(Ref_TargetPlayer)]))])))`)(), a.PlayerID())
	a.SetLife(5)
	b.SetLife(18)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if a.Life() != 5 || b.Life() != 18 {
		t.Fatal("delayed exchange ran early")
	}
	g.BouncePermanentToHand(p)
	a.SetLife(7)
	b.SetLife(23)
	clone := g.Clone()
	g.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
	g.PutTriggersOnStack()
	g.ResolveStack()
	if a.Life() != 23 || b.Life() != 7 || clone.GetPlayer(a.PlayerID()).Life() != 7 {
		t.Fatal("delayed exchange lost saved target, current totals or clone isolation")
	}
}

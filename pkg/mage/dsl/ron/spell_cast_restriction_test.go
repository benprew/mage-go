package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
)

func TestRONSpellCastRestrictionCity(t *testing.T) {
	record := staticBaseRecord(t, "ARN", "City in a Bottle")
	factory := staticBaseSubrule(t, record, record.Fields["Rules"].Args[1])
	for _, active := range []int{0, 1} {
		g, a, _ := ronGame()
		g.SetActivePlayerIndex(active)
		source := g.PutOnBattlefield(factory(), a.PlayerID())
		player := g.ActivePlayerObj()
		spell := mage.NewInstant("Aladdin's Lamp", "")
		player.AddToHand(spell)
		if err := g.CastSpellByID(player.PlayerID(), spell.ID(), nil, 0); err == nil {
			t.Fatal("matching name from another printing was cast")
		}
		if len(player.Hand()) != 1 || g.StackSize() != 0 || len(g.GetCastableSpells(player.PlayerID())) != 0 {
			t.Fatal("rejected cast or query changed state")
		}
		land := mage.NewLand("Desert")
		player.AddToHand(land)
		if err := g.PlayLand(player.PlayerID(), land.ID()); err == nil {
			t.Fatal("compound rule lost the land restriction")
		}
		if g.PutOnBattlefield(mage.NewArtifact("Aladdin's Lamp", ""), player.PlayerID()) == nil {
			t.Fatal("casting restriction stopped direct entry")
		}
		allowed := mage.NewInstant("Mountain", "")
		player.AddToHand(allowed)
		if err := g.CastSpellByID(player.PlayerID(), allowed.ID(), nil, 0); err != nil {
			t.Fatal("basic land name entered the ARN catalog", err)
		}
		g.ResolveStack()
		g.PhaseOut(source.ID())
		if err := g.CastSpellByID(player.PlayerID(), spell.ID(), nil, 0); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Compile(staticBaseRecord(t, "ARN", "City in a Bottle")); err != nil {
		t.Fatal("complete state trigger did not compile", err)
	}
}

func TestRONSpellCastRestrictionInvalid(t *testing.T) {
	for _, rule := range []string{
		`EachPlayerEffect(AnyPlayer(),[CantCastSpells(SharesANameOriginallyPrintedInArabianNights)])`,
		`EachPlayerEffect(You,[CantCastSpells(SharesANameOriginallyPrintedInArabianNights)])`,
		`EachPlayerEffect(AnyPlayer,[CantCastSpells])`,
		`EachPlayerEffect(AnyPlayer,[CantCastSpells()])`,
		`EachPlayerEffect(AnyPlayer,[CantCastSpells(SharesANameOriginallyPrintedInAntiquities,Integer(1))])`,
		`EachPlayerEffect(AnyPlayer,[CantCastSpells(SharesANameOriginallyPrintedInArabianNights())])`,
		`EachPlayerEffect(AnyPlayer,[CantCastSpells(ThisPermanent)])`,
		`EachPlayerEffect(AnyPlayer,[CantCastSpells(IsCardtype(Artifact))])`,
		`EachPlayerEffect(AnyPlayer,[CantCastSpells(SharesANameOriginallyPrintedInArabianNights),Unknown])`,
		`EachPlayerEffect(AnyPlayer,[])`,
		`PlayerEffect(You,[CantCastSpells(SharesANameOriginallyPrintedInArabianNights)])`,
		`Activated(PayMana([]),ActionList([CantCastSpells(SharesANameOriginallyPrintedInArabianNights)]))`,
	} {
		text := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rule + `])`
		if registry, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(text)}}, "*.ron"); err == nil || registry != nil {
			t.Fatal("invalid declaration published", rule)
		}
	}
}

func TestRONSpellCastRestrictionListOrderAndInstances(t *testing.T) {
	for _, operations := range []string{
		`CantCastSpells(SharesANameOriginallyPrintedInArabianNights),CantPlayLands`,
		`CantPlayLands,CantCastSpells(SharesANameOriginallyPrintedInArabianNights)`,
		`CantCastSpells(SharesANameOriginallyPrintedInArabianNights),CantCastSpells(SharesANameOriginallyPrintedInArabianNights)`,
	} {
		factory := permanentScopeFactory(t, "Artifact", `EachPlayerEffect(AnyPlayer,[`+operations+`])`)
		g, a, _ := ronGame()
		first := g.PutOnBattlefield(factory(), a.PlayerID())
		second := g.PutOnBattlefield(factory(), a.PlayerID())
		if first.ID() == second.ID() {
			t.Fatal("factory shared identity")
		}
		g.RemoveFromBattlefield(first)
		if !g.IsSpellCastProhibited(a.PlayerID(), mage.NewInstant("Desert", "")) {
			t.Fatal("one factory instance removed another rule")
		}
		g.RemoveFromBattlefield(second)
		if g.IsSpellCastProhibited(a.PlayerID(), mage.NewInstant("Desert", "")) {
			t.Fatal("departed instances kept rules")
		}
	}
	text := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[EachPlayerEffect(AnyPlayer,[CantCastSpells(SharesANameOriginallyPrintedInArabianNights)])])`
	if registry, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(text)}}, "*.ron"); err == nil || registry != nil {
		t.Fatal("battlefield rule on instant accepted")
	}
}

func TestRONSpellCastRestrictionCompositionAndInventory(t *testing.T) {
	rule := `EachPlayerEffect(AnyPlayer,[CantCastSpells(SharesANameOriginallyPrintedInAntiquities),CantPlayLands,PlaysWithHandRevealed,PlaysWithTopOfLibraryRevealed])`
	factory := permanentScopeFactory(t, "Artifact", rule)
	g, a, _ := ronGame()
	g.PutOnBattlefield(factory(), a.PlayerID())
	spell := mage.NewInstant("Atog", "")
	a.AddToHand(spell)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), nil, 0); err == nil {
		t.Fatal("ATQ name was cast")
	}
	if g.IsSpellCastProhibited(a.PlayerID(), mage.NewInstant("Desert", "")) || !g.IsLandPlayProhibited(mage.NewLand("Mountain")) || !g.IsHandRevealed(a.PlayerID()) || !g.IsTopCardRevealed(a.PlayerID()) {
		t.Fatal("catalog or compound operation changed")
	}
	text := `Card(Name:"Restriction",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + rule + `])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(text)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory", err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "SharesANameOriginallyPrintedInAntiquities" && signature.Role == "casting_card_filter" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing casting card-filter role")
	}
	registry := mage.NewRegistry()
	bad := strings.Replace(text, "PlaysWithHandRevealed", "Unknown", 1)
	if err := LoadInto(registry, fstest.MapFS{"good.ron": {Data: []byte(text)}, "bad.ron": {Data: []byte(bad)}}, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 0 {
		t.Fatal("partial registry published")
	}
}

package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCreatedTokenLeaveGameplay(t *testing.T) {
	registry := linkedHiveRegistryWithRules(t, `,TriggerA(WhenAPermanentLeavesTheBattlefield(TheCreatedTokens),ActionList([SacrificePermanent(ThisPermanent)]))`)
	factory := func() mage.Card {
		card, err := registry.CreateCard("The Hive")
		if err != nil {
			t.Fatal(err)
		}
		return card
	}
	g, a, b := ronGame()
	source, other := g.PutOnBattlefield(factory(), a.PlayerID()), g.PutOnBattlefield(factory(), b.PlayerID())
	for _, p := range []*mage.Permanent{source, other} {
		g.GetPlayer(p.ControllerID()).ManaPool().Add(core.Colorless, 5)
		if err := g.ActivateAbilityByIndex(p.ControllerID(), p.ID(), 0, nil); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
	}
	tokens := g.FilterBattlefield(mage.IsToken)
	var member *mage.Permanent
	for _, p := range tokens {
		if p.ControllerID() == a.PlayerID() {
			member = p
		}
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: member.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.ExilePermanent(member)
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("wanted one departure trigger, got %d", g.StackSize())
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("trigger copy failed")
	}
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.ResolveStack()
		if game.FindPermanent(source.ID()) != nil || game.FindPermanent(other.ID()) == nil || len(game.FilterBattlefield(mage.IsToken)) != 1 {
			t.Fatal("departure crossed token groups")
		}
	}
}

func TestRONCreatedTokenLeaveInvalid(t *testing.T) {
	create := `Activated(PayMana([]),ActionList([CreateTokens([` + saprolingRON + `],[])]))`
	for _, subject := range []string{`TheCreatedTokens()`, `SinglePermanent(TheCreatedTokens)`, `TheTokensCreatedThisWay`, `AnyPermanent`} {
		src := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + create + `,TriggerA(WhenAPermanentLeavesTheBattlefield(` + subject + `),ActionList([GainLife(Integer(1))]))])`
		if registry, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil || registry != nil {
			t.Fatalf("accepted %s", subject)
		}
	}
	valid := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + create + `,TriggerA(WhenAPermanentLeavesTheBattlefield(TheCreatedTokens),ActionList([GainLife(Integer(1))]))])`
	for _, src := range []string{strings.Replace(valid, create+`,`, "", 1), strings.Replace(valid, create, create+`,`+create, 1), strings.Replace(valid, `GainLife(Integer(1))`, `SacrificePermanent(Trigger_ThatPermanent)`, 1)} {
		if registry, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(src)}}, "*.ron"); err == nil || registry != nil {
			t.Fatal("accepted ambiguous link or unbound event read")
		}
	}
}

func TestRONCreatedTokenLeaveSourceReturns(t *testing.T) {
	registry := linkedHiveRegistryWithRules(t, `,TriggerA(WhenAPermanentLeavesTheBattlefield(TheCreatedTokens),ActionList([SacrificePermanent(ThisPermanent)]))`)
	g, a, b := ronGame()
	card, err := registry.CreateCard("The Hive")
	if err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	a.ManaPool().Add(core.Colorless, 5)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	token := g.FilterBattlefield(mage.IsToken)[0]
	phase := g.PhaseOut(token.ID())
	g.ResolveStack()
	if g.FindPermanent(source.ID()) == nil {
		t.Fatal("phasing caused sacrifice")
	}
	g.PhaseIn(phase)
	g.DestroyPermanent(token)
	g.BouncePermanentToHand(source)
	a.RemoveFromHand(card.ID())
	returned := g.PutOnBattlefield(card, b.PlayerID())
	clone := g.Clone()
	for _, game := range []*mage.Game{g, clone} {
		game.ResolveStack()
		if game.FindPermanent(returned.ID()) == nil {
			t.Fatal("queued sacrifice used returned source")
		}
	}
}

func TestRONCreatedTokenLeaveCollectionAtomic(t *testing.T) {
	registry := linkedHiveRegistryWithRules(t, `,TriggerA(WhenAPermanentLeavesTheBattlefield(TheCreatedTokens),ActionList([SacrificePermanent(ThisPermanent)]))`)
	if card := repositoryCard(t, "DRK", "Dance of Many"); mage.ValidateCardDefinition(card.Name(), card) != nil {
		t.Fatal("complete token-copy consumer rejected")
	}
	for _, tc := range []struct{ set, name string }{{"LEG", "Stangg"}, {"ATQ", "Tetravus"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for line := range strings.SplitSeq(string(data), "\n") {
			if !strings.Contains(line, `Name: "`+tc.name+`"`) {
				continue
			}
			found = true
			if err := LoadInto(registry, fstest.MapFS{"real.ron": {Data: []byte(line)}}, "*.ron"); err == nil {
				t.Fatalf("published incomplete %s", tc.name)
			}
			if names := registry.RegisteredCardNames(); len(names) != 1 || names[0] != "The Hive" {
				t.Fatal("collection failure replaced the registry")
			}
		}
		if !found {
			t.Fatalf("missing %s record", tc.name)
		}
	}
}

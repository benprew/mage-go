package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTokenCopyPermanentDance(t *testing.T) {
	for _, end := range []string{"source", "token", "phase", "upkeep-paid", "upkeep-unpaid"} {
		t.Run(end, func(t *testing.T) {
			a := &ronPaymentPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: end == "upkeep-paid"}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			donor := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), b.PlayerID())
			card := repositoryCard(t, "DRK", "Dance of Many")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
				t.Fatal("unpaid cast accepted")
			}
			if len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("failed cast changed state")
			}
			a.ManaPool().Add(core.Blue, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			tokens := g.FilterBattlefield(mage.IsToken)
			if len(tokens) != 1 || tokens[0].Name() != "Air Elemental" || !tokens[0].HasKeyword(core.Flying) || tokens[0].CurrentPower(g) != 4 || tokens[0].Card.ManaCost().CMC() != 5 || tokens[0].Card.Owner() != a.PlayerID() {
				t.Fatal("wrong copied token")
			}
			source := g.FindPermanent(card.ID())
			token := tokens[0]
			switch end {
			case "source":
				g.BouncePermanentToHand(source)
			case "token":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: donor.ID(), TargetID: token.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				g.ExilePermanent(token)
			case "phase":
				phase := g.PhaseOut(token.ID())
				g.ResolveStack()
				if g.FindPermanent(source.ID()) == nil {
					t.Fatal("phasing caused sacrifice")
				}
				g.PhaseIn(phase)
				g.DestroyPermanent(token)
			case "upkeep-paid":
				a.ManaPool().Add(core.Blue, 2)
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			case "upkeep-unpaid":
				g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
			}
			g.ResolveStack()
			if end == "upkeep-paid" {
				if g.FindPermanent(source.ID()) == nil || len(g.FilterBattlefield(mage.IsToken)) != 1 || a.ManaPool().TotalMana() != 0 {
					t.Fatal("paid upkeep failed")
				}
				return
			}
			if len(g.FilterBattlefield(mage.IsToken)) != 0 || g.FindPermanent(donor.ID()) == nil {
				t.Fatal("linked cleanup failed")
			}
			if end != "source" && g.FindPermanent(source.ID()) != nil {
				t.Fatal("missing source sacrifice")
			}
		})
	}
}

func TestRONTokenCopyPermanentPending(t *testing.T) {
	for _, state := range []string{"copied", "target-returned", "target-phased", "source-returned"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			donor := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), b.PlayerID())
			card := repositoryCard(t, "DRK", "Dance of Many")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			g.PutTriggersOnStack()
			if g.StackSize() != 1 {
				t.Fatal("missing entry trigger")
			}
			switch state {
			case "copied":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "target-returned":
				g.RemoveFromBattlefield(donor)
				g.PutOnBattlefield(donor.Card, b.PlayerID())
			case "target-phased":
				g.PhaseOut(donor.ID())
			case "source-returned":
				g.BouncePermanentToHand(source)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
				g.PutTriggersOnStack()
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveStack()
				want := 1
				if state == "copied" {
					want = 2
				}
				if state == "target-returned" || state == "target-phased" {
					want = 0
				}
				if state == "source-returned" {
					want = 2
				}
				if len(game.FilterBattlefield(mage.IsToken)) != want {
					t.Fatalf("%s tokens: %d want %d", state, len(game.FilterBattlefield(mage.IsToken)), want)
				}
				if state == "copied" {
					owners := map[uuid.UUID]int{}
					for _, p := range game.FilterBattlefield(mage.IsToken) {
						owners[p.Card.Owner()]++
					}
					if owners[a.PlayerID()] != 1 || owners[b.PlayerID()] != 1 {
						t.Fatal("copy lost its controller")
					}
					game.BouncePermanentToHand(game.FindPermanent(source.ID()))
					game.ResolveStack()
					if len(game.FilterBattlefield(mage.IsToken)) != 0 {
						t.Fatal("copied trigger lost original link")
					}
				}
				if state == "source-returned" {
					game.BouncePermanentToHand(game.FindPermanent(source.ID()))
					game.ResolveStack()
					tokens := game.FilterBattlefield(mage.IsToken)
					if len(tokens) != 1 || tokens[0].Card.Owner() != a.PlayerID() {
						t.Fatal("returned source consumed former link")
					}
				}
			}
		})
	}
}

func TestRONTokenCopyPermanentInvalid(t *testing.T) {
	valid := `Card(Name:"Copy",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(AnyPermanent)],ActionList([CreateTokens([TokenCopyOfPermanent(Ref_TargetPermanent,NoTokenCopyEffects)],[])])))])`
	for _, bad := range []string{
		strings.Replace(valid, "NoTokenCopyEffects", "NoTokenCopyEffects()", 1), strings.Replace(valid, "NoTokenCopyEffects", "CopyEffects([])", 1), strings.Replace(valid, "Ref_TargetPermanent", "Ref_TargetPermanent2", 1), strings.Replace(valid, "Ref_TargetPermanent", "ThisPermanent", 1), strings.Replace(valid, "Ref_TargetPermanent", "TheCreatedTokens", 1), strings.Replace(valid, "Ref_TargetPermanent,NoTokenCopyEffects", "Ref_TargetPermanent", 1),
	} {
		registry, err := LoadFS(fstest.MapFS{"bad.ron": {Data: []byte(bad)}}, "*.ron")
		if err == nil || registry != nil {
			t.Fatal("invalid copy accepted", bad)
		}
	}
	registry, err := LoadFS(fstest.MapFS{"good.ron": {Data: []byte(valid)}}, "*.ron")
	if err != nil || registry == nil {
		t.Fatal("valid resolving spell copy rejected", err)
	}
}

func TestRONTokenCopyPermanentInventory(t *testing.T) {
	record := repositoryRecord(t, "DRK", "Dance of Many", "DRK.ron")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := mage.ValidateCardDefinition("Dance of Many", factory()); err != nil {
		t.Fatal(err)
	}
	src := `Card(Name:"Copy",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([CreateTokens([TokenCopyOfPermanent(ThisPermanent,NoTokenCopyEffects)],[])]))])`
	report, err := InspectFS(fstest.MapFS{"copy.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{"ThisPermanent": "permanent_reference", "NoTokenCopyEffects": "token_copy_modification"}
	for name, role := range expected {
		found := false
		for _, entry := range report.Signatures {
			if entry.Name == name && entry.Role == role {
				found = true
				if entry.Support != "unverified" {
					t.Fatal("inventory claims gameplay")
				}
			}
		}
		if !found {
			t.Fatalf("missing %s role %s", name, role)
		}
	}
}

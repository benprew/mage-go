package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONGraveRobbersExile(t *testing.T) {
	for _, change := range []string{"normal", "own", "absent", "returned", "source-leaves", "source-control", "source-phases", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Grave Robbers")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			victim := repositoryCard(t, "LEB", "Sol Ring")
			holder := b
			if change == "own" {
				holder = a
			}
			victim.SetOwner(holder.PlayerID())
			holder.AddToGraveyard(victim)
			a.ManaPool().Add(core.Black, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 || a.Life() != 20 || g.FindExiledCard(victim.ID()) != nil {
				t.Fatal("announcement or early effect")
			}
			switch change {
			case "absent", "returned":
				moved, ok := g.MoveFromGraveyard(holder.PlayerID(), victim.ID(), core.ZoneHand)
				if !ok {
					t.Fatal("move failed")
				}
				if change == "returned" {
					holder.AddToGraveyard(moved)
				} else {
					holder.AddToHand(moved)
				}
			case "source-leaves":
				g.DestroyPermanent(g.FindPermanent(source.ID()))
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "source-phases":
				g.PhaseOut(source.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			legal := change != "absent" && change != "returned"
			wantLife := 20
			if legal {
				wantLife = 22
			}
			if a.Life() != wantLife || b.Life() != 20 || (g.FindExiledCard(victim.ID()) != nil) != legal {
				t.Fatal("exile, legality, or saved controller")
			}
			if legal && (g.FindExiledCard(victim.ID()).Card.Owner() != holder.PlayerID() || len(holder.Graveyard()) != 0) {
				t.Fatal("wrong owner or duplicate zone")
			}
			if clone.FindExiledCard(victim.ID()) != nil || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("clone changed")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != wantLife || (clone.FindExiledCard(victim.ID()) != nil) != legal {
				t.Fatal("clone lost target visit")
			}
		})
	}
}

func TestRONGraveRobbersRollback(t *testing.T) {
	for _, reason := range []string{"unpaid", "sick", "tapped", "phased", "wrong-type", "missing", "two-targets"} {
		t.Run(reason, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Grave Robbers"), a.PlayerID())
			if reason != "sick" {
				source.RevokeBaseAttr(core.AttrSummonSick)
			}
			if reason == "tapped" {
				g.TapPermanent(source)
			}
			if reason == "phased" {
				g.PhaseOut(source.ID())
			}
			victim := repositoryCard(t, "LEB", "Sol Ring")
			if reason == "wrong-type" {
				victim = repositoryCard(t, "LEB", "Forest")
			}
			victim.SetOwner(b.PlayerID())
			if reason != "missing" {
				b.AddToGraveyard(victim)
			}
			targets := []uuid.UUID{victim.ID()}
			if reason == "two-targets" {
				targets = append(targets, victim.ID())
			}
			mana := 0
			if reason != "unpaid" {
				a.ManaPool().Add(core.Black, 1)
				mana = 1
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err == nil {
				t.Fatal("invalid activation accepted")
			}
			if a.ManaPool().TotalMana() != mana || a.Life() != 20 || g.StackSize() != 0 || g.FindExiledCard(victim.ID()) != nil {
				t.Fatal("failed activation changed resources")
			}
			if reason != "phased" && g.FindPermanent(source.ID()).Tapped != (reason == "tapped") {
				t.Fatal("failed activation paid tap cost")
			}
		})
	}
}

func TestRONCyclopeanMummyDeathCard(t *testing.T) {
	for _, change := range []string{"normal", "control", "returned", "reentered", "phase", "copy", "ability-lost", "replaced", "other-dies"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Cyclopean Mummy")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			controller := a.PlayerID()
			if change == "control" {
				controller = b.PlayerID()
			}
			source := g.PutOnBattlefield(card, controller)
			switch change {
			case "phase":
				g.PhaseOut(source.ID())
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("phasing triggered death")
				}
				g.PhaseIn([]uuid.UUID{source.ID()})
			case "ability-lost":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), source.ID(), controller, nil); err != nil {
					t.Fatal(err)
				}
			case "replaced":
				g.AddExileIfWouldGoToGraveyardThisTurn(source.ID(), source.ID())
			case "other-dies":
				otherCard := repositoryCard(t, "LEB", "Savannah Lions")
				otherCard.SetOwner(a.PlayerID())
				other := g.PutOnBattlefield(otherCard, a.PlayerID())
				g.DestroyPermanent(other)
				g.PutTriggersOnStack()
				if g.StackSize() != 0 {
					t.Fatal("other death triggered mummy")
				}
			}
			g.DestroyPermanent(g.FindPermanent(source.ID()))
			if change == "returned" || change == "reentered" {
				moved, ok := g.MoveFromGraveyard(a.PlayerID(), card.ID(), core.ZoneHand)
				if !ok {
					t.Fatal("move failed")
				}
				if change == "returned" {
					a.AddToGraveyard(moved)
				} else {
					g.PutOnBattlefield(moved, a.PlayerID())
				}
			}
			g.PutTriggersOnStack()
			want := 1
			if change == "ability-lost" || change == "replaced" {
				want = 0
			}
			if g.StackSize() != want {
				t.Fatalf("triggers = %d, want %d", g.StackSize(), want)
			}
			if change == "copy" && g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			g.ResolveStack()
			exiled := change != "returned" && change != "reentered" && change != "ability-lost"
			if (g.FindExiledCard(card.ID()) != nil) != exiled {
				t.Fatal("death capture followed wrong visit")
			}
			if change == "reentered" && g.FindPermanent(card.ID()) == nil {
				t.Fatal("exiled new incarnation")
			}
			clone.ResolveStack()
			if (clone.FindExiledCard(card.ID()) != nil) != exiled {
				t.Fatal("clone lost death binding")
			}
		})
	}
}

func TestRONGraveyardExileInvalid(t *testing.T) {
	for _, body := range []string{
		"Exile([CardInGraveyards], [])", "Exile([CardInGraveyards()], [])",
		"Exile([CardInGraveyards(Ref_TargetGraveyardCard, You)], [])",
		"Exile([CardInGraveyards(Trigger_ThatGraveyardCard)], [])",
		"Exile([CardInGraveyards(Ref_TargetPermanent)], [])",
		"Exile([CardInGraveyards(Ref_TargetGraveyardCard())], [])",
		"Exile([CardInGraveyards(Ref_TargetGraveyardCard)], [ExilesFaceDown])",
		"Exile([CardInGraveyards(Ref_TargetGraveyardCard), CardInGraveyards(Ref_TargetGraveyardCard)], [])",
	} {
		src := "Card(Name:\"Bad\",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetGraveyardCard(IsCardtype(Artifact))],ActionList([" + body + "])))])"
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
}

func TestRONGraveyardExileIndependentDeaths(t *testing.T) {
	g, a, b := ronGame()
	first := repositoryCard(t, "LEG", "Cyclopean Mummy")
	second := repositoryCard(t, "LEG", "Cyclopean Mummy")
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	g.PutOnBattlefield(first, b.PlayerID())
	g.PutOnBattlefield(second, a.PlayerID())
	if err := mage.ApplyEffect(g, mage.DestroyEach(mage.MatchingPermanents(mage.IsCreature)), uuid.New(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatalf("triggers = %d", g.StackSize())
	}
	clone := g.Clone()
	g.ResolveStack()
	for _, card := range []mage.Card{first, second} {
		if g.FindExiledCard(card.ID()) == nil || clone.FindExiledCard(card.ID()) != nil {
			t.Fatal("shared death binding or clone state")
		}
	}
	clone.ResolveStack()
	if len(clone.GetExile()) != 2 {
		t.Fatal("clone lost independent captures")
	}
}

func TestRONGraveRobbersRetargetedCopy(t *testing.T) {
	a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	g.SetTurn(2)
	g.SetStep(core.PrecombatMain)
	source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Grave Robbers"), a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	first := repositoryCard(t, "LEB", "Sol Ring")
	second := repositoryCard(t, "LEB", "Sol Ring")
	first.SetOwner(a.PlayerID())
	second.SetOwner(b.PlayerID())
	a.AddToGraveyard(first)
	b.AddToGraveyard(second)
	a.ManaPool().Add(core.Black, 1)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	b.targets = []uuid.UUID{second.ID()}
	copied := g.CopyStackObjectDirect(original, b.PlayerID(), true)
	if copied == nil {
		t.Fatal("copy failed")
	}
	g.ResolveTopOfStack()
	if g.FindExiledCard(first.ID()) != nil || g.FindExiledCard(second.ID()) == nil || b.Life() != 22 || a.Life() != 20 || original.Targets[0] != first.ID() {
		t.Fatal("copy target or controller")
	}
	g.ResolveTopOfStack()
	if g.FindExiledCard(first.ID()) == nil || a.Life() != 22 {
		t.Fatal("original binding changed")
	}
}

func TestRONGraveyardExileRegistryAndInventory(t *testing.T) {
	files := fstest.MapFS{}
	for _, tc := range []struct{ set, name string }{{"DRK", "Grave Robbers"}, {"LEG", "Cyclopean Mummy"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), tc.set+".ron")
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			if node.Fields["Name"].String == tc.name {
				files[tc.name+".ron"] = &fstest.MapFile{Data: []byte(strings.Split(string(data), "\n")[node.Position.Line-1])}
			}
		}
	}
	if len(files) != 2 {
		t.Fatal("missing complete source records")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Cyclopean Mummy", "Grave Robbers"} {
		card, err := registry.CreateCard(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := mage.ValidateCardDefinition(name, card); err != nil {
			t.Fatal(err)
		}
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 2 || report.UniqueCards != 2 {
		t.Fatal("wrong compilation counts")
	}
	for _, name := range []string{"Ref_TargetGraveyardCard", "Trigger_ThatGraveyardCard"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == "card_reference" {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing card-reference role for %s", name)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte("Card(Name:\"Bad\",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([Exile([CardInGraveyards(Trigger_ThatGraveyardCard)], [])]))])")}
	if partial, err := LoadFS(files, "*.ron"); err == nil || partial != nil {
		t.Fatal("published partial registry")
	}
	existing, err := registry.CreateCard("Grave Robbers")
	if err != nil || existing == nil {
		t.Fatal("accepted registry lost")
	}
}

func TestRONDeathCardScope(t *testing.T) {
	for _, event := range []string{
		"WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent))",
		"WhenAPermanentDies(SinglePermanent(HostPermanent))",
		"WhenAPermanentDies(SinglePermanent(Ref_TargetPermanent))",
		"WhenAPermanentDies(SinglePermanent(ThisPermanent()))",
	} {
		src := "Card(Name:\"Bad\",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),Rules:[TriggerA(" + event + ",ActionList([Exile([CardInGraveyards(Trigger_ThatGraveyardCard)], [])]))],CardPT:(Power:1,Toughness:1))"
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted incompatible event %s", event)
		}
	}
}

func TestRONDeathCardReturnUsesFirstVisit(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", "TriggerA(WhenAPermanentDies(SinglePermanent(ThisPermanent)),ActionList([PutGraveyardCardIntoHand(Trigger_ThatGraveyardCard)]))")
	g, a, _ := ronGame()
	card := factory()
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.DestroyPermanent(source)
	g.PutTriggersOnStack()
	g.ResolveStack()
	if len(a.Hand()) != 1 || a.Hand()[0].ID() != card.ID() {
		t.Fatal("typed death binding did not reach existing return action")
	}
}

func TestRONDeathCardSeparateVisits(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEG", "Cyclopean Mummy")
	card.SetOwner(a.PlayerID())
	first := g.PutOnBattlefield(card, a.PlayerID())
	g.DestroyPermanent(first)
	g.PutTriggersOnStack()
	old := g.StackPeek()
	moved, ok := g.MoveFromGraveyard(a.PlayerID(), card.ID(), core.ZoneBattlefield)
	if !ok {
		t.Fatal("move failed")
	}
	second := g.PutOnBattlefield(moved, a.PlayerID())
	g.DestroyPermanent(second)
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("lost independent death events")
	}
	if g.CopyStackObjectDirect(old, b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	g.ResolveTopOfStack()
	if g.FindExiledCard(card.ID()) != nil || len(a.Graveyard()) != 1 {
		t.Fatal("old event followed new graveyard visit")
	}
	g.ResolveTopOfStack()
	if g.FindExiledCard(card.ID()) == nil {
		t.Fatal("new event lost its own visit")
	}
	g.ResolveTopOfStack()
	if len(g.GetExile()) != 1 {
		t.Fatal("old event repeated the move")
	}
}

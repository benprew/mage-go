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

func TestRONGraveyardBatchExileReal(t *testing.T) {
	for _, change := range []string{"opponent", "self", "added", "returned", "copy", "countered"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Tormod's Crypt")
			castRONSpell(t, g, a, card)
			g.ResolveStack()
			if g.FindPermanent(card.ID()) == nil {
				t.Fatal("zero-cost artifact did not enter")
			}
			holder := b
			if change == "self" {
				holder = a
			}
			victim := mage.NewArtifact("Victim", "")
			victim.SetOwner(holder.PlayerID())
			holder.AddToGraveyard(victim)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), card.ID(), 0, []uuid.UUID{holder.PlayerID()}); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil || len(a.Graveyard()) == 0 || len(g.GetExile()) != 0 {
				t.Fatal("cost or resolution timing")
			}
			switch change {
			case "added":
				later := mage.NewLand("Later")
				later.SetOwner(holder.PlayerID())
				holder.AddToGraveyard(later)
			case "returned":
				holder.RemoveFromGraveyard(victim.ID())
				holder.AddToGraveyard(victim)
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("missing ability copy")
				}
			case "countered":
				g.CounterStackObject(g.StackPeek().ID)
			}
			clone := g.Clone()
			g.ResolveStack()
			if (g.FindExiledCard(victim.ID()) != nil) != (change != "countered") {
				t.Fatal("wrong resolving graveyard membership")
			}
			if change == "self" && g.FindExiledCard(card.ID()) == nil {
				t.Fatal("paid source did not join its owner's graveyard")
			}
			if change == "added" && len(g.GetExile()) != 2 {
				t.Fatal("announcement saved graveyard membership too soon")
			}
			if change == "countered" && (len(a.Graveyard()) != 1 || a.Graveyard()[0].ID() != card.ID()) {
				t.Fatal("countered ability refunded its source")
			}
			if clone.FindExiledCard(victim.ID()) != nil {
				t.Fatal("resolution changed clone")
			}
			clone.ResolveStack()
			if len(clone.GetExile()) != len(g.GetExile()) {
				t.Fatal("clone lost the pending player reference")
			}
		})
	}
}

func TestRONGraveyardBatchExileInvalid(t *testing.T) {
	for _, subject := range []string{
		`EachCardInGraveyards`, `EachCardInGraveyards()`,
		`EachCardInGraveyards(InAPlayersGraveyard())`,
		`EachCardInGraveyards(InAPlayersGraveyard(You))`,
		`EachCardInGraveyards(InAPlayersGraveyard(SinglePlayer(Trigger_ThatPlayer)))`,
		`EachCardInGraveyards(InAPlayersGraveyard(SinglePlayer(You())))`,
		`EachCardInGraveyards(ThePermanentsSacrificedThisWay)`,
		`NumberCardsInGraveyards(Integer(1),InAPlayersGraveyard(SinglePlayer(You)))`,
	} {
		factory := `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([Exile([` + subject + `],[])]))])`
		nodes, err := Parse(strings.NewReader(factory), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", subject)
		}
	}
}

func TestRONGraveyardBatchExileLoadAndInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var source string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Tormod's Crypt"`) {
			source = line
		}
	}
	files := fstest.MapFS{"DRK.ron": {Data: []byte(source)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Tormod's Crypt")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Tormod's Crypt")
	if err != nil || first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared factory declarations", err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(source, "Ref_TargetPlayer", "Trigger_ThatPlayer", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Tormod's Crypt") {
		t.Fatal("partial collection published")
	}
	delete(files, "bad.ron")
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory rejected complete record", err)
	}
	for _, wanted := range []struct{ name, role string }{{"InAPlayersGraveyard", "graveyard_card_filter"}, {"SinglePlayer", "player_filter"}, {"Ref_TargetPlayer", "player_reference"}} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == wanted.name && signature.Role == wanted.role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing role %s for %s", wanted.role, wanted.name)
		}
	}
}

func TestRONGraveyardBatchExileRollback(t *testing.T) {
	for _, reason := range []string{"missing", "tapped", "phased"} {
		t.Run(reason, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, "DRK", "Tormod's Crypt")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			target := a.PlayerID()
			switch reason {
			case "missing":
				target = uuid.New()
			case "tapped":
				g.TapPermanent(source)
			case "phased":
				g.PhaseOut(source.ID())
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target}); err == nil {
				t.Fatal("illegal activation succeeded")
			}
			if len(a.Graveyard()) != 0 || len(g.GetExile()) != 0 || g.StackSize() != 0 {
				t.Fatal("failed activation spent the source")
			}
		})
	}
}

func TestRONGraveyardBatchExileRetargetedCopy(t *testing.T) {
	a := &ronModePlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	g.SetTurn(2)
	g.SetStep(core.PrecombatMain)
	source := g.PutOnBattlefield(repositoryCard(t, "DRK", "Tormod's Crypt"), a.PlayerID())
	for _, player := range []mage.Player{a, b} {
		card := mage.NewArtifact("Victim", "")
		card.SetOwner(player.PlayerID())
		player.AddToGraveyard(card)
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{b.PlayerID()}); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	b.targets = []uuid.UUID{a.PlayerID()}
	if g.CopyStackObjectDirect(original, b.PlayerID(), true) == nil {
		t.Fatal("missing copy")
	}
	g.ResolveTopOfStack()
	if len(a.Graveyard()) != 0 || len(b.Graveyard()) != 1 || original.Targets[0] != b.PlayerID() || len(g.GetExile()) != 2 {
		t.Fatal("copy shared target state or paid the source again")
	}
	g.ResolveTopOfStack()
	if len(b.Graveyard()) != 0 || len(g.GetExile()) != 3 {
		t.Fatal("original player binding changed")
	}
}

func TestRONGraveyardBatchExileDelayedSource(t *testing.T) {
	for _, state := range []string{"live", "control", "phased", "returned"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),ActionList([CreateFutureTrigger(AtTheBeginningOfTheNextEndStep,ActionList([Exile([EachCardInGraveyards(InAPlayersGraveyard(SinglePlayer(ControllerOfPermanent(ThisPermanent))))],[])]))]))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			original := g.CaptureDamageSource(source.ID())
			for _, player := range []mage.Player{a, b} {
				victim := mage.NewArtifact("Victim", "")
				victim.SetOwner(player.PlayerID())
				player.AddToGraveyard(victim)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			switch state {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phased":
				g.PhaseOut(source.ID())
			case "returned":
				g.BouncePermanentToHand(source)
				a.RemoveFromHand(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			}
			for range 3 {
				got := g.WaitingDelayedPermanentSources()
				if len(got) != 1 || !got[0].SameObject(original) || len(g.GetExile()) != 0 {
					t.Fatal("waiting query lost source or exiled cards")
				}
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.FireEvent(core.GameEvent{Type: core.EvtEndStep, PlayerID: a.PlayerID()})
				game.PutTriggersOnStack()
				game.ResolveStack()
				id, other := a.PlayerID(), b.PlayerID()
				if state == "control" {
					id, other = other, id
				}
				if len(game.GetPlayer(id).Graveyard()) != 0 || len(game.GetPlayer(other).Graveyard()) != 1 || len(game.GetExile()) != 1 || len(game.WaitingDelayedPermanentSources()) != 0 {
					t.Fatal("delayed body lost exact source controller or lifetime")
				}
			}
		})
	}
}

func TestRONGraveyardBatchExileUnsupportedContexts(t *testing.T) {
	valid := `Card(Name:"Exile",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(PayMana([]),ActionList([Exile([EachCardInGraveyards(InAPlayersGraveyard(SinglePlayer(You)))],[])]))])`
	for _, source := range []string{
		strings.Replace(valid, `],[])`, `],[ExilesFaceDown])`, 1),
		strings.Replace(valid, `SinglePlayer(You)`, `AnyPlayer`, 1),
		strings.Replace(valid, `SinglePlayer(You)`, `SinglePlayer(Ref_TargetPlayer)`, 1),
		strings.Replace(valid, `Activated(PayMana([]),ActionList`, `AsPermanentEnters(ThisPermanent,ActionList`, 1),
		strings.Replace(valid, `PayMana([])`, `Exile([EachCardInGraveyards(InAPlayersGraveyard(SinglePlayer(You)))],[])`, 1),
	} {
		nodes, err := Parse(strings.NewReader(source), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted unsupported declaration: %s", source)
		}
	}
}

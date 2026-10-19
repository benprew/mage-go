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

func TestRONDiscardCausePsychicPurge(t *testing.T) {
	for _, mode := range []string{"cast", "effect", "cost", "cleanup", "own-effect", "departed", "two"} {
		t.Run(mode, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Psychic Purge")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if mode == "cast" {
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err == nil {
					t.Fatal("unpaid cast accepted")
				}
				a.ManaPool().Add(core.Blue, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if b.Life() != 19 || a.ManaPool().TotalMana() != 0 || len(a.Graveyard()) != 1 {
					t.Fatal("complete spell did not pay and deal damage")
				}
				return
			}
			controller := b.PlayerID()
			if mode == "own-effect" {
				controller = a.PlayerID()
			}
			cause := g.PutOnBattlefield(mage.NewArtifact("cause", ""), controller)
			switch mode {
			case "cost":
				g.PlayerDiscardByCost(a, card.ID(), cause.ID())
			case "cleanup":
				g.PlayerDiscard(a, card.ID())
			default:

				if mode == "two" {
					second := repositoryCard(t, "LEG", "Psychic Purge")
					second.SetOwner(a.PlayerID())
					a.AddToHand(second)
					g.PlayerDiscardByEffect(a, second.ID(), cause.ID())
				}
				g.PlayerDiscardByEffect(a, card.ID(), cause.ID())
			}
			want := 15
			if mode == "cost" || mode == "cleanup" || mode == "own-effect" {
				want = 20
			}
			if mode == "two" {
				want = 10
			}
			if mode == "departed" {
				c, _ := a.RemoveFromGraveyard(card.ID())
				a.AddToHand(c)
				g.RemoveFromBattlefield(cause)
				g.PutOnBattlefield(cause.Card, a.PlayerID())
			}
			clone := g.Clone()
			g.ResolveStack()
			if b.Life() != want || a.Life() != 20 {
				t.Fatalf("life A=%d B=%d want B=%d", a.Life(), b.Life(), want)
			}
			clone.ResolveStack()
			if clone.GetPlayer(b.PlayerID()).Life() != want {
				t.Fatal("clone lost event binding")
			}
		})
	}
}

func TestRONDiscardCauseDeclarations(t *testing.T) {
	valid := `FromAnyZone(TriggerA(WhenASpellOrAbilityCausesAPlayerToDiscardACard(ControlledByAPlayer(Opponent),SinglePlayer(You),SingleCardInHand(ThisCardInHand)),ActionList([PlayerAction(Trigger_ControllerOfThatSpellOrAbility,LoseLife(Integer(5)))])))`
	for _, rule := range []string{
		strings.Replace(valid, "FromAnyZone(", "FromGraveyard(", 1),
		strings.Replace(valid, "Opponent", "You", 1),
		strings.Replace(valid, "SinglePlayer(You)", "SinglePlayer(Opponent)", 1),
		strings.Replace(valid, "ThisCardInHand", "Ref_TargetCard", 1),
		strings.Replace(valid, "ThisCardInHand", "ThisCardInHand()", 1),
		strings.Replace(valid, "LoseLife(Integer(5))", "CopySpell(ThisSpell)", 1),
		strings.Replace(valid, "SingleCardInHand(ThisCardInHand)", "SingleCardInHand(ThisCardInHand,ThisCardInHand)", 1),
		strings.Replace(valid, "ControlledByAPlayer(Opponent)", "ControlledByAPlayer(Opponent,You)", 1),
		strings.Replace(valid, "TriggerA(", "TriggerI(", 1),
		strings.Replace(valid, "LoseLife(Integer(5))", "SpellDealsDamage(ThisSpell,Integer(5),You)", 1),
		`SpellActions(ActionList([PlayerAction(Trigger_ControllerOfThatSpellOrAbility,LoseLife(Integer(5)))]))`,
		`FromAnyZone(TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([])))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
}

func TestRONDiscardCauseLoad(t *testing.T) {
	record := repositoryRecord(t, "LEG", "Psychic Purge", "LEG.ron")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	if err = mage.ValidateCardDefinition("Psychic Purge", factory()); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var source string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Psychic Purge"`) {
			source = line
			break
		}
	}
	if source == "" {
		t.Fatal("missing complete record")
	}
	registry := mage.NewRegistry()
	files := fstest.MapFS{"purge.ron": {Data: []byte(source)}}
	if err = LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(source, "SinglePlayer(You)", "SinglePlayer(Opponent)", 1))}
	if err = LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("invalid collection accepted")
	}
	if !registry.CardRegistered("Psychic Purge") {
		t.Fatal("failed load replaced registry")
	}
	delete(files, "invalid.ron")
	inventory, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"FromAnyZone": "rule", "WhenASpellOrAbilityCausesAPlayerToDiscardACard": "event", "ControlledByAPlayer": "discard_cause_filter", "SingleCardInHand": "hand_card_set", "ThisCardInHand": "hand_card_reference", "Trigger_ControllerOfThatSpellOrAbility": "player_reference"} {
		found := false
		for _, sig := range inventory.Signatures {
			if sig.Name == name && sig.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing inventory role %s/%s", role, name)
		}
	}

}

func TestRONDiscardCauseSpellAndActivation(t *testing.T) {
	for _, mode := range []string{"spell", "ability", "returned-source", "copy"} {
		t.Run(mode, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetActivePlayerIndex(1)
			card := repositoryCard(t, "LEG", "Psychic Purge")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			body := `Targeted([TargetPlayer(AnyPlayer)],ActionList([PlayerAction(Ref_TargetPlayer,DiscardACard)]))`
			if mode == "spell" {
				source := permanentScopeFactory(t, "Instant", `SpellActions(`+body+`)`)()
				source.SetOwner(b.PlayerID())
				b.AddToHand(source)
				if err := g.CastSpellByID(b.PlayerID(), source.ID(), []uuid.UUID{a.PlayerID()}, 0); err != nil {
					t.Fatal(err)
				}
			} else {
				source := permanentScopeFactory(t, "Artifact", `Activated(SacrificePermanent(ThisPermanent),`+body+`)`)()
				source.SetOwner(b.PlayerID())
				p := g.PutOnBattlefield(source, b.PlayerID())
				if err := g.ActivateAbilityByIndex(b.PlayerID(), p.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				if mode == "returned-source" {
					c, _ := b.RemoveFromGraveyard(source.ID())
					g.PutOnBattlefield(c, a.PlayerID())
				}
			}
			g.ResolveTopOfStack()
			if g.StackSize() != 1 || b.Life() != 20 {
				t.Fatal("discard trigger did not wait for priority")
			}
			want := 15
			if mode == "copy" {
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				want = 10
			}
			g.ResolveStack()
			if b.Life() != want || a.Life() != 20 {
				t.Fatal("stack effect cause controller was lost")
			}
		})
	}
}

func TestRONDiscardCauseHiddenDestination(t *testing.T) {
	a := &ronDiscardLibraryPlayer{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.PutOnBattlefield(mage.NewArtifact("discard replacement", "", mage.WithStaticAbility(mage.DiscardToLibraryReplacement())), a.PlayerID())
	source := g.PutOnBattlefield(mage.NewArtifact("cause", ""), b.PlayerID())
	card := repositoryCard(t, "LEG", "Psychic Purge")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if _, ok := g.PlayerDiscardByEffect(a, card.ID(), source.ID()); !ok {
		t.Fatal("replacement did not discard")
	}
	g.ResolveStack()
	if b.Life() != 20 || len(a.Library()) != 1 || len(a.Graveyard()) != 0 {
		t.Fatal("hidden card supplied an ability")
	}
}

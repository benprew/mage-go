package ron

import (
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONDependentGraveyardDrafna(t *testing.T) {
	for _, count := range []int{0, 1, 2} {
		g, a, b := ronGame()
		card := repositoryCard(t, "ATQ", "Drafna's Restoration")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		first, second := mage.NewArtifact("First", ""), mage.NewArtifact("Second", "")
		first.SetOwner(b.PlayerID())
		second.SetOwner(b.PlayerID())
		b.AddToGraveyard(first)
		b.AddToGraveyard(second)
		ids := []uuid.UUID{b.PlayerID(), first.ID(), second.ID()}
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids[:count+1], 0); err == nil {
			t.Fatal("unpaid cast succeeded")
		}
		a.ManaPool().Add(core.Blue, 1)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids[:count+1], 0); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		if len(b.Library()) != count || len(b.Graveyard()) != 2-count || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
			t.Fatal("complete record failed")
		}
	}
}
func TestRONDependentGraveyardInvalid(t *testing.T) {
	record := staticBaseRecord(t, "ATQ", "Drafna's Restoration")
	src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Sorcery],Subtypes:[]),ManaCost:[],Rules:[SpellActions(Targeted([TargetPlayer(AnyPlayer),AnyNumberOfTargetGraveyardCards(And([IsCardtype(Artifact),InAPlayersGraveyard(SinglePlayer(Ref_TargetPlayer))]))],ActionList([ReturnGraveyardCardsToTopOfLibraryInAnyOrder(Ref_TargetGraveyardCards)])))])`
	for _, bad := range []string{
		strings.Replace(src, "SinglePlayer(Ref_TargetPlayer)", "SinglePlayer(You)", 1),
		strings.Replace(src, "TargetPlayer(AnyPlayer),", "", 1),
		strings.Replace(src, "SinglePlayer(Ref_TargetPlayer)", "SinglePlayer(Ref_TargetPlayer,You)", 1),
		strings.Replace(src, "SinglePlayer(Ref_TargetPlayer)", "SinglePlayer(ActionPlayer)", 1),
		strings.Replace(src, "TargetPlayer(AnyPlayer)", "TargetPlayer(AnyPlayer,Opponent)", 1),
		strings.Replace(src, "IsCardtype(Artifact)", "IsCardtype(Aura)", 1),
		strings.Replace(src, "IsCardtype(Artifact)", "InAPlayersGraveyard(SinglePlayer(Ref_TargetPlayer))", 1),
		strings.Replace(src, "SinglePlayer(Ref_TargetPlayer)", "AnyPlayer", 1),

		strings.Replace(src, "Ref_TargetGraveyardCards)]", "Ref_TargetGraveyardCard)]", 1),
		strings.Replace(src, "AnyNumberOfTargetGraveyardCards(And(", "AnyNumberOfTargetGraveyardCards(AndX(", 1),
		strings.Replace(src, "IsCardtype(Artifact),", "", 1),
		strings.Replace(src, "ReturnGraveyardCardsToTopOfLibraryInAnyOrder(Ref_TargetGraveyardCards)", "ReturnGraveyardCardsToTopOfLibraryInAnyOrder", 1),
	} {
		nodes, err := Parse(strings.NewReader(bad), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", bad)
		}
	}
	f, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	if slices.Equal(f().CastTargets(), f().CastTargets()) {
		t.Fatal("factory reused target declarations")
	}
}

func TestRONDependentGraveyardOrderAndVisits(t *testing.T) {
	for _, scenario := range []string{"order", "partial", "return", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			a := &ronLibraryViewer{BasePlayer: mage.NewBasePlayer("A")}
			b := mage.NewBasePlayer("B")
			g := mage.NewGame(a, b)
			g.SetActivePlayerIndex(0)
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "ATQ", "Drafna's Restoration")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			first, second := mage.NewArtifact("First", ""), mage.NewArtifact("Second", "")
			first.SetOwner(b.PlayerID())
			second.SetOwner(b.PlayerID())
			b.AddToGraveyard(first)
			b.AddToGraveyard(second)
			creature := mage.NewCreature("Wrong type", "", 1, 1)
			creature.SetOwner(b.PlayerID())
			b.AddToGraveyard(creature)
			a.ManaPool().Add(core.Blue, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID(), creature.ID()}, 0); err == nil || a.ManaPool().TotalMana() != 1 {
				t.Fatal("wrong type paid")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID(), first.ID(), second.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if scenario == "partial" || scenario == "return" {
				b.RemoveFromGraveyard(first.ID())
				if scenario == "return" {
					b.AddToGraveyard(first)
				} else {
					b.AddToHand(first)
				}
			}
			clone := g.Clone()
			if scenario == "clone" {
				clone.GetPlayer(b.PlayerID()).RemoveFromGraveyard(first.ID())
			}
			g.ResolveStack()
			if scenario == "order" || scenario == "clone" {
				if len(b.Library()) != 2 || b.Library()[0].ID() != second.ID() || b.Library()[1].ID() != first.ID() || a.orders != 1 {
					t.Fatal("caster ordering failed")
				}
			} else if len(b.Library()) != 1 || b.Library()[0].ID() != second.ID() {
				t.Fatal("wrong visit returned")
			}
			if len(clone.GetPlayer(b.PlayerID()).Library()) != 0 {
				t.Fatal("clone shared library")
			}
			clone.ResolveStack()
			if scenario == "clone" && len(clone.GetPlayer(b.PlayerID()).Library()) != 1 {
				t.Fatal("clone used absent target")
			}
		})
	}
}
func TestRONDependentGraveyardLoad(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/ATQ.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Drafna's Restoration"`) {
			record = line
		}
	}
	if record == "" {
		t.Fatal("missing full record")
	}
	files := fstest.MapFS{"drafna.ron": {Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("wrong registry")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 || len(report.Issues) != 0 {
		t.Fatal("inventory rejected supported record", err)
	}

	expected := map[string]string{"AnyNumberOfTargetGraveyardCards": "target", "Ref_TargetGraveyardCards": "card_collection_reference"}
	for name, role := range expected {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role && signature.CompiledOccurrences == 1 {
				found = true
			}
		}
		if !found {
			t.Fatal("missing semantic inventory role", name, role)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(record, "Ref_TargetGraveyardCards)", "UnboundCards)", 1))}
	if partial, err := LoadFS(files, "*.ron"); err == nil || partial != nil {
		t.Fatal("partial collection escaped")
	}
}

func TestRONDependentGraveyardContexts(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`ReturnGraveyardCardsToTopOfLibraryInAnyOrder(Ref_TargetGraveyardCards)`), "context.ron")
	if err != nil {
		t.Fatal(err)
	}
	valid := scope{spellTargets: true, resolving: true, graveyardTargets: &graveyardTargetsPlan{name: "Ref_TargetGraveyardCards"}}
	for _, state := range []string{"unbound", "entry", "draw", "delayed", "activation", "static", "permanent"} {
		ctx := valid
		switch state {
		case "unbound":
			ctx.graveyardTargets = nil
		case "entry":
			ctx.enteringSource = true
		case "draw":
			ctx.drawReplacementBody = true
		case "delayed":
			ctx.delayedStep = true
		case "activation":
			ctx.activation = true
		case "static":
			ctx.resolving = false
		case "permanent":
			ctx.permanent = true
		}
		if _, err := compileEffect(nodes[0], ctx); err == nil {
			t.Fatal("invalid context accepted", state)
		}
	}
}

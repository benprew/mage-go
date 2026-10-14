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

type ronVariantChooser struct {
	*mage.BasePlayer
	answer, choices int
}

func (p *ronVariantChooser) ChooseMode(_ []string, _ string) int { p.choices++; return p.answer }

func TestRONEntryVariantReal(t *testing.T) {
	for _, set := range []string{"ATQ", "3ED"} {
		for choice, want := range [][2]int{{3, 3}, {2, 2}, {1, 6}} {
			t.Run(set+string(rune('0'+choice)), func(t *testing.T) {
				factory, err := Compile(repositoryRecord(t, set, "Primal Clay", set+".lines.ron"))
				if err != nil {
					t.Fatal(err)
				}
				a := &ronVariantChooser{BasePlayer: mage.NewBasePlayer("A"), answer: choice}
				b := mage.NewBasePlayer("B")
				g := mage.NewGame(a, b)
				g.SetStep(core.PrecombatMain)
				card := factory()
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
					t.Fatal("unpaid cast accepted")
				}
				if a.choices != 0 || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("unpaid cast chose or moved")
				}
				a.RemoveFromHand(card.ID())
				castRONSpell(t, g, a.BasePlayer, card)
				if a.choices != 0 {
					t.Fatal("choice at announcement")
				}
				g.ResolveStack()
				p := g.FindPermanent(card.ID())
				if p == nil {
					t.Fatal("missing entrant")
				}
				checkConditionalPT(t, g, p.ID(), want[0], want[1])
				if !p.HasType(core.TypeArtifact) || !p.HasType(core.TypeCreature) || !p.HasSubType("Shapeshifter") || p.HasKeyword(core.Flying) != (choice == 1) || p.HasKeyword(core.Defender) != (choice == 2) || p.HasSubType("Wall") != (choice == 2) || a.choices != 1 {
					t.Fatal("wrong complete variant")
				}
				blocker := g.PutOnBattlefield(mage.NewCreature("Ground blocker", "", 2, 2), b.PlayerID())
				if mage.CanBlock(blocker, p, g) == (choice == 1) {
					t.Fatal("flying block legality")
				}
				g.MutablePermanent(p.ID()).RevokeBaseAttr(core.AttrSummonSick)
				g.SetStep(core.DeclareAttackers)
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
				if g.GetCombat().IsAttacking(p.ID()) == (choice == 2) {
					t.Fatal("defender attack legality")
				}
				clone := g.Clone()
				g.PhaseIn(g.PhaseOut(p.ID()))
				g.ApplyContinuousEffects()
				checkConditionalPT(t, g, p.ID(), want[0], want[1])
				if a.choices != 1 {
					t.Fatal("phase choice repeated")
				}
				g.BouncePermanentToHand(g.FindPermanent(p.ID()))
				returned := a.Hand()[0]
				if returned.Power() != 0 || returned.Toughness() != 0 {
					t.Fatal("printed departure lost")
				}
				a.RemoveFromHand(returned.ID())
				a.answer = (choice + 1) % 3
				g.PutOnBattlefield(returned, a.PlayerID())
				checkConditionalPT(t, g, p.ID(), [][2]int{{3, 3}, {2, 2}, {1, 6}}[a.answer][0], [][2]int{{3, 3}, {2, 2}, {1, 6}}[a.answer][1])
				checkConditionalPT(t, clone, p.ID(), want[0], want[1])
			})
		}
	}
}

func TestRONEntryVariantInvalid(t *testing.T) {
	for _, operation := range []string{
		`EntersWithLayerEffectOfChoice()`, `EntersWithLayerEffectOfChoice([])`, `EntersWithLayerEffectOfChoice([[]])`,
		`EntersWithLayerEffectOfChoice([[SetCopiablePT(PT(1))]])`, `EntersWithLayerEffectOfChoice([[SetCopiablePT(PT(Integer(1),2))]])`,
		`EntersWithLayerEffectOfChoice([[AddCopiableCardtype(Instant)]])`, `EntersWithLayerEffectOfChoice([[AddCopiableAbility([])]])`,
		`EntersWithLayerEffectOfChoice([[AddCopiableAbility([Unknown])]])`, `EntersWithLayerEffectOfChoice([[AddCopiableCreatureType()]])`,
		`EntersWithLayerEffectOfChoice([[AdjustPT(1,1)]])`,
	} {
		record := repositoryRecord(t, "ATQ", "Primal Clay", "ATQ.lines.ron")
		nodes, err := Parse(strings.NewReader(`AsPermanentEnters(ThisPermanent,[`+operation+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		record.Fields["Rules"] = Node{Kind: List, Args: nodes}
		if _, err := Compile(record); err == nil {
			t.Fatalf("accepted %s", operation)
		}
	}
}

func TestRONEntryVariantLoad(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"ATQ", "3ED"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "Primal Clay"`) {
				files[set+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	if len(files) != 2 {
		t.Fatal("reprint records missing")
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Primal Clay")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Primal Clay")
	if err != nil {
		t.Fatal(err)
	}
	if first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("factory shared an ability")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[AsPermanentEnters(ThisPermanent,[EntersWithLayerEffectOfChoice([[]])])])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("invalid collection accepted")
	}
	if !registry.CardRegistered("Primal Clay") || registry.CardRegistered("Bad") {
		t.Fatal("failed load changed registry")
	}
	for _, rule := range []string{
		`AsPermanentEnters(HostPermanent,[EntersWithLayerEffectOfChoice([[SetCopiablePT(PT(1,1))]])])`,
		`PermanentLayerEffect(ThisPermanent,[EntersWithLayerEffectOfChoice([[SetCopiablePT(PT(1,1))]])])`,
		`Activated(PayMana([]),ActionList([EntersWithLayerEffectOfChoice([[SetCopiablePT(PT(1,1))]])]))`,
	} {
		assertConditionalRejected(t, rule)
	}
}

func TestRONEntryVariantSpellCopy(t *testing.T) {
	factory, err := Compile(repositoryRecord(t, "ATQ", "Primal Clay", "ATQ.lines.ron"))
	if err != nil {
		t.Fatal(err)
	}
	a := &ronVariantChooser{BasePlayer: mage.NewBasePlayer("A"), answer: 2}
	b := &ronVariantChooser{BasePlayer: mage.NewBasePlayer("B"), answer: 1}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	card := factory()
	castRONSpell(t, g, a.BasePlayer, card)
	copySpell := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copySpell == nil {
		t.Fatal("missing spell copy")
	}
	g.ResolveTopOfStack()
	p := g.FindPermanent(copySpell.SourceID)
	if p == nil || !p.IsToken || !p.Card.IsToken() || p.ControllerID() != b.PlayerID() || p.Card.Owner() != b.PlayerID() {
		t.Fatal("copied permanent spell did not enter as token")
	}
	checkConditionalPT(t, g, p.ID(), 2, 2)
	if !p.HasKeyword(core.Flying) || b.choices != 1 || a.choices != 0 {
		t.Fatal("copy did not choose separately")
	}
	clone := g.Clone()
	g.ResolveStack()
	checkConditionalPT(t, g, card.ID(), 1, 6)
	if a.choices != 1 || !g.FindPermanent(card.ID()).HasKeyword(core.Defender) {
		t.Fatal("original did not choose separately")
	}
	g.BouncePermanentToHand(g.FindPermanent(p.ID()))
	if len(b.Hand()) != 0 || clone.FindPermanent(p.ID()) == nil {
		t.Fatal("token departure or clone isolation")
	}
}

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

type ronNumericChooser struct {
	*mage.BasePlayer
	answer, calls int
	accept        bool
}

func (p *ronNumericChooser) ChooseNumber(_, _ int, _ string) int { p.calls++; return p.answer }
func (p *ronNumericChooser) ChooseMayAbility(string) bool        { return p.accept }

func TestRONNumericChoiceReal(t *testing.T) {
	for number := 0; number <= 7; number++ {
		factory, err := Compile(repositoryRecord(t, "ATQ", "Shapeshifter", "ATQ.lines.ron"))
		if err != nil {
			t.Fatal(err)
		}
		a := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("A"), answer: number, accept: true}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
			t.Fatal("unpaid cast")
		}
		if a.calls != 0 || len(a.Hand()) != 1 {
			t.Fatal("failed payment chose")
		}
		a.RemoveFromHand(card.ID())
		castRONSpell(t, g, a.BasePlayer, card)
		if a.calls != 0 {
			t.Fatal("choice at announcement")
		}
		g.ResolveStack()
		if number == 7 {
			if g.FindPermanent(card.ID()) != nil || len(a.Graveyard()) != 1 || a.calls != 1 {
				t.Fatal("zero toughness state action")
			}
			continue
		}
		checkConditionalPT(t, g, card.ID(), number, 7-number)
		p := g.FindPermanent(card.ID())
		if p == nil || !p.HasType(core.TypeArtifact) || !p.HasSubType("Shapeshifter") {
			t.Fatal("metadata")
		}
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: b.PlayerID()})
		g.ResolveStack()
		if a.calls != 1 {
			t.Fatal("opponent upkeep")
		}
		a.accept = false
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		g.ResolveStack()
		checkConditionalPT(t, g, card.ID(), number, 7-number)
		if a.calls != 1 {
			t.Fatal("decline chose")
		}
		a.accept = true
		a.answer = (number + 1) % 8
		g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
		if a.calls != 1 {
			t.Fatal("choice before resolution")
		}
		clone := g.Clone()
		g.ResolveStack()
		clone.ResolveStack()
		if a.answer == 7 {
			if g.FindPermanent(card.ID()) != nil || len(a.Graveyard()) != 1 {
				t.Fatal("updated zero toughness state action")
			}
			continue
		}
		checkConditionalPT(t, g, card.ID(), a.answer, 7-a.answer)
		g.PhaseIn(g.PhaseOut(card.ID()))
		checkConditionalPT(t, g, card.ID(), a.answer, 7-a.answer)
	}
}
func TestRONNumericChoiceInvalid(t *testing.T) {
	record := repositoryRecord(t, "ATQ", "Shapeshifter", "ATQ.lines.ron")
	for _, rules := range []string{
		`AsPermanentEnters(ThisPermanent,[ChooseANumberBetween(-1,7)])`,
		`AsPermanentEnters(ThisPermanent,[ChooseANumberBetween(7,0)])`,
		`AsPermanentEnters(ThisPermanent,[ChooseANumberBetween(Integer(0),7)])`,
		`AsPermanentEnters(ThisPermanent,[ChooseANumberBetween(0)])`,
		`AsPermanentEnters(ThisPermanent,[ChooseANumberBetween(0,7)]),AsPermanentEnters(ThisPermanent,[ChooseANumberBetween(0,7)])`,
		`PermanentLayerEffect(ThisPermanent,[SetPower(TheChosenNumber)])`,
		`TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([MayAction(ChooseANumberBetween(0,7))]))`,
		`AsPermanentEnters(ThisPermanent,[ChooseANumberBetween(0,7)]),PermanentLayerEffect(ThisPermanent,[SetPower(TheChosenNumber())])`,
	} {
		nodes, err := Parse(strings.NewReader("["+rules+"]"), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		record.Fields["Rules"] = nodes[0]
		if _, err := Compile(record); err == nil {
			t.Fatalf("accepted %s", rules)
		}
	}
}

func TestRONNumericChoiceSpellCopyAndReturn(t *testing.T) {
	factory, err := Compile(repositoryRecord(t, "ATQ", "Shapeshifter", "ATQ.lines.ron"))
	if err != nil {
		t.Fatal(err)
	}
	a := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("A"), answer: 2, accept: true}
	b := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("B"), answer: 5, accept: true}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	card := factory()
	castRONSpell(t, g, a.BasePlayer, card)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("copy missing")
	}
	g.ResolveTopOfStack()
	checkConditionalPT(t, g, copied.SourceID, 5, 2)
	p := g.FindPermanent(copied.SourceID)
	if !p.IsToken || p.ControllerID() != b.PlayerID() || a.calls != 0 || b.calls != 1 {
		t.Fatal("copy chooser or token identity")
	}
	g.ResolveStack()
	checkConditionalPT(t, g, card.ID(), 2, 5)
	g.FireEvent(core.GameEvent{Type: core.EvtUpkeep, PlayerID: a.PlayerID()})
	clone := g.Clone()
	g.BouncePermanentToHand(g.FindPermanent(card.ID()))
	a.RemoveFromHand(card.ID())
	a.answer = 4
	returned := g.PutOnBattlefield(card, a.PlayerID())
	g.ResolveStack()
	checkConditionalPT(t, g, returned.ID(), 4, 3)
	if a.calls != 2 {
		t.Fatal("pending update followed return")
	}
	if clone.FindPermanent(card.ID()).CurrentPower(clone) != 2 {
		t.Fatal("clone inherited new visit")
	}
	clone.ResolveStack()
	if g.FindPermanent(returned.ID()).CurrentPower(g) != 4 {
		t.Fatal("clone update changed original")
	}
	g.MutablePermanent(returned.ID()).RevokeBaseAttr(core.AttrSummonSick)
	g.SetStep(core.DeclareAttackers)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{returned.ID()})
	g.ExecuteCombatDamage()
	if b.Life() != 16 {
		t.Fatal("chosen power did not supply native combat damage")
	}
	g.BouncePermanentToHand(g.FindPermanent(copied.SourceID))
	if len(b.Hand()) != 0 {
		t.Fatal("spell copy did not cease as token")
	}
}

func TestRONNumericChoiceLoadAndInstances(t *testing.T) {
	record := repositoryRecord(t, "ATQ", "Shapeshifter", "ATQ.lines.ron")
	factory, err := Compile(record)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../../../../cards/mtgish/ATQ.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var source string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Shapeshifter"`) {
			source = line
		}
	}
	if source == "" {
		t.Fatal("record missing")
	}
	files := fstest.MapFS{"ATQ.ron": &fstest.MapFile{Data: []byte(source)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Shapeshifter")
	if err != nil {
		t.Fatal(err)
	}
	second := factory()
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[PermanentLayerEffect(ThisPermanent,[SetPower(TheChosenNumber)])])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Shapeshifter") || registry.CardRegistered("Bad") {
		t.Fatal("rejected collection changed registry")
	}

	if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
		t.Fatal(err)
	}
	if first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared ability")
	}
	a := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("A"), answer: 1, accept: true}
	b := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("B"), answer: 6, accept: true}
	g := mage.NewGame(a, b)
	left := g.PutOnBattlefield(first, a.PlayerID())
	right := g.PutOnBattlefield(second, b.PlayerID())
	checkConditionalPT(t, g, left.ID(), 1, 6)
	checkConditionalPT(t, g, right.ID(), 6, 1)
	a.answer = 3
	g.RunStepWithPriority(core.Upkeep)
	checkConditionalPT(t, g, left.ID(), 3, 4)
	checkConditionalPT(t, g, right.ID(), 6, 1)
}

func TestRONNumericChoiceContexts(t *testing.T) {
	entry := `AsPermanentEnters(ThisPermanent,[ChooseANumberBetween(0,7)])`
	for _, rule := range []string{
		`AsPermanentEnters(HostPermanent,[ChooseANumberBetween(0,7)])`,
		`AsPermanentEnters(ThisPermanent,[ChooseANumberBetween(0,7),EntersTapped])`,
		entry + `,PermanentLayerEffect(ThisPermanent,[SetPower(TheChosenNumber),SetPower(Integer(1))])`,
		entry + `,PermanentLayerEffect(ThisPermanent,[SetToughness(TheChosenNumber),SetToughness(Integer(1))])`,
		entry + `,EachPermanentLayerEffect(IsCardtype(Creature),[SetPower(TheChosenNumber)])`,
		entry + `,PermanentLayerEffect(ThisPermanent,[SetPower(TheChosenNumber),AddAbility([Flying])])`,
		entry + `,Activated(ChooseANumberBetween(0,7),ActionList([DrawACard]))`,
		entry + `,PermanentLayerEffect(HostPermanent,[AddAbility([TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)),ActionList([MayAction(ChooseANumberBetween(0,7))]))])])`,
	} {
		assertConditionalRejected(t, rule)
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Spell",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+entry+`])`), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("nonpermanent entry accepted")
	}
}

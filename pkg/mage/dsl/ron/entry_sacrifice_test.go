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

func TestRONEntrySacrificeReal(t *testing.T) {
	factory, err := Compile(repositoryRecord(t, "LEG", "Wood Elemental", "LEG.lines.ron"))
	if err != nil {
		t.Fatal(err)
	}
	for _, count := range []int{0, 1, 2} {
		a := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("A"), answer: count}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || a.calls != 0 {
			t.Fatal("failed cast chose")
		}
		a.RemoveFromHand(card.ID())
		for range 2 {
			g.PutOnBattlefield(mage.NewLand("Forest", mage.WithSubTypes("Forest")), a.PlayerID())
		}
		tapped := g.PutOnBattlefield(mage.NewLand("Tapped", mage.WithSubTypes("Forest")), a.PlayerID())
		g.TapPermanent(tapped)
		opponent := g.PutOnBattlefield(mage.NewLand("Opponent", mage.WithSubTypes("Forest")), b.PlayerID())
		g.PutOnBattlefield(mage.NewLand("Plains", mage.WithSubTypes("Plains")), a.PlayerID())
		castRONSpell(t, g, a.BasePlayer, card)
		if a.calls != 0 {
			t.Fatal("announcement chose")
		}
		g.ResolveStack()
		if g.FindPermanent(tapped.ID()) == nil || g.FindPermanent(opponent.ID()) == nil {
			t.Fatal("illegal sacrifice")
		}
		if count == 0 {
			if g.FindPermanent(card.ID()) != nil || len(a.Graveyard()) != 1 {
				t.Fatal("zero toughness state action")
			}
			continue
		}
		checkConditionalPT(t, g, card.ID(), count, count)
		if len(a.Graveyard()) != count || a.calls != 1 {
			t.Fatal("wrong sacrifice count")
		}
		g.MutablePermanent(card.ID()).RevokeBaseAttr(core.AttrSummonSick)
		g.SetStep(core.DeclareAttackers)
		g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{card.ID()})
		g.ExecuteCombatDamage()
		if b.Life() != 20-count {
			t.Fatal("native damage")
		}
	}
}

func TestRONEntrySacrificeInvalid(t *testing.T) {
	for _, rules := range []string{
		`CDA_Power(TheNumberOfPermanentsSacrificedAsPermenantEnteredBattlefield(IsLandType(Forest),ThisPermanent)),CDA_Toughness(TheNumberOfPermanentsSacrificedAsPermenantEnteredBattlefield(IsLandType(Forest),ThisPermanent))`,
		`AsPermanentEnters(HostPermanent,[SacrificeAnyNumberOfPermanents(IsLandType(Forest))])`,
		`AsPermanentEnters(ThisPermanent,[SacrificeAnyNumberOfPermanents(IsLandType(Forest)),EntersTapped])`,
		`AsPermanentEnters(ThisPermanent,[SacrificeAnyNumberOfPermanents(IsLandType(Forest))]),AsPermanentEnters(ThisPermanent,[EntersTapped])`,
		`AsPermanentEnters(ThisPermanent,[SacrificeAnyNumberOfPermanents(SinglePermanent(Ref_TargetPermanent))])`,
		`AsPermanentEnters(ThisPermanent,[SacrificeAnyNumberOfPermanents(And([]))])`,
		`AsPermanentEnters(ThisPermanent,[SacrificeAnyNumberOfPermanents(IsLandType(Goblin))])`,
		`AsPermanentEnters(ThisPermanent,[SacrificeAnyNumberOfPermanents()])`,
		`AsPermanentEnters(ThisPermanent,[SacrificeAnyNumberOfPermanents(IsLandType(Forest))]),CDA_Toughness(TheNumberOfPermanentsSacrificedAsPermenantEnteredBattlefield(IsLandType(Forest),HostPermanent))`,
		`Activated(SacrificeAnyNumberOfPermanents(IsLandType(Forest)),ActionList([DrawACard(You)]))`,
		`SpellActions(ActionList([SacrificeAnyNumberOfPermanents(IsLandType(Forest))]))`,
	} {
		record := repositoryRecord(t, "LEG", "Wood Elemental", "LEG.lines.ron")
		nodes, err := Parse(strings.NewReader("["+rules+"]"), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		record.Fields["Rules"] = nodes[0]
		if _, err := Compile(record); err == nil {
			t.Fatal("accepted", rules)
		}
	}
}

func TestRONEntrySacrificeCopiesAndReturn(t *testing.T) {
	factory, err := Compile(repositoryRecord(t, "LEG", "Wood Elemental", "LEG.lines.ron"))
	if err != nil {
		t.Fatal(err)
	}
	a := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("A"), answer: 2}
	b := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("B"), answer: 1}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	for range 2 {
		g.PutOnBattlefield(mage.NewLand("A Forest", mage.WithSubTypes("Forest")), a.PlayerID())
	}
	g.PutOnBattlefield(mage.NewLand("B Forest", mage.WithSubTypes("Forest")), b.PlayerID())
	card := factory()
	castRONSpell(t, g, a.BasePlayer, card)
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("missing spell copy")
	}
	g.ResolveTopOfStack()
	checkConditionalPT(t, g, copied.SourceID, 1, 1)
	if !g.FindPermanent(copied.SourceID).IsToken || a.calls != 0 || b.calls != 1 {
		t.Fatal("copy identity or chooser")
	}
	g.ResolveStack()
	checkConditionalPT(t, g, card.ID(), 2, 2)
	clone := g.Clone()
	g.PhaseIn(g.PhaseOut(card.ID()))
	checkConditionalPT(t, g, card.ID(), 2, 2)
	if a.calls != 1 {
		t.Fatal("phasing made a new choice")
	}
	g.BouncePermanentToHand(g.FindPermanent(card.ID()))
	power, toughness := g.CardPowerToughness(a.Hand()[0])
	if power != 0 || toughness != 0 {
		t.Fatal("hand retained battlefield history")
	}
	a.RemoveFromHand(card.ID())
	g.PutOnBattlefield(mage.NewLand("New Forest", mage.WithSubTypes("Forest")), a.PlayerID())
	a.answer = 1
	returned := g.PutOnBattlefield(card, a.PlayerID())
	checkConditionalPT(t, g, returned.ID(), 1, 1)
	checkConditionalPT(t, clone, card.ID(), 2, 2)
	if a.calls != 2 {
		t.Fatal("return did not choose again")
	}
	g.BouncePermanentToHand(g.FindPermanent(copied.SourceID))
	if len(b.Hand()) != 0 {
		t.Fatal("spell copy did not cease")
	}
}

func TestRONEntrySacrificeLoadAndInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var source string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Wood Elemental"`) {
			source = line
		}
	}
	if source == "" {
		t.Fatal("missing source")
	}
	files := fstest.MapFS{"LEG.ron": {Data: []byte(source)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Wood Elemental")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Wood Elemental")
	if err != nil {
		t.Fatal(err)
	}
	if first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared declaration")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(source, "ThisPermanent", "HostPermanent", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("Wood Elemental") {
		t.Fatal("invalid collection was published")
	}
	delete(files, "bad.ron")
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 || len(report.Issues) != 0 {
		t.Fatal("inventory rejected supported record", err)
	}
}

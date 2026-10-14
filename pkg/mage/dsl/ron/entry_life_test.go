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

func TestRONEntryLifeNamelessRace(t *testing.T) {
	factory, err := Compile(repositoryRecord(t, "DRK", "Nameless Race", "DRK.lines.ron"))
	if err != nil {
		t.Fatal(err)
	}
	for _, count := range []int{0, 1, 3, 4} {
		a := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("A"), answer: count}
		b := mage.NewBasePlayer("B")
		g := mage.NewGame(a, b)
		g.SetStep(core.PrecombatMain)
		white := func(name string) mage.Card { return mage.NewCreature(name, "{W}", 1, 1) }
		g.PutOnBattlefield(white("Opponent white"), b.PlayerID())
		g.PutOnBattlefield(white("Your white"), a.PlayerID())
		g.PutOnBattlefield(mage.NewCreature("Opponent black", "{B}", 1, 1), b.PlayerID())
		token := g.PutOnBattlefield(white("Token"), b.PlayerID())
		token.IsToken = true
		phased := g.PutOnBattlefield(white("Phased"), b.PlayerID())
		g.PhaseOut(phased.ID())
		for range 2 {
			c := white("Graveyard white")
			c.SetOwner(b.PlayerID())
			b.AddToGraveyard(c)
		}
		c := white("Your graveyard")
		c.SetOwner(a.PlayerID())
		a.AddToGraveyard(c)
		card := factory()
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || a.calls != 0 || a.Life() != 20 {
			t.Fatal("unpaid cast chose or spent life")
		}
		a.RemoveFromHand(card.ID())
		castRONSpell(t, g, a.BasePlayer, card)
		if a.calls != 0 || a.Life() != 20 {
			t.Fatal("announcement paid entry life")
		}
		g.ResolveStack()
		want := count
		if count > 3 {
			want = 0
		}
		if a.Life() != 20-want || a.calls != 1 {
			t.Fatal("entry life bounds", a.Life(), a.calls)
		}
		if want == 0 {
			if g.FindPermanent(card.ID()) != nil {
				t.Fatal("zero toughness survived")
			}
			continue
		}
		p := g.FindPermanent(card.ID())
		if p == nil || !p.HasKeyword(core.Trample) {
			t.Fatal("entry or trample")
		}
		checkConditionalPT(t, g, card.ID(), want, want)
		g.RemoveFromBattlefield(g.FindPermanent(token.ID()))
		g.PhaseIn(g.PhaseOut(card.ID()))
		checkConditionalPT(t, g, card.ID(), want, want)
		if a.calls != 1 {
			t.Fatal("phase repeated payment")
		}
		if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetBasePowerToughness(mage.This, mage.Fixed(1), mage.Fixed(2))), card.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		g.ApplyContinuousEffects()
		checkConditionalPT(t, g, card.ID(), 1, 2)
	}
}

func TestRONEntryLifeInvalid(t *testing.T) {
	for _, rules := range []string{
		`CDA_Power(TheLifePaid),CDA_Toughness(TheLifePaid)`,
		`AsPermanentEnters(HostPermanent,[PayAnyAmountOfLifeUpto(Integer(3))])`,
		`AsPermanentEnters(ThisPermanent,[PayAnyAmountOfLifeUpto(Integer(3)),EntersTapped])`,
		`AsPermanentEnters(ThisPermanent,[PayAnyAmountOfLifeUpto(Integer(3))]),AsPermanentEnters(ThisPermanent,[EntersTapped])`,
		`AsPermanentEnters(ThisPermanent,[PayAnyAmountOfLifeUpto(ValueX)])`,
		`AsPermanentEnters(ThisPermanent,[PayAnyAmountOfLifeUpto(TheLifePaid)])`,
		`AsPermanentEnters(ThisPermanent,[PayAnyAmountOfLifeUpto(Integer(-1))])`,
		`AsPermanentEnters(ThisPermanent,[PayAnyAmountOfLifeUpto()])`,
		`AsPermanentEnters(ThisPermanent,[PayAnyAmountOfLifeUpto(Integer(3))]),CDA_Power(TheLifePaid)`,
		`AsPermanentEnters(ThisPermanent,[PayAnyAmountOfLifeUpto(Integer(3))]),CDA_Power(TheLifePaid()),CDA_Toughness(TheLifePaid())`,
		`AsPermanentEnters(ThisPermanent,[PayAnyAmountOfLifeUpto(Integer(3))]),PermanentLayerEffect(ThisPermanent,[SetPowerAndToughnessBoth(TheLifePaid)])`,
		`Activated(PayAnyAmountOfLifeUpto(Integer(3)),ActionList([GainLife(Integer(1))]))`,
	} {
		record := repositoryRecord(t, "DRK", "Nameless Race", "DRK.lines.ron")
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

func TestRONEntryLifeCopiesClonesAndReturn(t *testing.T) {
	factory, err := Compile(repositoryRecord(t, "DRK", "Nameless Race", "DRK.lines.ron"))
	if err != nil {
		t.Fatal(err)
	}
	a := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("A"), answer: 2}
	b := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("B"), answer: 1}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	for _, owner := range []*mage.BasePlayer{a.BasePlayer, b.BasePlayer} {
		for range 2 {
			c := mage.NewArtifact("White", "{W}")
			c.SetOwner(owner.PlayerID())
			owner.AddToGraveyard(c)
		}
	}
	card := factory()
	castRONSpell(t, g, a.BasePlayer, card)
	clone := g.Clone()
	copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("missing copy")
	}
	g.ResolveTopOfStack()
	checkConditionalPT(t, g, copied.SourceID, 1, 1)
	if !g.FindPermanent(copied.SourceID).IsToken || a.Life() != 20 || b.Life() != 19 {
		t.Fatal("copy payment/controller")
	}
	g.ResolveStack()
	checkConditionalPT(t, g, card.ID(), 2, 2)
	clone.ResolveStack()
	checkConditionalPT(t, clone, card.ID(), 2, 2)
	if clone.GetPlayer(a.PlayerID()).Life() != 18 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("pending clone shared payment")
	}
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: card.ID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	checkConditionalPT(t, g, card.ID(), 2, 2)
	g.BouncePermanentToHand(g.FindPermanent(card.ID()))
	if power, toughness := g.CardPowerToughness(a.Hand()[0]); power != 0 || toughness != 0 {
		t.Fatal("hand kept payment")
	}
	a.RemoveFromHand(card.ID())
	a.answer = 1
	p := g.PutOnBattlefield(card, a.PlayerID())
	checkConditionalPT(t, g, p.ID(), 1, 1)
	if a.Life() != 17 || b.Life() != 19 {
		t.Fatal("return did not pay again")
	}
	p = g.MutablePermanent(p.ID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	g.SetActivePlayerIndex(0)
	g.SetStep(core.DeclareAttackers)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID()})
	g.ExecuteCombatDamage()
	if b.Life() != 18 {
		t.Fatal("native combat did not use entry power")
	}
}

func TestRONEntryLifeLoadAndInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	var source string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Nameless Race"`) {
			source = line
		}
	}
	files := fstest.MapFS{"DRK.ron": {Data: []byte(source)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Nameless Race")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Nameless Race")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID() == second.ID() {
		t.Fatal("shared card identity")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 || len(report.Issues) != 0 {
		t.Fatal("inventory", err)
	}
	want := map[string]string{"PayAnyAmountOfLifeUpto": "entry_operation", "Plus": "value", "TheNumberOfGraveyardCards": "value", "TheLifePaid": "characteristic_count_value"}
	for _, signature := range report.Signatures {
		if role, ok := want[signature.Name]; ok && signature.Role == role {
			delete(want, signature.Name)
		}
	}
	if len(want) != 0 {
		t.Fatal("missing roles", want)
	}
	files["Bad.ron"] = &fstest.MapFile{Data: []byte(strings.ReplaceAll(source, "TheLifePaid", "TheLifePaid()"))}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("invalid collection accepted")
	}
	if _, err := registry.CreateCard("Nameless Race"); err != nil {
		t.Fatal("failed load changed registry")
	}
}

func TestRONEntryLifeMaximumDeclarations(t *testing.T) {
	for _, maximum := range []string{
		`Integer`, `Integer(You)`, `Plus(Integer(1))`, `Plus(Integer(1),TheLifePaid)`,
		`TheNumberOfPermanentsOnTheBattlefield(And([]))`,
		`TheNumberOfPermanentsOnTheBattlefield(SinglePermanent(Ref_TargetPermanent))`,
		`TheNumberOfPermanentsOnTheBattlefield(ControlledByAPlayer(SinglePlayer(Trigger_ThatPlayer)))`,
		`TheNumberOfGraveyardCards(IsColor(White))`,
		`TheNumberOfGraveyardCards(And([]))`,
		`TheNumberOfGraveyardCards(And([InAPlayersGraveyard(Opponent),InAPlayersGraveyard(Opponent)]))`,
		`TheNumberOfGraveyardCards(And([IsColor(White),InAPlayersGraveyard(SinglePlayer(Ref_TargetPlayer))]))`,
		`TheNumberOfGraveyardCards(And([IsColor(White),InAPlayersGraveyard(You)]))`,
		`TheNumberOfGraveyardCards(And([IsColor(White),InAPlayersGraveyard(Opponent())]))`,
	} {
		record := repositoryRecord(t, "DRK", "Nameless Race", "DRK.lines.ron")
		rules, err := Parse(strings.NewReader(`[AsPermanentEnters(ThisPermanent,[PayAnyAmountOfLifeUpto(`+maximum+`)]),CDA_Power(TheLifePaid),CDA_Toughness(TheLifePaid)]`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		record.Fields["Rules"] = rules[0]
		if _, err := Compile(record); err == nil {
			t.Fatal("accepted cap", maximum)
		}
	}
	for _, maximum := range []string{`Integer(2)`, `Plus(Integer(1),Integer(1))`, `TheNumberOfGraveyardCards(And([InAPlayersGraveyard(SinglePlayer(You)),IsColor(White)]))`} {
		record := repositoryRecord(t, "DRK", "Nameless Race", "DRK.lines.ron")
		rules, err := Parse(strings.NewReader(`[AsPermanentEnters(ThisPermanent,[PayAnyAmountOfLifeUpto(`+maximum+`)]),CDA_Power(TheLifePaid),CDA_Toughness(TheLifePaid)]`), "valid.ron")
		if err != nil {
			t.Fatal(err)
		}
		record.Fields["Rules"] = rules[0]
		factory, err := Compile(record)
		if err != nil {
			t.Fatal(err)
		}
		a := &ronNumericChooser{BasePlayer: mage.NewBasePlayer("A"), answer: 2}
		g := mage.NewGame(a, mage.NewBasePlayer("B"))
		for range 2 {
			c := mage.NewInstant("White", "{W}")
			c.SetOwner(a.PlayerID())
			a.AddToGraveyard(c)
		}
		p := g.PutOnBattlefield(factory(), a.PlayerID())
		checkConditionalPT(t, g, p.ID(), 2, 2)
		if a.Life() != 18 || a.calls != 1 {
			t.Fatal("fixed or explicit player cap payment")
		}
	}
}

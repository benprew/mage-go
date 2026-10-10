package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTextConditionalPTSpellRecords(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		old       core.BasicLandType
		dp, dt    int
		opponent  bool
	}{
		{"ARN", "Kird Ape", core.ForestLandType, 1, 2, false},
		{"3ED", "Kird Ape", core.ForestLandType, 1, 2, false},
		{"DRK", "Water Wurm", core.IslandLandType, 0, 1, true},
	} {
		t.Run(tc.set+"/"+tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Red, 10)
			a.ManaPool().Add(core.Blue, 10)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			paid := a.ManaPool().TotalMana()
			change, _ := mage.ReplaceBasicLandWord(tc.old, core.PlainsLandType)
			if err := g.RevisePrintedSpellText(g.StackPeek().ID, change); err != nil {
				t.Fatal(err)
			}
			if g.FindPermanent(card.ID()) != nil || g.StackPeek().Card != card || a.ManaPool().TotalMana() != paid {
				t.Fatal("revision changed payment or timing")
			}
			copied := g.CopySpellOnStack(card.ID(), b.PlayerID(), false)
			second, _ := mage.ReplaceBasicLandWord(core.PlainsLandType, core.SwampLandType)
			if err := g.RevisePrintedSpellText(copied.ID, second); err != nil {
				t.Fatal(err)
			}
			clone := g.Clone()
			third, _ := mage.ReplaceBasicLandWord(core.SwampLandType, core.MountainLandType)
			if err := clone.RevisePrintedSpellText(g.StackPeek().ID, third); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			clone.ResolveStack()
			source := g.FindPermanent(card.ID())
			token := g.FindPermanent(copied.SourceID)
			if source == nil || source.Card.ID() != card.ID() || source.Card.Owner() != a.PlayerID() || token == nil || !token.Card.IsToken() {
				t.Fatal("entry lost identity")
			}
			staticPTCheck(t, g, source, 1, 1)
			payer, other := a, b
			if tc.opponent {
				payer, other = b, a
			}
			old := g.PutOnBattlefield(repositoryCard(t, "LEA", tc.old.String()), payer.PlayerID())
			staticPTCheck(t, g, source, 1, 1)
			wrong := g.PutOnBattlefield(repositoryCard(t, "LEA", "Plains"), other.PlayerID())
			staticPTCheck(t, g, source, 1, 1)
			land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Plains"), payer.PlayerID())
			staticPTCheck(t, g, source, 1+tc.dp, 1+tc.dt)
			staticPTCheck(t, g, token, 1, 1)
			swamp := g.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), other.PlayerID())
			staticPTCheck(t, g, token, 1+tc.dp, 1+tc.dt)
			clone.PutOnBattlefield(repositoryCard(t, "LEA", "Swamp"), other.PlayerID())
			staticPTCheck(t, clone, token, 1, 1)
			staticPTCheck(t, clone, source, 1, 1)
			g.RemoveFromBattlefield(g.FindPermanent(old.ID()))
			g.RemoveFromBattlefield(g.FindPermanent(wrong.ID()))
			g.RemoveFromBattlefield(g.FindPermanent(swamp.ID()))
			id := textStaticID(t, source)
			if err := g.AddPrintedStaticPTTextEffect(source, id, core.EndOfTurn, second); err != nil {
				t.Fatal(err)
			}
			staticPTCheck(t, g, source, 1, 1)
			phase := g.PhaseOut(card.ID())
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			g.PhaseIn(phase)
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, source, 1+tc.dp, 1+tc.dt)
			if textStaticID(t, g.FindPermanent(card.ID())) != id {
				t.Fatal("phasing changed declaration")
			}
			phase = g.PhaseOut(land.ID())
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, source, 1, 1)
			g.PhaseIn(phase)
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, source, 1+tc.dp, 1+tc.dt)
			g.AddContinuousEffect(mage.RemoveAllAbilities(card.ID(), core.EndOfTurn))
			staticPTCheck(t, g, source, 1, 1)
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, source, 1+tc.dp, 1+tc.dt)
			g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
			staticPTCheck(t, g, source, 1, 1)
			g.GetEffects().RemoveEndOfTurn()
			g.ApplyContinuousEffects()
			staticPTCheck(t, g, source, 1+tc.dp, 1+tc.dt)
			values, err := g.CaptureCopiableValues(g.FindPermanent(card.ID()))
			if err != nil {
				t.Fatal(err)
			}
			printed, err := values.Build()
			if err != nil {
				t.Fatal(err)
			}
			sibling := g.PutOnBattlefield(printed, a.PlayerID())
			staticPTCheck(t, g, sibling, 1, 1)
			g.BouncePermanentToHand(g.FindPermanent(card.ID()))
			lki := g.LKI(card.ID())
			if lki == nil {
				t.Fatal("missing LKI")
			}
			saved := g.PutOnBattlefield(lki.Snapshot.Card.Copy(), a.PlayerID())
			staticPTCheck(t, g, saved, 1+tc.dp, 1+tc.dt)
			if err := g.AddPrintedStaticPTTextEffect(source, id, core.Indefinite, second); err == nil {
				t.Fatal("departed visit accepted")
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			staticPTCheck(t, g, g.FindPermanent(card.ID()), 1, 1)
		})
	}
}

func TestRONTextConditionalPTSpellBoundaries(t *testing.T) {
	for _, tc := range []struct{ set, name string }{
		{"DRK", "Goblin Caves"}, {"DRK", "Goblin Shrine"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			for _, color := range []core.Color{core.White, core.Blue, core.Black, core.Red, core.Green} {
				a.ManaPool().Add(color, 10)
			}
			var targets []uuid.UUID
			if card.HasSubType("Aura") {
				land := g.PutOnBattlefield(repositoryCard(t, "LEA", "Mountain"), a.PlayerID())
				targets = []uuid.UUID{land.ID()}
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
				t.Fatal(err)
			}
			before := g.StackPeek()
			change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.PlainsLandType)
			if err := g.RevisePrintedSpellText(before.ID, change); err == nil {
				t.Fatal("mixed context accepted")
			}
			if g.StackPeek() != before {
				t.Fatal("rejection changed stack")
			}
		})
	}
	for _, rules := range []string{
		`If(PlayerPassesFilter(You, ControlsA(IsLandType(Forest))), [])`,
		`If(PlayerPassesFilter(You, ControlsA(IsLandType(Forest))), [PermanentLayerEffect(Ref_TargetPermanent, [AdjustPT(+1,+2)])])`,
		`If(APlayerPassesFilter(Opponent, ControlsA(IsLandType(Forest))), [PermanentLayerEffect(ThisPermanent, [AdjustPT(+1)])])`,
	} {
		if _, err := Compile(nameScopeNode(t, `Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: []), ManaCost: [], Rules: [`+rules+`], CardPT: (Power: 1, Toughness: 1))`)); err == nil {
			t.Fatal("invalid declaration accepted", rules)
		}
	}
}

func TestRONTextConditionalPTSpellMixedBoundary(t *testing.T) {
	node := nameScopeNode(t, `Card(Name: "Mixed", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: []), ManaCost: [], Rules: [If(PlayerPassesFilter(You, ControlsA(IsLandType(Forest))), [PermanentLayerEffect(ThisPermanent, [AdjustPT(+1,+2)])]), TriggerA(AtTheBeginningOfAPlayersUpkeep(SinglePlayer(You)), ActionList([MayCost(PayMana([ManaCostG])), Unless(CostWasPaid, [SacrificePermanent(ThisPermanent)])]))], CardPT: (Power: 1, Toughness: 1))`)
	factory, err := Compile(node)
	if err != nil {
		t.Fatal(err)
	}
	g, a, _ := ronGame()
	card := factory()
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	before := g.StackPeek()
	change, _ := mage.ReplaceBasicLandWord(core.ForestLandType, core.IslandLandType)
	if err := g.RevisePrintedSpellText(before.ID, change); err == nil {
		t.Fatal("conditional mixed continuation accepted")
	}
	if g.StackPeek() != before {
		t.Fatal("rejection changed stack")
	}
}

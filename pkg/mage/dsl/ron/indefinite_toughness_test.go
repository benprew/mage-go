package ron

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func toughnessCheck(t *testing.T, g *mage.Game, id uuid.UUID, power, toughness int) {
	t.Helper()
	g.ApplyContinuousEffects()
	p := g.FindPermanent(id)
	if p == nil || p.CurrentPower(g) != power || p.CurrentToughness(g) != toughness {
		t.Fatalf("wrong P/T: want %d/%d, permanent %v", power, toughness, p)
	}
}

func TestRONIndefiniteToughnessSentinel(t *testing.T) {
	for _, direction := range []string{"attacking", "blocking"} {
		for _, change := range []string{"none", "target-power", "combat-exit", "source-return", "source-phase", "target-return", "target-phase", "copy"} {
			t.Run(direction+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "LEG", "Sentinel")
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				target := g.PutOnBattlefield(mage.NewCreature("Opponent", "", 4, 9), b.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				target.RevokeBaseAttr(core.AttrSummonSick)
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err == nil || g.StackSize() != 0 {
					t.Fatal("noncombat target accepted")
				}
				attacker, blocker := source.ID(), target.ID()
				active := a.PlayerID()
				if direction == "blocking" {
					attacker, blocker = blocker, attacker
					active = b.PlayerID()
				}
				g.ExecuteAttackers(active, []uuid.UUID{attacker})
				g.ExecuteBlockers([]mage.BlockAssignment{{AttackerID: attacker, BlockerID: blocker}})
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
					t.Fatal(err)
				}
				clone := g.Clone()
				switch change {
				case "target-power":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(2), mage.Fixed(0))), target.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "combat-exit":
					g.GetCombat().RemoveFromCombat(target.ID())
				case "source-return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					g.PutOnBattlefield(card, a.PlayerID())
				case "source-phase":
					g.PhaseOut(source.ID())
				case "target-return":
					c := target.Card
					g.RemoveFromBattlefield(g.FindPermanent(target.ID()))
					g.PutOnBattlefield(c, b.PlayerID())
				case "target-phase":
					g.PhaseOut(target.ID())
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				g.ResolveStack()
				g.PhaseIn([]uuid.UUID{source.ID(), target.ID()})
				want := 5
				if change == "target-power" {
					want = 7
				}
				if change == "combat-exit" || change == "source-return" || change == "source-phase" || change == "target-return" || change == "target-phase" {
					want = 1
				}
				toughnessCheck(t, g, source.ID(), 1, want)
				clone.ResolveStack()
				toughnessCheck(t, clone, source.ID(), 1, 5)
				if want != 1 {
					g.GetCombat().RemoveFromCombat(source.ID())
					g.PhaseIn(g.PhaseOut(source.ID()))
					g.FireEvent(core.GameEvent{Type: core.EvtCleanup})
					toughnessCheck(t, g, source.ID(), 1, want)
					g.MutablePermanent(source.ID()).AddCounter(core.P1P1, 1)
					toughnessCheck(t, g, source.ID(), 2, want+1)
				}
			})
		}
	}
}

func TestRONIndefiniteToughnessWall(t *testing.T) {
	for _, change := range []string{"none", "count-before", "source-control", "source-return", "phase-before", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Wall of Tombstones")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			if !source.HasAttr(core.Defender) {
				t.Fatal("missing defender")
			}
			a.AddToGraveyard(mage.NewCreature("One", "", 1, 1))
			a.AddToGraveyard(mage.NewArtifact("Ignored", ""))
			b.AddToGraveyard(mage.NewCreature("Opponent", "", 1, 1))
			queueRONUpkeep(g, b)
			if g.StackSize() != 0 {
				t.Fatal("opponent upkeep triggered")
			}
			queueRONUpkeep(g, a)
			if g.StackSize() != 1 {
				t.Fatal("missing upkeep trigger")
			}
			clone := g.Clone()
			want := 2
			switch change {
			case "count-before":
				a.AddToGraveyard(mage.NewCreature("Before", "", 1, 1))
				want = 3
			case "source-control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.PutOnBattlefield(card, a.PlayerID())
				want = 1
			case "phase-before":
				g.PhaseOut(source.ID())
				want = 1
			case "copy":
				b.AddToGraveyard(mage.NewCreature("Copy controller", "", 1, 1))
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				g.ResolveTopOfStack()
				toughnessCheck(t, g, source.ID(), 0, 3)
			}
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{source.ID()})
			toughnessCheck(t, g, source.ID(), 0, want)
			clone.ResolveStack()
			toughnessCheck(t, clone, source.ID(), 0, 2)
			a.AddToGraveyard(mage.NewCreature("After", "", 1, 1))
			g.FireEvent(core.GameEvent{Type: core.EvtCleanup})
			toughnessCheck(t, g, source.ID(), 0, want)
			if change == "none" {
				queueRONUpkeep(g, a)
				g.ResolveStack()
				toughnessCheck(t, g, source.ID(), 0, 3)
				g.PhaseIn(g.PhaseOut(source.ID()))
				toughnessCheck(t, g, source.ID(), 0, 3)
			}
		})
	}
}

func TestRONIndefiniteToughnessInvalid(t *testing.T) {
	for _, value := range []string{`SetToughness`, `SetToughness()`, `SetToughness(1)`, `SetToughness(Integer(1),Integer(2))`, `SetToughness(ValueX)`, `SetToughness(TheChosenNumber)`, `SetToughness(PowerOfPermanent(Ref_TargetPermanent))`, `SetToughness(Plus(Integer(1)))`, `SetToughness(Plus(Integer(1),Integer(2),Integer(3)))`, `SetToughness(TheNumberOfGraveyardCards(And([])))`, `SetToughness(TheNumberOfGraveyardCards(And([IsCardtype(Creature),InAPlayersGraveyard(SinglePlayer(You)),InAPlayersGraveyard(SinglePlayer(You))])))`, `SetToughness(TheNumberOfGraveyardCards(And([IsCardtype(Creature),InAPlayersGraveyard(SinglePlayer(Ref_TargetPlayer))])))`} {
		text := fmt.Sprintf(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffect(ThisPermanent,[%s])]))])`, value)
		nodes, err := Parse(strings.NewReader(text), "invalid-toughness.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", value)
		}
	}
}

func TestRONIndefiniteToughnessStrictLoad(t *testing.T) {
	data := ""
	for _, name := range []string{"Sentinel", "Wall of Tombstones"} {
		source, err := os.ReadFile("../../../../cards/mtgish/LEG.lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		n := hostUntapRecord(t, "LEG", name)
		data += strings.Split(string(source), "\n")[n.Position.Line-1] + "\n"
	}
	registry, err := LoadFS(fstest.MapFS{"a.ron": {Data: []byte(data)}, "b.ron": {Data: []byte(data)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("wrong canonical names")
	}
	for _, name := range registry.RegisteredCardNames() {
		one, _ := registry.CreateCard(name)
		two, _ := registry.CreateCard(name)
		if err := mage.ValidateCardDefinition(name, one); err != nil {
			t.Fatal(err)
		}
		if one.Abilities()[0] == two.Abilities()[0] {
			t.Fatal("shared abilities")
		}
	}
}

func TestRONIndefiniteToughnessCastingAndInstances(t *testing.T) {
	for _, name := range []string{"Sentinel", "Wall of Tombstones"} {
		t.Run(name, func(t *testing.T) {
			g, a, _ := ronGame()
			card := repositoryCard(t, "LEG", name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("unpaid casting did not roll back")
			}
			if name == "Sentinel" {
				a.ManaPool().Add(core.Colorless, 4)
			} else {
				a.ManaPool().Add(core.Colorless, 1)
				a.ManaPool().Add(core.Black, 1)
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()) == nil || a.ManaPool().TotalMana() != 0 {
				t.Fatal("wrong casting payment or entry")
			}
			two := repositoryCard(t, "LEG", name)
			if card.Abilities()[0] == two.Abilities()[0] {
				t.Fatal("shared ability")
			}
			if name == "Sentinel" && card.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == two.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("shared target declaration")
			}
			if name == "Wall of Tombstones" {
				g.PutOnBattlefield(two, a.PlayerID())
				a.AddToGraveyard(mage.NewCreature("One", "", 1, 1))
				queueRONUpkeep(g, a)
				if g.StackSize() != 2 {
					t.Fatal("instances lost triggers")
				}
				g.ResolveStack()
				toughnessCheck(t, g, card.ID(), 0, 2)
				toughnessCheck(t, g, two.ID(), 0, 2)
				g.MutablePermanent(card.ID()).RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{card.ID()})
				if g.GetCombat().IsAttacking(card.ID()) {
					t.Fatal("defender attacked")
				}
			}
		})
	}
}

func TestRONIndefiniteToughnessLKIAndContexts(t *testing.T) {
	for _, change := range []string{"sacrificed", "returned", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			factory := permanentScopeFactory(t, "Artifact", `Activated(SacrificePermanent(ThisPermanent),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentLayerEffect(Ref_TargetPermanent,[SetToughness(Plus(Integer(1),PowerOfPermanent(ThisPermanent)))])])))`)
			card := factory()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			target := g.PutOnBattlefield(mage.NewCreature("Target", "", 3, 8), b.PlayerID())
			if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mage.SetBasePower(mage.This, mage.Fixed(4))), source.ID(), a.PlayerID(), nil); err != nil {
				t.Fatal(err)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if change == "returned" {
				a.RemoveFromGraveyard(card.ID())
				g.PutOnBattlefield(card, a.PlayerID())
			}
			if change == "clone" {
				g = g.Clone()
			}
			g.ResolveStack()
			toughnessCheck(t, g, target.ID(), 3, 5)
		})
	}
	for _, rule := range []string{
		`PermanentLayerEffect(ThisPermanent,[SetToughness(Integer(2))])`,
		`Activated(PayMana([]),ActionList([CreatePermanentLayerEffectUntil(ThisPermanent,[SetToughness(Integer(2))],UntilEndOfTurn)]))`,
		`Activated(SetToughness(Integer(2)),ActionList([TapPermanent(ThisPermanent)]))`,
		`Activated(PayMana([]),ActionList([SetToughness(Integer(2))]))`,
		`Activated(PayMana([]),ActionList([CreatePermanentLayerEffect(ThisPermanent,[SetToughness(Integer(2)),AddCardtype(Artifact)])]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rule+`])`), "invalid-context.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONIndefiniteToughnessValueShapes(t *testing.T) {
	for _, value := range []string{
		`Plus(Integer(4),Minus(Integer(3),Integer(5)))`,
		`TheNumberOfGraveyardCards(And([InAPlayersGraveyard(SinglePlayer(You)),And([IsCardtype(Creature),IsNonColor(Blue)])]))`,
	} {
		g, a, b := ronGame()
		a.AddToGraveyard(mage.NewCreature("One", "", 1, 1))
		a.AddToGraveyard(mage.NewCreature("Two", "", 1, 1))
		a.AddToGraveyard(mage.NewCreature("Excluded", "{U}", 1, 1))
		factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentLayerEffect(Ref_TargetPermanent,[SetToughness(`+value+`)])])))`)
		source := g.PutOnBattlefield(factory(), a.PlayerID())
		target := g.PutOnBattlefield(mage.NewCreature("Target", "", 3, 8), b.PlayerID())
		if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
			t.Fatal(err)
		}
		g.ResolveStack()
		toughnessCheck(t, g, target.ID(), 3, 2)
	}
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(`Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[Activated(PayMana([]),ActionList([CreatePermanentLayerEffect(ThisPermanent,[SetToughness(Plus(Integer(1),TheNumberOfGraveyardCards(And([IsCardtype(Creature),InAPlayersGraveyard(SinglePlayer(You))]))))])]))])`)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v", err)
	}
	wanted := map[string]string{"SetToughness": "resolving_modification", "Plus": "value", "TheNumberOfGraveyardCards": "value", "InAPlayersGraveyard": "card_filter", "You": "player_reference"}
	for _, entry := range report.Signatures {
		if role, ok := wanted[entry.Name]; ok && entry.Role == role {
			delete(wanted, entry.Name)
		}
	}
	if len(wanted) != 0 {
		t.Fatalf("missing roles: %v", wanted)
	}
}

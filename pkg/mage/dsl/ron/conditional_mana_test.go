package ron

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONConditionalManaCards(t *testing.T) {
	for _, name := range []string{"Urza's Mine", "Urza's Power Plant", "Urza's Tower"} {
		for mask := range 4 {
			t.Run(fmt.Sprintf("%s/%d", name, mask), func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, "ATQ", name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.PlayLand(a.PlayerID(), card.ID()); err != nil {
					t.Fatal(err)
				}
				source := g.FindPermanent(card.ID())
				others := []string{}
				for _, other := range []string{"Urza's Mine", "Urza's Power Plant", "Urza's Tower"} {
					if other != name {
						others = append(others, other)
					}
				}
				for i, other := range others {
					controller := b.PlayerID()
					if mask&(1<<i) != 0 {
						controller = a.PlayerID()
					}
					c := repositoryCard(t, "ATQ", other)
					c.SetOwner(b.PlayerID())
					g.PutOnBattlefield(c, controller)
				}
				want := 1
				if mask == 3 {
					want = 2
					if name == "Urza's Tower" {
						want = 3
					}
				}
				for range 3 {
					got := mage.ManaProductionsForAbilityInGame(card.Abilities()[0], g, source.ID())
					total := 0
					for _, production := range got {
						if production.Color != core.Colorless {
							t.Fatal("wrong mana color")
						}
						total += production.Amount
					}
					if total != want {
						t.Fatalf("production = %+v, want %d", got, want)
					}
				}
				if source.Tapped || a.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
					t.Fatal("query mutated game")
				}
				clone := g.Clone()
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
					t.Fatal(err)
				}
				if !g.FindPermanent(source.ID()).Tapped || a.ManaPool().Count(core.Colorless) != want || b.ManaPool().TotalMana() != 0 || g.StackSize() != 0 {
					t.Fatal("mana activation")
				}
				if clone.FindPermanent(source.ID()).Tapped || clone.GetPlayer(a.PlayerID()).ManaPool().TotalMana() != 0 {
					t.Fatal("clone changed")
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err == nil || a.ManaPool().Count(core.Colorless) != want {
					t.Fatal("tapped cost accepted")
				}
			})
		}
	}
}

func TestRONConditionalManaInvalid(t *testing.T) {
	valid := `PlayerPassesFilter(You,And([ControlsA(And([IsLandType(Urzas),IsLandType(Tower)]))]))`
	for _, rule := range []string{
		`Activated(TapPermanent(ThisPermanent),ActionList([IfElse(` + valid + `,[AddMana(ManaProduceC)],[])]))`,
		`Activated(TapPermanent(ThisPermanent),ActionList([IfElse(` + valid + `,[AddMana(ManaProduceC),DrawACard],[AddMana(ManaProduceC)])]))`,
		`Activated(TapPermanent(ThisPermanent),ActionList([IfElse(PlayerPassesFilter(Ref_TargetPlayer,ControlsA(IsLandType(Tower))),[AddMana(ManaProduceC)],[AddMana(ManaProduceC)])]))`,
		`Activated(TapPermanent(ThisPermanent),ActionList([IfElse(PlayerPassesFilter(You,And([])),[AddMana(ManaProduceC)],[AddMana(ManaProduceC)])]))`,
		`Activated(TapPermanent(ThisPermanent),ActionList([IfElse(PlayerPassesFilter(You,ControlsA(IsLandType(Aura))),[AddMana(ManaProduceC)],[AddMana(ManaProduceC)])]))`,
		`Activated(TapPermanent(HostPermanent),ActionList([IfElse(` + valid + `,[AddMana(ManaProduceC)],[AddMana(ManaProduceC)])]))`,
		`Activated(And([PayMana([ManaCostGeneric(1)]),TapPermanent(ThisPermanent)]),ActionList([IfElse(` + valid + `,[AddMana(ManaProduceC)],[AddMana(ManaProduceC)])]))`,
		`Activated(TapPermanent(ThisPermanent),ActionList([IfElse(` + valid + `,[AddMana(ManaProduceC)],[AddMana(ManaProduceC)]),DrawACard]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Land],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONConditionalManaCurrentState(t *testing.T) {
	for _, change := range []string{"phasing", "departure", "subtype", "control", "source control", "same name", "missing Urzas"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			sourceCard := repositoryCard(t, "ATQ", "Urza's Tower")
			source := g.PutOnBattlefield(sourceCard, a.PlayerID())
			mineCard := repositoryCard(t, "ATQ", "Urza's Mine")
			mine := g.PutOnBattlefield(mineCard, a.PlayerID())
			plant := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Urza's Power Plant"), a.PlayerID())
			clone := g.Clone()
			switch change {
			case "phasing":
				g.PhaseOut(mine.ID())
			case "departure":
				g.RemoveFromBattlefield(mine)
			case "subtype":
				g.AddContinuousEffect(mage.SetSubtypes(mine.ID(), mage.SubtypeLand, core.Indefinite, "Forest"))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{TargetID: mine.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
			case "source control":
				g.AddControlEffect(mage.ControlEffectSpec{TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.EndOfTurn})
			case "same name":
				g.RemoveFromBattlefield(mine)
				g.PutOnBattlefield(mage.NewLand("Urza's Mine"), a.PlayerID())
			case "missing Urzas":
				g.AddContinuousEffect(mage.SetSubtypes(plant.ID(), mage.SubtypeLand, core.Indefinite, "Power-Plant"))
			}
			total := func(game *mage.Game) int {
				amount := 0
				for _, p := range mage.ManaProductionsForAbilityInGame(sourceCard.Abilities()[0], game, source.ID()) {
					amount += p.Amount
				}
				return amount
			}
			if total(g) != 1 || total(clone) != 3 {
				t.Fatal("current state or clone production")
			}
			if change == "phasing" {
				g.PhaseIn([]uuid.UUID{mine.ID()})
				if total(g) != 3 {
					t.Fatal("phase return lost subtype condition")
				}
			}
			if change == "departure" {
				g.PutOnBattlefield(mineCard, a.PlayerID())
				if total(g) != 3 {
					t.Fatal("new incarnation excluded from current collection")
				}
			}
		})
	}
}

func TestRONConditionalManaPayment(t *testing.T) {
	for _, complete := range []bool{false, true} {
		g, a, _ := ronGame()
		tower := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Urza's Tower"), a.PlayerID())
		var lands []*mage.Permanent
		if complete {
			for _, name := range []string{"Urza's Mine", "Urza's Power Plant"} {
				lands = append(lands, g.PutOnBattlefield(repositoryCard(t, "ATQ", name), a.PlayerID()))
			}
		}
		card := repositoryCard(t, "ATQ", "Tawnos's Wand")
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0)
		if !complete {
			if err == nil || g.FindPermanent(tower.ID()).Tapped || g.StackSize() != 0 || len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 0 {
				t.Fatal("failed payment changed live state")
			}
		} else {
			if err != nil {
				t.Fatal(err)
			}
			if g.StackSize() != 1 || len(a.Hand()) != 0 {
				t.Fatal("solver did not use conditional production")
			}
			tapped := 0
			produced := 0
			for _, p := range append(lands, tower) {
				if g.FindPermanent(p.ID()).Tapped {
					tapped++
					for _, production := range mage.ManaProductionsForAbilityInGame(p.Card.Abilities()[0], g, p.ID()) {
						produced += production.Amount
					}
				}
			}
			if tapped != 2 || a.ManaPool().Count(core.Colorless) != produced-4 {
				t.Fatal("payment plan and production disagree")
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()) == nil {
				t.Fatal("paid spell failed to resolve")
			}
		}
	}
}

func TestRONConditionalManaInventory(t *testing.T) {
	source := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Land],Subtypes:[]),Rules:[Activated(TapPermanent(ThisPermanent),ActionList([IfElse(PlayerPassesFilter(You,And([ControlsA(And([IsLandType(Urzas),IsLandType(Tower)]))])),[AddMana(ManaProduceC)],[AddMana(ManaProduceC)])]))])`
	report, err := InspectFS(fstest.MapFS{"mana.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"IfElse": "action", "PlayerPassesFilter": "condition", "ControlsA": "static_player_filter", "IsLandType": "permanent_filter", "ManaProduceC": "mana_production"} {
		found := false
		for _, entry := range report.Signatures {
			if entry.Name == name {
				found = true
				if entry.Role != role {
					t.Errorf("%s role = %s, want %s", name, entry.Role, role)
				}
			}
		}
		if !found {
			t.Errorf("missing %s", name)
		}
	}
}

func TestRONConditionalManaShapes(t *testing.T) {
	for _, condition := range []string{`ControlsA(IsLandType(Tower))`, `And([ControlsA(IsLandType(Tower)),And([ControlsA(IsCardtype(Land))])])`} {
		factory := permanentScopeFactory(t, "Land", `Activated(TapPermanent(ThisPermanent),ActionList([IfElse(PlayerPassesFilter(You,`+condition+`),[AddMana(And([ManaProduceR,ManaProduceG]))],[AddMana(ManaProduceC)])]))`)
		g, a, _ := ronGame()
		source := g.PutOnBattlefield(factory(), a.PlayerID())
		other := g.PutOnBattlefield(mage.NewLand("Other name", mage.WithSubTypes("Tower")), a.PlayerID())
		if err := g.TapForMana(a.PlayerID(), source.ID()); err != nil {
			t.Fatal(err)
		}
		if a.ManaPool().Count(core.Red) != 1 || a.ManaPool().Count(core.Green) != 1 || g.StackSize() != 0 {
			t.Fatal("mixed fixed mana branch")
		}
		g.PhaseOut(other.ID())
		g.UntapPermanent(g.FindPermanent(source.ID()))
		if err := g.TapForMana(a.PlayerID(), source.ID()); err != nil {
			t.Fatal(err)
		}
		if a.ManaPool().Count(core.Colorless) != 1 {
			t.Fatal("false branch")
		}
	}
}

func TestRONConditionalManaMalformed(t *testing.T) {
	for _, condition := range []string{`PlayerPassesFilter(You(),ControlsA(IsLandType(Tower)))`, `PlayerPassesFilter(You,Or([ControlsA(IsLandType(Tower))]))`, `PlayerPassesFilter(You,ControlsNo(IsLandType(Tower)))`, `PlayerPassesFilter(You,ControlsA(And([])))`, `PlayerPassesFilter(You,ControlsA(IsLandType(Tower())))`, `PlayerPassesFilter(You,ControlsA(PowerIs(EqualTo(ValueX))))`, `CostWasPaid`, `PlayerPassesFilter(You,ControlsA(IsUntapped))`, `PlayerPassesFilter(You,ControlsA(IsTapped))`, `PlayerPassesFilter(You,ControlsA(IsAttacking))`} {
		assertConditionalRejected(t, `Activated(TapPermanent(ThisPermanent),ActionList([IfElse(`+condition+`,[AddMana(ManaProduceC)],[AddMana(ManaProduceC)])]))`)
	}
	for _, branch := range []string{`[]`, `[AddMana(And([]))]`, `[AddMana(AnyManaColor)]`, `[AddManaRepeated(ManaProduceC,ValueX)]`, `[AddMana(ManaProduceC())]`, `[AddMana(ManaProduceC,ManaProduceG)]`, `[GainLife(Integer(1))]`} {
		assertConditionalRejected(t, `Activated(TapPermanent(ThisPermanent),ActionList([IfElse(PlayerPassesFilter(You,ControlsA(IsLandType(Tower))),`+branch+`,[AddMana(ManaProduceC)])]))`)
	}
}

func TestRONConditionalManaBonusesAndRemoval(t *testing.T) {
	for _, name := range []string{"Urza's Mine", "Urza's Power Plant", "Urza's Tower"} {
		g, a, b := ronGame()
		var source *mage.Permanent
		for _, other := range []string{"Urza's Mine", "Urza's Power Plant", "Urza's Tower"} {
			p := g.PutOnBattlefield(repositoryCard(t, "ATQ", other), a.PlayerID())
			if other == name {
				source = p
			}
		}
		g.PutOnBattlefield(repositoryCard(t, "LEB", "Mana Flare"), b.PlayerID())
		if err := g.TapForMana(a.PlayerID(), source.ID()); err != nil {
			t.Fatal(err)
		}
		want := 3
		if name == "Urza's Tower" {
			want = 4
		}
		if a.ManaPool().Count(core.Colorless) != want || g.StackSize() != 0 {
			t.Fatal("mana bonus must apply once per tap")
		}
		g.UntapPermanent(g.FindPermanent(source.ID()))
		g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
		if err := g.TapForMana(a.PlayerID(), source.ID()); err == nil || g.FindPermanent(source.ID()).Tapped || a.ManaPool().Count(core.Colorless) != want {
			t.Fatal("removed ability used as mana source")
		}
	}
}

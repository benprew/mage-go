package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONNumericTargetsCards(t *testing.T) {
	for _, name := range []string{"Pendelhaven", "Stone Giant"} {
		t.Run(name, func(t *testing.T) {
			set := "LEG"
			index := 1
			power, toughness := 1, 1
			if name == "Stone Giant" {
				set = "2ED"
				index = 0
				power, toughness = 2, 2
			}
			g, a, b := ronGame()
			card := repositoryCard(t, set, name)
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", power, toughness), b.PlayerID())
			if name == "Stone Giant" {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, []uuid.UUID{victim.ID()}); err == nil {
					t.Fatal("opponent target accepted")
				}
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: victim.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			if victim.CurrentPower(g) != power || victim.HasKeyword(core.Flying) {
				t.Fatal("effect ran before resolution")
			}
			clone := g.Clone()
			g.ResolveStack()
			if name == "Pendelhaven" {
				if g.FindPermanent(victim.ID()).CurrentPower(g) != 2 || g.FindPermanent(victim.ID()).CurrentToughness(g) != 3 {
					t.Fatal("wrong bonus")
				}
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				if g.FindPermanent(victim.ID()).CurrentPower(g) != 1 {
					t.Fatal("bonus did not expire")
				}
			} else {
				if !g.FindPermanent(victim.ID()).HasKeyword(core.Flying) || len(g.DelayedTriggers()) != 1 {
					t.Fatal("missing flying or delayed destruction")
				}
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				g.RunStepWithPriority(core.EndStep)
				if g.FindPermanent(victim.ID()) != nil {
					t.Fatal("delayed destruction failed")
				}
			}
			if clone.FindPermanent(victim.ID()) == nil || clone.FindPermanent(victim.ID()).HasKeyword(core.Flying) || clone.FindPermanent(victim.ID()).CurrentPower(clone) != power {
				t.Fatal("clone shared effects")
			}
			clone.ResolveStack()
		})
	}
}

func TestRONNumericTargetsLegality(t *testing.T) {
	for _, name := range []string{"Pendelhaven", "Stone Giant"} {
		for _, change := range []string{"none", "target-counter", "source-counter", "source-depart", "source-return", "source-phase", "target-return", "target-phase", "phase-cycle", "shroud", "control", "copy"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				set, index, power, toughness := "LEG", 1, 1, 1
				if name == "Stone Giant" {
					set, index, power, toughness = "2ED", 0, 2, 2
				}
				card := repositoryCard(t, set, name)
				source := g.PutOnBattlefield(card, a.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				victimCard := mage.NewCreature("Victim", "", power, toughness)
				victimCard.SetOwner(b.PlayerID())
				victim := g.PutOnBattlefield(victimCard, a.PlayerID())
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, []uuid.UUID{victim.ID()}); err != nil {
					t.Fatal(err)
				}
				active := true
				switch change {
				case "target-counter":
					g.FindPermanent(victim.ID()).AddCounter(core.P1P1, 1)
					active = false
				case "source-counter":
					if name == "Stone Giant" {
						g.FindPermanent(source.ID()).AddCounter(core.M1M1, 1)
						active = false
					}
				case "source-depart", "source-return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if change == "source-return" {
						g.PutOnBattlefield(card, a.PlayerID()).AddCounter(core.M1M1, 3)
					}
				case "source-phase":
					g.PhaseOut(source.ID())
				case "target-return":
					g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
					victim = g.PutOnBattlefield(victimCard, a.PlayerID())
					active = false
				case "target-phase":
					g.PhaseOut(victim.ID())
					active = false
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(victim.ID()))
				case "shroud":
					g.FindPermanent(victim.ID()).GrantBaseAttr(core.Shroud)
					active = false
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: victim.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					active = name == "Pendelhaven"
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				g.ResolveStack()
				p := g.FindPermanentIncludingPhased(victim.ID())
				if name == "Stone Giant" {
					if p.HasKeyword(core.Flying) != active || (len(g.DelayedTriggers()) > 0) != active {
						t.Fatal("wrong legality or delay")
					}
					if active {
						p.GrantBaseAttr(core.Shroud)
						p.AddCounter(core.P1P1, 4)
						g.RunStepWithPriority(core.EndStep)
						if g.FindPermanent(victim.ID()) != nil {
							t.Fatal("delay repeated target legality")
						}
					}
				} else {
					want := power
					if change == "target-counter" {
						want++
					}
					if active {
						want++
					}
					if p.CurrentPower(g) != want {
						t.Fatalf("power=%d want=%d", p.CurrentPower(g), want)
					}
				}
			})
		}
	}
}

func TestRONNumericTargetsMetadataAndPayment(t *testing.T) {
	g, a, _ := ronGame()
	card := repositoryCard(t, "2ED", "Stone Giant")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid casting accepted")
	}
	if len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 3 || g.StackSize() != 0 {
		t.Fatal("payment was not rolled back")
	}
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	source := g.FindPermanent(card.ID())
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 2), a.PlayerID())
	if source.CurrentPower(g) != 3 || source.CurrentToughness(g) != 4 || !source.HasSubType("Giant") {
		t.Fatal("wrong metadata")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err == nil {
		t.Fatal("sick source tapped")
	}
	land := g.PutOnBattlefield(repositoryCard(t, "LEG", "Pendelhaven"), a.PlayerID())
	if !land.Card.HasSuperType(core.SuperLegendary) || !land.HasType(core.TypeLand) {
		t.Fatal("wrong land metadata")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), land.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if a.ManaPool().Count(core.Green) != 1 || g.StackSize() != 0 || !g.FindPermanent(land.ID()).Tapped {
		t.Fatal("wrong mana ability")
	}
}

func TestRONNumericTargetsInvalid(t *testing.T) {
	for _, filter := range []string{`PowerAndToughnessIsEqualTo`, `PowerAndToughnessIsEqualTo(PT(1))`, `PowerAndToughnessIsEqualTo(PT("1",1))`, `ToughnessIs`, `ToughnessIs(LessThan())`, `ToughnessIs(LessThan(PowerOfPermanent(Ref_TargetPermanent)))`, `ToughnessIs(LessThan(PowerOfPermanent(HostPermanent)))`, `ToughnessIs(LessThan(Integer(2)))`, `ToughnessIs(EqualTo(ValueX))`, `ToughnessIs(LessThanOrEqualTo(Integer("2")))`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(` + filter + `)],ActionList([TapPermanent(Ref_TargetPermanent)])))])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", filter)
		}
	}
	src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetPermanent(ToughnessIs(LessThan(PowerOfPermanent(ThisPermanent))))],ActionList([TapPermanent(Ref_TargetPermanent)])))])`
	nodes, err := Parse(strings.NewReader(src), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Compile(nodes[0]); err == nil {
		t.Fatal("spell accepted battlefield source")
	}
}

func TestRONNumericTargetsFixedQueries(t *testing.T) {
	for _, filter := range []string{`ToughnessIs(LessThanOrEqualTo(Integer(-1)))`, `ToughnessIs(EqualTo(Integer(-1)))`, `ToughnessIs(GreaterThanOrEqualTo(Integer(-1)))`, `PowerAndToughnessIsEqualTo(PT(-1,-1))`} {
		nodes, err := Parse(strings.NewReader(filter), "filter.ron")
		if err != nil {
			t.Fatal(err)
		}
		query, err := compileFilter(nodes[0])
		if err != nil {
			t.Fatal(err)
		}
		g, a, _ := ronGame()
		p := g.PutOnBattlefield(mage.NewCreature("Negative", "", -1, -1), a.PlayerID())
		if !query.Match(p, g) {
			t.Fatal("negative current value failed")
		}
		p.AddCounter(core.P1P1, 2)
		want := strings.Contains(filter, "GreaterThanOrEqualTo")
		if query.Match(p, g) != want {
			t.Fatal("filter ignored current counters")
		}
	}
}

func TestRONNumericTargetsDelayedIdentity(t *testing.T) {
	for _, change := range []string{"return", "phase", "phase-cycle", "control", "indestructible"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			source := g.PutOnBattlefield(repositoryCard(t, "2ED", "Stone Giant"), a.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			card := mage.NewCreature("Victim", "", 1, 2)
			card.SetOwner(a.PlayerID())
			victim := g.PutOnBattlefield(card, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			switch change {
			case "return":
				g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
				victim = g.PutOnBattlefield(card, a.PlayerID())
			case "phase":
				g.PhaseOut(victim.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(victim.ID()))
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: victim.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "indestructible":
				g.FindPermanent(victim.ID()).GrantBaseAttr(core.Indestructible)
			}
			g.RunStepWithPriority(core.EndStep)
			survives := change == "return" || change == "phase" || change == "indestructible"
			if (g.FindPermanentIncludingPhased(victim.ID()) != nil) != survives || len(g.DelayedTriggers()) != 0 {
				t.Fatal("wrong captured identity or delay lifetime")
			}
			if change == "phase" {
				g.PhaseIn([]uuid.UUID{victim.ID()})
				g.RunStepWithPriority(core.EndStep)
				if g.FindPermanent(victim.ID()) == nil {
					t.Fatal("consumed delay repeated")
				}
			}
		})
	}
}

func TestRONNumericTargetsInstances(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(PowerAndToughnessIsEqualTo(PT(1,1)))],ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AdjustPT(1,2)],UntilEndOfTurn)])))`)
	first, second := factory(), factory()
	if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("shared target declaration")
	}
	g, a, _ := ronGame()
	s1 := g.PutOnBattlefield(first, a.PlayerID())
	s2 := g.PutOnBattlefield(second, a.PlayerID())
	p1 := g.PutOnBattlefield(mage.NewCreature("First", "", 1, 1), a.PlayerID())
	p2 := g.PutOnBattlefield(mage.NewCreature("Second", "", 1, 1), a.PlayerID())
	for _, pair := range []struct{ source, target uuid.UUID }{{s1.ID(), p1.ID()}, {s2.ID(), p2.ID()}} {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), pair.source, 0, []uuid.UUID{pair.target}); err != nil {
			t.Fatal(err)
		}
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(p1.ID()).CurrentPower(g) != 1 || g.FindPermanent(p2.ID()).CurrentPower(g) != 2 {
		t.Fatal("bindings shared chosen state")
	}
	g.ResolveTopOfStack()
	if g.FindPermanent(p1.ID()).CurrentPower(g) != 2 {
		t.Fatal("first binding lost")
	}
}

func TestRONNumericTargetsInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:3,Toughness:4),Rules:[Activated(TapPermanent(ThisPermanent),Targeted([TargetPermanent(And([PowerAndToughnessIsEqualTo(PT(1,1)),ToughnessIs(LessThan(PowerOfPermanent(ThisPermanent)))]))],ActionList([TapPermanent(Ref_TargetPermanent)])))])`
	report, err := InspectFS(fstest.MapFS{"test.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 {
		t.Fatalf("inventory compile failed: %v", report.Issues)
	}
	found := map[string]bool{}
	for _, s := range report.Signatures {
		if s.Name == "LessThan" || s.Name == "PowerOfPermanent" || s.Name == "PT" {
			if s.Role == "unknown" {
				t.Fatalf("numeric descendant has unknown role: %s", s.Name)
			}
			found[s.Name] = true
		}
	}
	if len(found) != 3 {
		t.Fatal("numeric descendants missing")
	}
}

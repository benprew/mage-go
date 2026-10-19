package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const permanentAbsenceRequirement = `CastEffect(CantBeCastUnless(NoPermanentsPassFilter(IsNamed(NamedCard("Tidal Influence")),OnTheBattlefield)))`

func TestRONPermanentCastingRequirementTidal(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "FEM", "Tidal Influence")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast succeeded")
	}
	if len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("failed payment changed state")
	}
	a.ManaPool().Add(core.Blue, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	blueA := g.PutOnBattlefield(mage.NewCreature("Blue A", "{U}", 4, 5), a.PlayerID())
	blueB := g.PutOnBattlefield(mage.NewCreature("Blue B", "{U}", 3, 5), b.PlayerID())
	green := g.PutOnBattlefield(mage.NewCreature("Green", "{G}", 4, 5), b.PlayerID())
	check := func(counters, bonus int) {
		t.Helper()
		g.ApplyContinuousEffects()
		if got := g.FindPermanent(card.ID()).Counters[core.Tide]; got != counters {
			t.Fatalf("counters = %d, want %d", got, counters)
		}
		for _, subject := range []struct {
			id    uuid.UUID
			power int
		}{{blueA.ID(), 4 + bonus}, {blueB.ID(), 3 + bonus}, {green.ID(), 4}} {
			p := g.FindPermanent(subject.id)
			if p.CurrentPower(g) != subject.power || p.CurrentToughness(g) != 5 {
				t.Fatalf("%s = %d/%d, want %d/5", p.Card.Name(), p.CurrentPower(g), p.CurrentToughness(g), subject.power)
			}
		}
	}
	if a.ManaPool().TotalMana() != 0 {
		t.Fatal("casting did not pay mana")
	}
	check(1, -2)
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Upkeep)
	check(1, -2)
	g.SetActivePlayerIndex(0)
	for _, want := range []struct{ counters, bonus int }{{2, 0}, {3, 2}, {0, 0}, {1, -2}} {
		g.RunStepWithPriority(core.Upkeep)
		check(want.counters, want.bonus)
	}
	g.MutablePermanent(card.ID()).AddCounter(core.Tide, 2)
	check(3, 2)
	g.PhaseOut(card.ID())
	g.ApplyContinuousEffects()
	if g.FindPermanent(blueA.ID()).CurrentPower(g) != 4 {
		t.Fatal("phased source supplied layer")
	}
	g.PhaseIn([]uuid.UUID{card.ID()})
	check(3, 2)
	g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
	g.RunStepWithPriority(core.Upkeep)
	check(3, 2)
	g.SetActivePlayerIndex(1)
	g.RunStepWithPriority(core.Upkeep)
	check(0, 0)
}

func TestRONPermanentCastingRequirementMembership(t *testing.T) {
	for _, scenario := range []string{"own", "opponent", "token", "phased", "returned", "on-stack", "late-entry", "copy", "clone"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "FEM", "Tidal Influence")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 3)
			otherCard := repositoryCard(t, "FEM", "Tidal Influence")
			otherCard.SetOwner(b.PlayerID())
			blocked := scenario == "own" || scenario == "opponent" || scenario == "token" || scenario == "returned"
			if scenario != "on-stack" && scenario != "late-entry" && scenario != "copy" && scenario != "clone" {
				controller := b.PlayerID()
				if scenario == "own" {
					controller = a.PlayerID()
				}
				if scenario == "token" {
					otherCard = mage.NewToken("Tidal Influence", 1, 1, []core.CardType{core.TypeCreature}, nil)
				}
				other := g.PutOnBattlefield(otherCard, controller)
				if scenario == "phased" {
					g.PhaseOut(other.ID())
				}
				if scenario == "returned" {
					g.BouncePermanentToHand(other)
					b.RemoveFromHand(otherCard.ID())
					g.PutOnBattlefield(otherCard, b.PlayerID())
				}
			}
			if scenario == "on-stack" {
				mage.WithKeyword(core.Flash)(card.(*mage.BaseCard))
				b.AddToHand(otherCard)
				if err := g.CastCardFromZoneWithoutPaying(b.PlayerID(), otherCard.ID(), core.ZoneHand, nil, 0); err != nil {
					t.Fatal(err)
				}
			}
			stackBefore := g.StackSize()
			if got := len(g.GetCastableSpells(a.PlayerID())); (got == 0) != blocked {
				t.Fatalf("castable = %d, blocked = %v", got, blocked)
			}
			err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0)
			if (err != nil) != blocked {
				t.Fatalf("cast = %v, blocked = %v", err, blocked)
			}
			if blocked {
				if len(a.Hand()) != 1 || a.ManaPool().TotalMana() != 3 || g.StackSize() != stackBefore {
					t.Fatal("rejected requirement spent resources")
				}
				return
			}
			if scenario == "late-entry" || scenario == "copy" || scenario == "clone" {
				g.PutOnBattlefield(otherCard, b.PlayerID())
			}
			if scenario == "copy" && g.CopySpellOnStack(card.ID(), b.PlayerID(), false) == nil {
				t.Fatal("spell copy repeated requirement")
			}
			if scenario == "clone" {
				clone := g.Clone()
				clone.ResolveStack()
				if clone.FindPermanent(card.ID()) == nil || g.FindPermanent(card.ID()) != nil {
					t.Fatal("clone resolution or isolation")
				}
			}
			g.ResolveStack()
			if g.FindPermanent(card.ID()) == nil {
				t.Fatal("resolution repeated requirement")
			}
		})
	}
}

func TestRONPermanentCastingRequirementInvalid(t *testing.T) {
	for _, declaration := range []string{
		`CastEffect`, `CastEffect()`, `CastEffect(CantBeCastUnless())`,
		`CastEffect(CantBeCastUnless(NoPermanentsPassFilter()))`,
		`CastEffect(CantBeCastUnless(NoPermanentsPassFilter(AnyPermanent)))`,
		`CastEffect(CantBeCastUnless(NoPermanentsPassFilter(AnyPermanent,OnTheBattlefield,OnTheBattlefield)))`,
		`CastEffect(CantBeCastUnless(NoPermanentsPassFilter(IsNamed(NamedCard("")),OnTheBattlefield)))`,
		strings.Replace(permanentAbsenceRequirement, `OnTheBattlefield`, `OnTheBattlefield()`, 1),
		strings.Replace(permanentAbsenceRequirement, `OnTheBattlefield`, `InGraveyard`, 1),
		strings.Replace(permanentAbsenceRequirement, `IsNamed(NamedCard("Tidal Influence"))`, `SinglePermanent(Ref_TargetPermanent)`, 1),
		strings.Replace(permanentAbsenceRequirement, `IsNamed(NamedCard("Tidal Influence"))`, `Other(ThisPermanent)`, 1),
		strings.Replace(permanentAbsenceRequirement, `IsNamed(NamedCard("Tidal Influence"))`, `ControlledByAPlayer(SinglePlayer(You))`, 1),
		permanentAbsenceRequirement + `,` + permanentAbsenceRequirement,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[`+declaration+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Errorf("accepted %s", declaration)
		}
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid land",Typeline:(Supertypes:[],Cardtypes:[Land],Subtypes:[]),Rules:[`+permanentAbsenceRequirement+`])`), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("land accepted casting requirement")
	}
}

func TestRONPermanentCastingRequirementComposition(t *testing.T) {
	for _, kind := range []string{"Artifact", "Creature", "Enchantment"} {
		for _, filter := range []string{
			`IsNamed(NamedCard("Blocker"))`,
			`And([IsCardtype(Land),IsNamed(NamedCard("Blocker"))])`,
			`Or([IsNamed(NamedCard("Blocker")),IsCardtype(Creature)])`,
			`ExceptFor(IsNonCardtype(Land))`,
		} {
			t.Run(kind+"/"+filter, func(t *testing.T) {
				g, a, b := ronGame()
				rule := strings.Replace(permanentAbsenceRequirement, `IsNamed(NamedCard("Tidal Influence"))`, filter, 1)
				source := `Card(Name:"Scope",Typeline:(Supertypes:[],Cardtypes:[` + kind + `],Subtypes:[]),ManaCost:[],Rules:[` + rule + `]`
				if kind == "Creature" {
					source += `,CardPT:(Power:1,Toughness:1)`
				}
				nodes, err := Parse(strings.NewReader(source+`)`), "composition.ron")
				if err != nil {
					t.Fatal(err)
				}
				factory, err := Compile(nodes[0])
				if err != nil {
					t.Fatal(err)
				}
				card := factory()
				fresh := factory()
				if card.Abilities()[0].AbilityID() == fresh.Abilities()[0].AbilityID() {
					t.Fatal("factory shared casting action")
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				blocker := g.PutOnBattlefield(mage.NewLand("Blocker"), b.PlayerID())
				if len(g.GetCastableSpells(a.PlayerID())) != 0 {
					t.Fatal("composed filter did not prevent casting")
				}
				g.BouncePermanentToHand(blocker)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if g.FindPermanent(card.ID()) == nil {
					t.Fatal("permanent did not resolve")
				}
			})
		}
	}
}

func TestRONPermanentCastingRequirementInventory(t *testing.T) {
	files := fstest.MapFS{"casting.ron": &fstest.MapFile{Data: []byte(`Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[` + permanentAbsenceRequirement + `])`)}}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 || len(report.Issues) != 0 {
		t.Fatal("requirement did not compile")
	}
	for name, role := range map[string]string{"CastEffect": "rule", "CantBeCastUnless": "casting_restriction", "NoPermanentsPassFilter": "casting_condition", "IsNamed": "permanent_filter", "NamedCard": "card_name", "OnTheBattlefield": "zone"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name {
				found = true
				if signature.Role != role {
					t.Fatalf("%s role = %s, want %s", name, signature.Role, role)
				}
			}
		}
		if !found {
			t.Fatal("missing inventory role", name)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[CastEffect(CantBeCastUnless(NoPermanentsPassFilter(AnyPermanent,InGraveyard)))])`)}
	if registry, err := LoadFS(files, "*.ron"); err == nil || registry != nil {
		t.Fatal("invalid collection was published")
	}
}

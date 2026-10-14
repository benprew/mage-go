package ron

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

const sourceCounterManaRule = `Activated(TapPermanent(ThisPermanent),ActionList([AddManaRepeated(ManaProduceC,TheTotalNumberOfCountersOfTypeAmongPermanents(StorageCounter,SinglePermanent(ThisPermanent)))]))`

func TestRONSourceCounterManaCity(t *testing.T) {
	for _, scenario := range []string{"normal", "copy-counter-rule", "counter-counter-rule"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "City of Shadows")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.PlayLand(a.PlayerID(), card.ID()); err != nil {
				t.Fatal(err)
			}
			p := g.FindPermanent(card.ID())
			victim := mage.NewCreature("Victim", "", 2, 2)
			victim.SetOwner(b.PlayerID())
			g.PutOnBattlefield(victim, a.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			if g.FindExiledCard(victim.ID()) == nil || p.Counters[core.Storage] != 0 || g.StackSize() != 1 {
				t.Fatal("exile payment and counter timing")
			}
			want := 1
			if scenario == "copy-counter-rule" {
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
				want = 2
			}
			if scenario == "counter-counter-rule" {
				g.CounterStackObject(g.StackPeek().ID)
				want = 0
			}
			g.ResolveStack()
			g.MutablePermanent(p.ID()).Tapped = false
			if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 1, nil); err != nil {
				t.Fatal(err)
			}
			if a.ManaPool().Count(core.Colorless) != want || g.StackSize() != 0 || p.Counters[core.Storage] != want || len(g.GetExile()) != 1 {
				t.Fatal("complete City mana/counter interaction")
			}
		})
	}
}

func TestRONSourceCounterManaPayment(t *testing.T) {
	for _, scenario := range []string{"success", "short", "zero", "phased", "removed", "control", "replacement", "phase-cycle"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			p := g.PutOnBattlefield(repositoryCard(t, "DRK", "City of Shadows"), a.PlayerID())
			p.Counters[core.Storage] = 3
			payer := a
			cost := "{3}"
			valid := true
			switch scenario {
			case "short":
				cost = "{4}"
				valid = false
			case "zero":
				p.Counters[core.Storage] = 0
				valid = false
			case "phased":
				g.PhaseOut(p.ID())
				valid = false
			case "removed":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
				g.ApplyContinuousEffects()
				valid = false
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				payer = b
				g.SetActivePlayerIndex(1)
			case "replacement":
				cost = "{U}{U}{U}"
				if err := mage.ApplyEffect(g, mage.ReplaceTappedManaTypeForPlayerThisTurn(mage.You, mage.IsLand, core.Blue), p.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(p.ID()))
			}
			spell := mage.NewArtifact("Payment", cost)
			spell.SetOwner(payer.PlayerID())
			payer.AddToHand(spell)
			clone := g.Clone()
			err := g.CastSpellByID(payer.PlayerID(), spell.ID(), nil, 0)
			if (err == nil) != valid {
				t.Fatalf("cast error = %v", err)
			}
			if g.FindPermanentIncludingPhased(p.ID()).Tapped != valid || payer.ManaPool().TotalMana() != 0 {
				t.Fatal("payment or rollback")
			}
			if valid && g.StackSize() != 1 || !valid && (g.StackSize() != 0 || len(payer.Hand()) != 1) {
				t.Fatal("cast state")
			}
			if clone.FindPermanentIncludingPhased(p.ID()).Tapped {
				t.Fatal("clone tapped")
			}
			cloneErr := clone.CastSpellByID(payer.PlayerID(), spell.ID(), nil, 0)
			if (cloneErr == nil) != valid {
				t.Fatalf("clone cast = %v", cloneErr)
			}
		})
	}
}

func TestRONSourceCounterManaInvalid(t *testing.T) {
	for _, rule := range []string{
		strings.ReplaceAll(sourceCounterManaRule, "StorageCounter", "UnknownCounter"),
		strings.ReplaceAll(sourceCounterManaRule, "StorageCounter", "StorageCounter()"),
		strings.Replace(sourceCounterManaRule, "SinglePermanent(ThisPermanent)", "SinglePermanent(ThisPermanent())", 1),
		strings.Replace(sourceCounterManaRule, "SinglePermanent(ThisPermanent)", "SinglePermanent(Ref_TargetPermanent)", 1),
		strings.Replace(sourceCounterManaRule, "SinglePermanent(ThisPermanent)", "SinglePermanent(HostPermanent)", 1),
		strings.Replace(sourceCounterManaRule, "SinglePermanent(ThisPermanent)", "EachPermanent(AnyPermanent)", 1),
		strings.Replace(sourceCounterManaRule, "SinglePermanent(ThisPermanent)", "SinglePermanent()", 1),
		strings.Replace(sourceCounterManaRule, "StorageCounter,SinglePermanent(ThisPermanent)", "StorageCounter", 1),
		strings.Replace(sourceCounterManaRule, "ManaProduceC", "AnyManaColor", 1),
		strings.Replace(sourceCounterManaRule, "ManaProduceC", "And([ManaProduceC,ManaProduceU])", 1),
		strings.Replace(sourceCounterManaRule, "TapPermanent(ThisPermanent)", "TapPermanent(HostPermanent)", 1),
		strings.Replace(sourceCounterManaRule, "TapPermanent(ThisPermanent)", "PayMana([])", 1),
		strings.Replace(sourceCounterManaRule, "ActionList([", "ActionList([DrawACard,", 1),
		strings.Replace(sourceCounterManaRule, "Activated(", "ActivatedWithModifiers(", 1),
		`SpellActions(ActionList([AddManaRepeated(ManaProduceC,TheTotalNumberOfCountersOfTypeAmongPermanents(StorageCounter,SinglePermanent(ThisPermanent)))]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Land],Subtypes:[]),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("invalid declaration accepted", rule)
		}
	}
}

func TestRONSourceCounterManaRegistry(t *testing.T) {
	record := repositoryRecord(t, "DRK", "City of Shadows", "city.ron")
	data, err := os.ReadFile("../../../../cards/mtgish/DRK.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	line := strings.Split(string(data), "\n")[record.Position.Line-1]
	files := fstest.MapFS{"city.ron": {Data: []byte(line)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("City of Shadows")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("City of Shadows")
	if err != nil {
		t.Fatal(err)
	}
	if first.Abilities()[1] == second.Abilities()[1] || !mage.IsActivatedManaAbility(first.Abilities()[1]) {
		t.Fatal("factory state or classification")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal("inventory compilation", err)
	}
	for name, role := range map[string]string{"TheTotalNumberOfCountersOfTypeAmongPermanents": "value", "StorageCounter": "permanent_counter_type", "SinglePermanent": "permanent_filter", "ThisPermanent": "permanent_reference"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatal("missing inventory role", name, role)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(line, "StorageCounter, SinglePermanent(ThisPermanent)", "StorageCounter, SinglePermanent(Ref_TargetPermanent)", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || !registry.CardRegistered("City of Shadows") || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection published")
	}
}

func TestRONSourceCounterManaSourceContext(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+sourceCounterManaRule+`])`), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("nonpermanent source accepted")
	}
}

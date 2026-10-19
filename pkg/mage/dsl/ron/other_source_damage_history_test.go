package ron

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONOtherSourceDamageHistoryEffigy(t *testing.T) {
	g, a, b := opponentTargetGame()
	g.SetTurn(1)
	card := repositoryCard(t, "LEG", "Blazing Effigy")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast")
	}
	a.ManaPool().Add(core.Red, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 0, 30), b.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Blazing Effigy", "", 0, 20), b.PlayerID())
	g.DealDamageToPermanent(p, 2, q.ID())
	g.DestroyPermanent(p)
	a.requested = []uuid.UUID{victim.ID()}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatalf("stack %d", g.StackSize())
	}
	g.ResolveStack()
	if victim.Damage != 5 {
		t.Fatalf("damage %d, want 5", victim.Damage)
	}
}

func TestRONOtherSourceDamageHistoryInvalid(t *testing.T) {
	valid := `DeadPermanentDealsDamage(Plus(Integer(3),TheAmountOfDamageDealtToPermanentThisTurnBySources(Trigger_ThatPermanent,And([Other(Trigger_ThatPermanent),IsNamed(NamedCard("Blazing Effigy"))]))),Permanent(Ref_TargetPermanent))`
	for _, bad := range []string{
		strings.Replace(valid, `Trigger_ThatPermanent,And`, `You,And`, 1),
		strings.Replace(valid, `Other(Trigger_ThatPermanent)`, `Other(ThisPermanent)`, 1),
		strings.Replace(valid, `Other(Trigger_ThatPermanent),`, ``, 1),
		strings.Replace(valid, `NamedCard("Blazing Effigy")`, `NamedCard("")`, 1),
		strings.Replace(valid, `IsNamed(NamedCard("Blazing Effigy"))`, `IsNamed(NamedCard("Blazing Effigy"),NamedCard("Other"))`, 1),
		strings.Replace(valid, `Permanent(Ref_TargetPermanent)`, `Player(Trigger_ThatPlayer)`, 1),
		strings.Replace(valid, `Integer(3)`, `Integer(3,4)`, 1),
		strings.Replace(valid, `Other(Trigger_ThatPermanent)`, `Other(Trigger_ThatPermanent,ThisPermanent)`, 1),
		strings.Replace(valid, `IsNamed(NamedCard("Blazing Effigy"))`, `Other(Trigger_ThatPermanent)`, 1),
		strings.Replace(valid, `Other(Trigger_ThatPermanent)`, `IsNamed(NamedCard("Other"))`, 1),
		strings.Replace(valid, `TheAmountOfDamageDealtToPermanentThisTurnBySources(Trigger_ThatPermanent,`, `TheAmountOfDamageDealtToPermanentThisTurnBySources(Trigger_ThatPermanent,ThisPermanent,`, 1),
		strings.Replace(valid, `NamedCard("Blazing Effigy")`, `NamedCard("   ")`, 1),
		strings.Replace(valid, `And([Other`, `Or([Other`, 1),
		valid,
	} {
		record := staticBaseRecord(t, "LEG", "Blazing Effigy")
		if bad != valid {
			nodes, err := Parse(strings.NewReader(`TriggerA(WhenAPermanentDies(SinglePermanent(ThisPermanent)),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([`+bad+`])))`), "bad.ron")
			if err != nil {
				t.Fatal(err)
			}
			rules := record.Fields["Rules"]
			rules.Args = []Node{nodes[0]}
			record.Fields["Rules"] = rules
			if _, err := Compile(record); err == nil {
				t.Fatal("invalid accepted", bad)
			}
		} else {
			nodes, err := Parse(strings.NewReader(valid), "unbound.ron")
			if err != nil {
				t.Fatal(err)
			}
			for _, ctx := range []scope{{permanent: true, resolving: true}, {resolving: true}, {permanent: true, resolving: true, enteringSource: true}, {permanent: true, resolving: true, drawReplacementBody: true}} {
				if _, err := compileEffect(nodes[0], ctx); err == nil {
					t.Fatal("unbound death damage accepted")
				}
			}
		}
	}
}

func TestRONOtherSourceDamageHistoryLifetime(t *testing.T) {
	for _, change := range []string{"normal", "return", "phase-cycle", "phased", "exiled", "regenerated", "removed", "copy", "clone", "target-phase", "target-return", "protection", "prevention"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := opponentTargetGame()
			card := repositoryCard(t, "LEG", "Blazing Effigy")
			card.SetOwner(a.PlayerID())
			p := g.PutOnBattlefield(card, a.PlayerID())
			q := g.PutOnBattlefield(mage.NewCreature("Blazing Effigy", "", 0, 30), b.PlayerID())
			target := g.PutOnBattlefield(mage.NewCreature("Victim", "", 0, 30), b.PlayerID())
			a.requested = []uuid.UUID{target.ID()}
			g.AddPreventionShield(p.ID(), 1)
			g.DealDamageToPermanent(p, 3, q.ID())
			switch change {
			case "phase-cycle":
				g.PhaseOut(p.ID())
				g.PhaseIn([]uuid.UUID{p.ID()})
			case "phased":
				g.PhaseOut(p.ID())
			case "regenerated":
				g.AddRegenerationShield(p.ID())
			case "removed":
				g.AddContinuousEffect(mage.RemoveAllAbilities(p.ID(), core.Indefinite))
			}
			if change == "exiled" {
				g.ExilePermanent(p)
			} else {
				g.DestroyPermanent(p)
			}
			g.PutTriggersOnStack()
			triggers := 1
			if change == "phased" || change == "regenerated" || change == "removed" || change == "exiled" {
				triggers = 0
			}
			if g.StackSize() != triggers {
				t.Fatalf("stack %d want %d", g.StackSize(), triggers)
			}
			switch change {
			case "return":
				a.RemoveFromGraveyard(card.ID())
				g.PutOnBattlefield(card, b.PlayerID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			case "target-phase":
				g.PhaseOut(target.ID())
			case "target-return":
				g.BouncePermanentToHand(target)
				b.RemoveFromHand(target.ID())
				target = g.PutOnBattlefield(target.Card, b.PlayerID())
			case "protection":
				if err := mage.ApplyEffect(g, mage.Until(core.Indefinite, mage.GrantProtectionColors(mage.This, core.Red)), target.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "prevention":
				g.AddPreventionShield(target.ID(), 2)
			}
			cloned := g.Clone()
			if change == "clone" {
				cloned.AddPreventionShield(target.ID(), 1)
			}
			g.ResolveStack()
			want := 5 * triggers
			switch change {
			case "copy":
				want *= 2
			case "target-phase", "target-return", "protection":
				want = 0
			case "prevention":
				want -= 2
			}
			if got := g.FindPermanentIncludingPhased(target.ID()).Damage; got != want {
				t.Fatalf("damage %d want %d", got, want)
			}
			if cloned.FindPermanentIncludingPhased(target.ID()).Damage != 0 {
				t.Fatal("original changed clone")
			}
			cloned.ResolveStack()
			if change == "clone" {
				want--
			}
			if got := cloned.FindPermanentIncludingPhased(target.ID()).Damage; got != want {
				t.Fatalf("clone damage %d want %d", got, want)
			}
		})
	}
}

func TestRONOtherSourceDamageHistoryChain(t *testing.T) {
	g, a, b := opponentTargetGame()
	first := repositoryCard(t, "LEG", "Blazing Effigy")
	first.SetOwner(a.PlayerID())
	second := repositoryCard(t, "LEG", "Blazing Effigy")
	second.SetOwner(b.PlayerID())
	p := g.PutOnBattlefield(first, a.PlayerID())
	q := g.PutOnBattlefield(second, b.PlayerID())
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 0, 30), a.PlayerID())
	a.requested = []uuid.UUID{q.ID()}
	b.requested = []uuid.UUID{victim.ID()}
	g.DestroyPermanent(p)
	g.PutTriggersOnStack()
	g.ResolveStack()
	g.CheckStateBasedActions()
	g.PutTriggersOnStack()
	g.ResolveStack()
	if g.FindPermanent(q.ID()) != nil || g.FindPermanent(victim.ID()).Damage != 6 {
		t.Fatalf("chain damage %d want 6, stack %d, q %v", g.FindPermanent(victim.ID()).Damage, g.StackSize(), g.FindPermanent(q.ID()))
	}
}

func TestRONOtherSourceDamageHistoryLoadAndInventory(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("../../../../cards/mtgish", "LEG.lines.ron"))
	if err != nil {
		t.Fatal(err)
	}
	var record string
	for line := range strings.SplitSeq(string(data), "\n") {
		if strings.Contains(line, `Name: "Blazing Effigy"`) {
			record = line
		}
	}
	files := fstest.MapFS{"effigy.ron": {Data: []byte(record)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, err := registry.CreateCard("Blazing Effigy")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Blazing Effigy")
	if err != nil {
		t.Fatal(err)
	}
	if first == second || first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared factory instance")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{"TheAmountOfDamageDealtToPermanentThisTurnBySources": "value", "Trigger_ThatPermanent": "permanent_reference", "And": "damage_source_filter", "Other": "damage_source_filter", "IsNamed": "damage_source_filter", "NamedCard": "card_name"}
	for name, role := range roles {
		found := false
		for _, use := range report.Signatures {
			if use.Name == name && use.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing inventory %s %s", name, role)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(record, `Other(Trigger_ThatPermanent)`, `Other(You)`, 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil {
		t.Fatal("invalid collection accepted")
	}
	if _, err := registry.CreateCard("Blazing Effigy"); err != nil {
		t.Fatal("atomic load changed registry", err)
	}
}

func TestRONOtherSourceDamageHistoryIndependentDeaths(t *testing.T) {
	g, a, b := opponentTargetGame()
	first := repositoryCard(t, "LEG", "Blazing Effigy")
	first.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(first, a.PlayerID())
	second := repositoryCard(t, "LEG", "Blazing Effigy")
	second.SetOwner(b.PlayerID())
	q := g.PutOnBattlefield(second, b.PlayerID())
	source := g.PutOnBattlefield(mage.NewCreature("Blazing Effigy", "", 0, 20), a.PlayerID())
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 0, 30), b.PlayerID())
	a.requested = []uuid.UUID{victim.ID()}
	b.requested = []uuid.UUID{victim.ID()}
	g.DealSimultaneousDamage(mage.PermanentDamageAssignment(p, 1, g.CaptureDamageSource(source.ID())), mage.PermanentDamageAssignment(q, 2, g.CaptureDamageSource(source.ID())))
	g.DestroyPermanents([]*mage.Permanent{p, q})
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("independent deaths lost triggers")
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	if g.FindPermanent(victim.ID()).Damage != 9 || clone.FindPermanent(victim.ID()).Damage != 9 {
		t.Fatal("death facts were shared")
	}
}

func TestRONOtherSourceDamageHistoryNoTarget(t *testing.T) {
	g, a, _ := opponentTargetGame()
	first := repositoryCard(t, "LEG", "Blazing Effigy")
	first.SetOwner(a.PlayerID())
	p := g.PutOnBattlefield(first, a.PlayerID())
	g.DestroyPermanent(p)
	g.PutTriggersOnStack()
	if g.StackSize() != 0 {
		t.Fatal("death without a legal target created a stack object")
	}
}

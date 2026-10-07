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

func TestRONReferencedDamageTracker(t *testing.T) {
	for _, scenario := range []string{"normal", "unpaid", "sick", "illegal", "target-return", "target-phase", "phase-cycle", "source-left", "source-return", "source-phase", "control", "prevention", "clone", "copy", "sequential"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "DRK", "Tracker")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			if scenario != "sick" {
				source.RevokeBaseAttr(core.AttrSummonSick)
			}
			targetCard := mage.NewCreature("Victim", "", 3, 9)
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			target.GrantBaseAttr(core.Lifelink)
			source.GrantBaseAttr(core.Lifelink)
			if scenario != "unpaid" {
				a.ManaPool().Add(core.Green, 2)
			}
			id := target.ID()
			if scenario == "illegal" {
				id = b.PlayerID()
			}
			before := a.ManaPool().TotalMana()
			err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{id})
			if scenario == "sick" || scenario == "unpaid" || scenario == "illegal" {
				if err == nil || g.FindPermanent(source.ID()).Tapped || g.StackSize() != 0 || a.ManaPool().TotalMana() != before {
					t.Fatal("invalid activation did not roll back")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source.ID()).Tapped || a.ManaPool().TotalMana() != 0 || target.Damage != 0 {
				t.Fatal("announcement timing")
			}
			sourceID, targetID := source.ID(), target.ID()
			wantTarget, wantSource, gainA, gainB := 2, 3, 2, 3
			switch scenario {
			case "target-return":
				g.BouncePermanentToHand(target)
				b.RemoveFromHand(targetCard.ID())
				targetID = g.PutOnBattlefield(targetCard, b.PlayerID()).ID()
				wantTarget, wantSource, gainA, gainB = 0, 0, 0, 0
			case "target-phase":
				g.PhaseOut(target.ID())
				wantTarget, wantSource, gainA, gainB = 0, 0, 0, 0
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(target.ID()))
			case "source-left", "source-return":
				g.BouncePermanentToHand(source)
				wantSource, gainB = 0, 0
				if scenario == "source-return" {
					a.RemoveFromHand(card.ID())
					sourceID = g.PutOnBattlefield(card, a.PlayerID()).ID()
				}
			case "source-phase":
				g.PhaseOut(source.ID())
				wantSource, gainB = 0, 0
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: a.PlayerID(), TargetID: target.ID(), ControllerID: a.PlayerID(), Duration: core.Indefinite})
				gainA, gainB = 5, 0
			case "prevention":
				if err := mage.ApplyEffect(g, mage.PreventDamageThisTurn(mage.This, mage.AnyPermanent), target.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				wantTarget, gainA = 0, 0
			case "sequential":
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(0), mage.Fixed(-7))), target.ID(), b.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
					t.Fatal("missing copy")
				}
				wantTarget, wantSource, gainA, gainB = 4, 3, 4, 3
			}
			clone := g.Clone()
			if scenario == "clone" {
				clone.ResolveStack()
				if a.Life() != 20 || b.Life() != 20 {
					t.Fatal("clone changed original")
				}
			}
			g.ResolveStack()
			p := g.FindPermanentIncludingPhased(targetID)
			if scenario == "sequential" {
				if p != nil {
					t.Fatal("lethal target survived")
				}
			} else if p == nil || p.Damage != wantTarget {
				t.Fatalf("target damage: %v, want %d", p, wantTarget)
			}
			p = g.FindPermanentIncludingPhased(sourceID)
			if scenario == "normal" || scenario == "phase-cycle" || scenario == "control" || scenario == "prevention" || scenario == "clone" || scenario == "copy" || scenario == "sequential" {
				if p != nil {
					t.Fatal("lethal source survived")
				}
			} else if p != nil && p.Damage != wantSource {
				t.Fatalf("source damage = %d, want %d", p.Damage, wantSource)
			}
			if a.Life() != 20+gainA || b.Life() != 20+gainB {
				t.Fatalf("source lifelink attributed incorrectly: %d/%d", a.Life(), b.Life())
			}
		})
	}
}

func TestRONReferencedDamageCity(t *testing.T) {
	for _, scenario := range []string{"tap", "mana", "left", "return", "phase", "control", "clone", "repeat", "suppressed", "prevented"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "ARN", "City of Brass")
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			if scenario == "suppressed" {
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "mana" {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 1, nil); err != nil {
					t.Fatal(err)
				}
				if a.ManaPool().TotalMana() != 1 {
					t.Fatal("mana was not immediate")
				}
			} else {
				g.TapPermanent(source)
			}
			if a.Life() != 20 {
				t.Fatal("damage resolved with tapping")
			}
			want := 1
			switch scenario {
			case "left", "return":
				g.BouncePermanentToHand(g.FindPermanent(source.ID()))
				if scenario == "return" {
					a.RemoveFromHand(card.ID())
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "repeat":
				g.TapPermanent(g.FindPermanent(source.ID()))
				g.UntapPermanent(g.FindPermanent(source.ID()))
				g.TapPermanent(g.FindPermanent(source.ID()))
				want = 2
			case "suppressed":
				want = 0
			case "prevented":
				if err := mage.ApplyEffect(g, mage.PreventDamageThisTurn(mage.You, mage.AnyPermanent), source.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				want = 0
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 {
				t.Fatal("clone changed pending original")
			}
			g.ResolveStack()
			if a.Life() != 20-want || b.Life() != 20 {
				t.Fatalf("wrong captured trigger player: %d/%d", a.Life(), b.Life())
			}
		})
	}
}

func TestRONReferencedDamageInvalid(t *testing.T) {
	for _, body := range []string{
		`ActionList([PermanentDealsDamage(Ref_TargetPermanent, Integer(1), Player(You))])`,
		`Targeted([TargetPlayer(AnyPlayer)], ActionList([PermanentDealsDamage(Ref_TargetPermanent, Integer(1), Player(You))]))`,
		`ActionList([PermanentDealsDamage(ThisPermanent(), Integer(1), Player(You))])`,
		`ActionList([PermanentDealsDamage(HostPermanent, Integer(1), Player(You))])`,
		`ActionList([PermanentDealsDamage(Trigger_ThatPermanent, Integer(1), Player(You))])`,
		`ActionList([PermanentDealsDamage(ThisPermanent, Integer(1), Permanent(Ref_TargetPermanent))])`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [Activated(PayMana([]), %s)])`, body)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid source: %s", body)
		}
	}
	for _, subject := range []string{`ThisPermanent()`, `Ref_TargetPermanent`, `SinglePermanent(ThisPermanent, ThisPermanent)`, `SinglePermanent(You)`} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [TriggerA(WhenAPermanentBecomesTapped(%s), ActionList([PermanentDealsDamage(Trigger_ThatPermanent, Integer(1), Player(You))]))])`, subject)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted invalid event: %s", subject)
		}
	}
}

func TestRONReferencedDamageBindingsAndLKI(t *testing.T) {
	for _, scenario := range []string{"live", "departed", "phased", "host", "event"} {
		t.Run(scenario, func(t *testing.T) {
			g, a, b := ronGame()
			targetCard := mage.NewCreature("Damage source", "", 4, 8, mage.WithKeyword(core.Lifelink))
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			ref := "Ref_TargetPermanent"
			prefix := ""
			if scenario == "departed" {
				prefix = `PutPermanentIntoItsOwnersHand(Ref_TargetPermanent), `
			}
			rule := `Activated(PayMana([]), Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([` + prefix + `PermanentDealsDamage(` + ref + `, PowerOfPermanent(` + ref + `), Player(You))])))`
			if scenario == "host" {
				rule = `EnchantPermanent(IsCardtype(Creature)), Activated(PayMana([]), ActionList([PermanentDealsDamage(HostPermanent, PowerOfPermanent(HostPermanent), Player(You))]))`
			}
			if scenario == "event" {
				rule = `EnchantPermanent(IsCardtype(Creature)), TriggerA(WhenAPermanentBecomesTapped(SinglePermanent(HostPermanent)), ActionList([PermanentDealsDamage(Trigger_ThatPermanent, PowerOfPermanent(Trigger_ThatPermanent), Player(You))]))`
			}
			kind := "Artifact"
			if scenario == "host" || scenario == "event" {
				kind = "Enchantment"
			}
			subtypes := ""
			if kind == "Enchantment" {
				subtypes = "Aura"
			}
			nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name: "Binding", Typeline: (Supertypes: [], Cardtypes: [%s], Subtypes: [%s]), Rules: [%s])`, kind, subtypes, rule)), "binding.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			first, second := factory(), factory()
			if err := mage.ValidateCardDefinition(first.Name(), first); err != nil {
				t.Fatal(err)
			}
			first.SetOwner(a.PlayerID())
			second.SetOwner(b.PlayerID())
			source := g.PutOnBattlefield(first, a.PlayerID())
			g.PutOnBattlefield(second, b.PlayerID())
			if scenario == "host" || scenario == "event" {
				g.Attach(source.ID(), target.ID())
			}
			if scenario == "event" {
				g.TapPermanent(target)
				g.BouncePermanentToHand(target)
			} else {
				var targets []uuid.UUID
				if scenario != "host" {
					targets = []uuid.UUID{target.ID()}
				}
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, targets); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "phased" {
				g.PhaseOut(target.ID())
			}
			clone := g.Clone()
			clone.ResolveStack()
			if a.Life() != 20 || b.Life() != 20 {
				t.Fatal("shared source binding")
			}
			g.ResolveStack()
			wantA, wantB := 16, 24
			if scenario == "phased" {
				wantA, wantB = 20, 20
			}
			if a.Life() != wantA || b.Life() != wantB || clone.GetPlayer(a.PlayerID()).Life() != wantA || clone.GetPlayer(b.PlayerID()).Life() != wantB {
				t.Fatalf("incorrect identity/LKI damage %d/%d", a.Life(), b.Life())
			}
		})
	}
}

func TestRONReferencedDamageCastingAndLandPlay(t *testing.T) {
	g, a, _ := ronGame()
	tracker := repositoryCard(t, "DRK", "Tracker")
	tracker.SetOwner(a.PlayerID())
	a.AddToHand(tracker)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), tracker.ID(), nil, 0); err == nil || a.ManaPool().TotalMana() != 2 || len(a.Hand()) != 1 || g.StackSize() != 0 {
		t.Fatal("casting payment did not roll back")
	}
	a.ManaPool().Add(core.Green, 1)
	if err := g.CastSpellByID(a.PlayerID(), tracker.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(tracker.ID()) != nil {
		t.Fatal("early creature entry")
	}
	g.ResolveStack()
	p := g.FindPermanent(tracker.ID())
	if p == nil || p.CurrentPower(g) != 2 || p.CurrentToughness(g) != 2 || !p.HasType(core.TypeCreature) {
		t.Fatal("Tracker metadata")
	}
	city := repositoryCard(t, "ARN", "City of Brass")
	city.SetOwner(a.PlayerID())
	a.AddToHand(city)
	if err := g.PlayLand(a.PlayerID(), city.ID()); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(city.ID()) == nil || a.Life() != 20 || g.StackSize() != 0 {
		t.Fatal("land entry fired tap event")
	}
}

func TestRONReferencedDamageInventoryAndAtomicLoad(t *testing.T) {
	records := []string{}
	entries := map[string]*InventorySignature{}
	for _, tc := range []struct{ set, name string }{{"ARN", "City of Brass"}, {"DRK", "Tracker"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), tc.set+".ron")
		if err != nil {
			t.Fatal(err)
		}
		for _, n := range nodes {
			if n.Fields["Name"].String == tc.name {
				records = append(records, strings.Split(string(data), "\n")[n.Position.Line-1])
				walkInventory(n, tc.name, "Card", "card", "collection", true, entries)
			}
		}
	}
	if len(records) != 2 {
		t.Fatal("missing complete records")
	}
	files := fstest.MapFS{"slice.ron": {Data: []byte(strings.Join(records, "\n"))}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("wrong registry")
	}
	for _, name := range []string{"ThisPermanent", "Ref_TargetPermanent", "Trigger_ThatPermanent"} {
		found := false
		for _, entry := range entries {
			if entry.Name == name && entry.Role == "permanent_reference" {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing typed inventory role: %s", name)
		}
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(records[1], "PermanentDealsDamage(Ref_TargetPermanent", "PermanentDealsDamage(Trigger_ThatPermanent", 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("invalid load replaced accepted registry")
	}
}

func TestRONReferencedDamageCopyRetarget(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "DRK", "Tracker")
	card.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	source.GrantBaseAttr(core.Lifelink)
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.ModifyPT(mage.This, mage.Fixed(0), mage.Fixed(20))), source.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	first := g.PutOnBattlefield(mage.NewCreature("First", "", 3, 9, mage.WithKeyword(core.Lifelink)), b.PlayerID())
	second := g.PutOnBattlefield(mage.NewCreature("Second", "", 5, 9, mage.WithKeyword(core.Lifelink)), b.PlayerID())
	a.ManaPool().Add(core.Green, 2)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{first.ID()}); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	phaseSource, phaseFirst := g.PhaseOut(source.ID()), g.PhaseOut(first.ID())
	copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != second.ID() {
		t.Fatal("copy did not retarget")
	}
	g.PhaseIn(phaseSource)
	g.PhaseIn(phaseFirst)
	g.ResolveStack()
	clone.ResolveStack()
	if g.FindPermanent(first.ID()).Damage != 2 || g.FindPermanent(second.ID()).Damage != 2 || g.FindPermanent(source.ID()).Damage != 8 || a.Life() != 24 || b.Life() != 28 {
		t.Fatal("copy changed the source or recipient binding")
	}
	if clone.FindPermanent(second.ID()).Damage != 0 || clone.FindPermanent(source.ID()).Damage != 3 || clone.GetPlayer(a.PlayerID()).Life() != 22 || clone.GetPlayer(b.PlayerID()).Life() != 23 {
		t.Fatal("copy mutated cloned original frame")
	}
}

func TestRONReferencedDamageCityInstancesAndCopy(t *testing.T) {
	g, a, b := ronGame()
	for _, player := range []*mage.BasePlayer{a, b} {
		card := repositoryCard(t, "ARN", "City of Brass")
		card.SetOwner(player.PlayerID())
		p := g.PutOnBattlefield(card, player.PlayerID())
		g.TapPermanent(p)
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 2 {
		t.Fatal("independent tap events lost")
	}
	copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false)
	if copied == nil {
		t.Fatal("pending trigger copy failed")
	}
	g.ResolveStack()
	if a.Life() != 19 || b.Life() != 18 {
		t.Fatalf("source instances or copy controller mixed: %d/%d", a.Life(), b.Life())
	}
}

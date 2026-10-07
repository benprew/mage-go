package ron

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCollectionDamageRealCards(t *testing.T) {
	for _, name := range []string{"Sandstorm", "Winter Blast"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			set := "ARN"
			if name == "Winter Blast" {
				set = "LEG"
				g.SetStep(core.PrecombatMain)
			}
			card := repositoryCard(t, set, name)
			if err := mage.ValidateCardDefinition(name, card); err != nil {
				t.Fatal(err)
			}
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			first := g.PutOnBattlefield(mage.NewCreature("Flyer", "", 2, 8, mage.WithKeyword(core.Flying)), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Ground", "", 2, 8), b.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 8, mage.WithKeyword(core.Flying)), b.PlayerID())
			targets := []uuid.UUID(nil)
			x := 0
			if name == "Winter Blast" {
				targets = []uuid.UUID{first.ID(), second.ID()}
				x = 2
			} else {
				g.GetCombat().AddAttacker(first.ID(), b.PlayerID())
				g.GetCombat().AddAttacker(second.ID(), a.PlayerID())
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, x); err == nil {
				t.Fatal("unpaid spell accepted")
			}
			if g.StackSize() != 0 || len(a.Hand()) != 1 || first.Tapped || second.Tapped {
				t.Fatal("failed payment changed game")
			}
			a.ManaPool().Add(core.Green, 1)
			a.ManaPool().Add(core.Colorless, x)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, x); err != nil {
				t.Fatal(err)
			}
			if first.Damage != 0 || second.Damage != 0 || first.Tapped || second.Tapped {
				t.Fatal("effect before resolution")
			}
			clone := g.Clone()
			g.ResolveStack()
			want := 1
			if name == "Winter Blast" {
				want = 2
			}
			if g.FindPermanent(first.ID()).Damage != want {
				t.Fatal("wrong first damage")
			}
			want = 1
			if name == "Winter Blast" {
				want = 0
			}
			if g.FindPermanent(second.ID()).Damage != want {
				t.Fatal("ground creature received flying-only damage")
			}
			if g.FindPermanent(other.ID()).Damage != 0 || g.FindPermanent(other.ID()).Tapped {
				t.Fatal("unselected creature affected")
			}
			if name == "Winter Blast" && (!g.FindPermanent(first.ID()).Tapped || !g.FindPermanent(second.ID()).Tapped) {
				t.Fatal("ground target must still tap")
			}
			if clone.FindPermanent(first.ID()).Damage != 0 || clone.FindPermanent(first.ID()).Tapped {
				t.Fatal("clone changed")
			}
			clone.ResolveStack()
			if clone.FindPermanent(first.ID()).Damage != g.FindPermanent(first.ID()).Damage {
				t.Fatal("clone lost pending instruction")
			}
		})
	}
}

func TestRONCollectionDamageWinterIdentity(t *testing.T) {
	for _, change := range []string{"gain-flying", "lose-flying", "return", "phase", "phase-in", "shroud", "control", "all-illegal"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "LEG", "Winter Blast")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			victim := mage.NewCreature("Flyer", "", 2, 8, mage.WithKeyword(core.Flying))
			victim.SetOwner(b.PlayerID())
			first := g.PutOnBattlefield(victim, b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Ground", "", 2, 8), b.PlayerID())
			a.ManaPool().Add(core.Green, 1)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID(), second.ID()}, 2); err != nil {
				t.Fatal(err)
			}
			current := first.ID()
			switch change {
			case "gain-flying":
				g.MutablePermanent(second.ID()).GrantBaseAttr(core.Flying)
			case "lose-flying":
				g.MutablePermanent(first.ID()).RevokeBaseAttr(core.Flying)
			case "return":
				g.BouncePermanentToHand(first)
				b.RemoveFromHand(victim.ID())
				current = g.PutOnBattlefield(victim, b.PlayerID()).ID()
			case "phase":
				g.PhaseOut(first.ID())
			case "phase-in":
				g.PhaseIn(g.PhaseOut(first.ID()))
			case "shroud":
				g.MutablePermanent(first.ID()).GrantBaseAttr(core.Shroud)
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: current, TargetID: current, ControllerID: a.PlayerID(), Duration: core.Indefinite})
			case "all-illegal":
				g.MutablePermanent(first.ID()).GrantBaseAttr(core.Shroud)
				g.MutablePermanent(second.ID()).GrantBaseAttr(core.Shroud)
			}
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{current})
			legal := change != "return" && change != "phase" && change != "shroud" && change != "all-illegal"
			damage := 0
			if legal && change != "lose-flying" {
				damage = 2
			}
			if p := g.FindPermanent(current); p.Tapped != legal || p.Damage != damage {
				t.Fatalf("first tapped=%v damage=%d", p.Tapped, p.Damage)
			}
			damage = 0
			if change == "gain-flying" {
				damage = 2
			}
			if p := g.FindPermanent(second.ID()); p.Tapped != (change != "all-illegal") || p.Damage != damage {
				t.Fatal("partial legality or current flying not respected")
			}
		})
	}
}

func TestRONCollectionDamageWinterAnnouncement(t *testing.T) {
	for _, bad := range []string{"few", "many", "duplicate", "land", "shroud", "phase"} {
		t.Run(bad, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "LEG", "Winter Blast")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 8), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 8), a.PlayerID())
			ids := []uuid.UUID{first.ID(), second.ID()}
			switch bad {
			case "few":
				ids = ids[:1]
			case "many":
				ids = append(ids, g.PutOnBattlefield(mage.NewCreature("Third", "", 2, 8), a.PlayerID()).ID())
			case "duplicate":
				ids[1] = ids[0]
			case "land":
				ids[0] = g.PutOnBattlefield(mage.NewLand("Land"), a.PlayerID()).ID()
			case "shroud":
				g.MutablePermanent(first.ID()).GrantBaseAttr(core.Shroud)
			case "phase":
				g.PhaseOut(first.ID())
			}
			a.ManaPool().Add(core.Green, 1)
			a.ManaPool().Add(core.Colorless, 2)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, 2); err == nil {
				t.Fatal("invalid targets accepted")
			}
			if a.ManaPool().TotalMana() != 3 || g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("invalid target spent resources")
			}
		})
	}
	for _, x := range []int{0, 1, 3} {
		t.Run(fmt.Sprint(x), func(t *testing.T) {
			g, a, b := ronGame()
			g.SetStep(core.PrecombatMain)
			card := repositoryCard(t, "LEG", "Winter Blast")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			ids := make([]uuid.UUID, x)
			for i := range ids {
				ids[i] = g.PutOnBattlefield(mage.NewCreature("Flyer", "", 2, 8, mage.WithKeyword(core.Flying)), b.PlayerID()).ID()
			}
			a.ManaPool().Add(core.Green, 1)
			a.ManaPool().Add(core.Colorless, x)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, x); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			for _, id := range ids {
				if p := g.FindPermanent(id); !p.Tapped || p.Damage != 2 {
					t.Fatal("X or damage amount incorrect")
				}
			}
			if a.ManaPool().TotalMana() != 0 || len(a.Graveyard()) != 1 {
				t.Fatal("casting cost or finish incorrect")
			}
		})
	}
}

func TestRONCollectionDamageSandstormMembership(t *testing.T) {
	for _, change := range []string{"leave-combat", "phase", "late-entry", "shroud", "blocked"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 8), a.PlayerID())
			first.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID()})
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 8), b.PlayerID())
			card := repositoryCard(t, "ARN", "Sandstorm")
			card.SetOwner(b.PlayerID())
			b.AddToHand(card)
			b.ManaPool().Add(core.Green, 1)
			if err := g.CastSpellByID(b.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "leave-combat":
				if err := mage.ApplyEffect(g, mage.RemovePermanentFromCombat(mage.This), first.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "phase":
				g.PhaseOut(first.ID())
			case "late-entry":
				g.PutOnBattlefieldAttacking(mage.NewCreature("Late", "", 2, 8), a.PlayerID(), b.PlayerID())
			case "shroud":
				g.MutablePermanent(first.ID()).GrantBaseAttr(core.Shroud)
			case "blocked":
				g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: other.ID(), AttackerID: first.ID()}})
			}
			g.ResolveStack()
			g.PhaseIn([]uuid.UUID{first.ID()})
			want := 1
			if change == "leave-combat" || change == "phase" {
				want = 0
			}
			if g.FindPermanent(first.ID()).Damage != want || g.FindPermanent(other.ID()).Damage != 0 {
				t.Fatal("current attacking membership or nontargeting damage failed")
			}
			if change == "late-entry" {
				for _, p := range g.FilterBattlefield(mage.IsCreature) {
					if p.Name() == "Late" && p.Damage != 1 {
						t.Fatal("later attacking entry excluded")
					}
				}
			}
		})
	}
}

func TestRONCollectionDamageInvalid(t *testing.T) {
	for _, recipient := range []string{
		`EachPermanent`, `EachPermanent()`, `EachPermanent(AnyPermanent,AnyPermanent)`,
		`EachPermanent(And([]))`, `EachPermanent(Or([]))`, `EachPermanent(Ref_TargetPermanent)`,
		`EachPermanent(Ref_TargetPlayer)`, `EachPermanent(Ref_TargetPermanents())`,
		`EachPermanent(And([Ref_TargetPermanents,Ref_TargetPermanents]))`,
		`EachPermanent(And([Ref_TargetPermanents,HasAbility(Flying),IsCardtype(Creature)]))`,
		`EachPermanent(And([Ref_TargetPermanents,HasAbility(Flying())]))`,
		`EachPermanent(Or([Ref_TargetPermanents,HasAbility(Flying)]))`,
		`EachPermanent(And([And([Ref_TargetPermanents]),HasAbility(Flying)]))`,
	} {
		t.Run(recipient, func(t *testing.T) {
			src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([NumberTargetPermanents(Integer(2),IsCardtype(Creature))],ActionList([SpellDealsDamage(ThisSpell,Integer(2),` + recipient + `)])))])`
			nodes, err := Parse(strings.NewReader(src), "invalid.ron")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Compile(nodes[0]); err == nil {
				t.Fatal("invalid collection accepted")
			}
		})
	}
	for _, body := range []string{
		`SpellActions(ActionList([SpellDealsDamage(ThisSpell,Integer(2),EachPermanent(Ref_TargetPermanents))]))`,
		`SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([SpellDealsDamage(ThisSpell,Integer(2),EachPermanent(And([Ref_TargetPermanents,HasAbility(Flying)])))])))`,
		`SpellActions(Targeted([NumberTargetPermanents(Integer(2),IsCardtype(Creature))],ActionList([CreateFutureTrigger(AtTheNextEndOfCombat,ActionList([SpellDealsDamage(ThisSpell,Integer(2),EachPermanent(Ref_TargetPermanents))]))])))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+body+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("unbound collection accepted")
		}
	}
}

func TestRONCollectionDamageInstancesCopyAndInventory(t *testing.T) {
	files := fstest.MapFS{}
	for _, tc := range []struct{ set, name string }{{"ARN", "Sandstorm"}, {"LEG", "Winter Blast"}} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.HasPrefix(line, `Card(Name: "`+tc.name+`",`) {
				files[tc.set+".ron"] = &fstest.MapFile{Data: []byte(line)}
				break
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("full records not loaded")
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 2 || len(report.Issues) != 0 {
		t.Fatal("inventory lost complete records")
	}
	for name, role := range map[string]string{"EachPermanent": "recipient", "Ref_TargetPermanents": "permanent_collection_reference", "HasAbility": "permanent_filter", "Flying": "keyword", "IsAttacking": "permanent_filter"} {
		if !slices.ContainsFunc(report.Signatures, func(s InventorySignature) bool { return s.Name == name && s.Role == role }) {
			t.Fatalf("missing %s/%s", name, role)
		}
	}
	a := &pluralColorPlayer{BasePlayer: mage.NewBasePlayer("A")}
	b := &pluralColorPlayer{BasePlayer: mage.NewBasePlayer("B")}
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	ids := make([]uuid.UUID, 4)
	for i := range ids {
		ids[i] = g.PutOnBattlefield(mage.NewCreature("Flyer", "", 2, 10, mage.WithKeyword(core.Flying)), b.PlayerID()).ID()
	}
	first, err := registry.CreateCard("Winter Blast")
	if err != nil {
		t.Fatal(err)
	}
	second, err := registry.CreateCard("Winter Blast")
	if err != nil {
		t.Fatal(err)
	}
	if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
		t.Fatal("shared target declarations")
	}
	for i, card := range []mage.Card{first, second} {
		card.SetOwner(a.PlayerID())
		a.AddToHand(card)
		a.ManaPool().Add(core.Green, 1)
		a.ManaPool().Add(core.Colorless, 2)
		if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids[i:i+2], 2); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			g.ResolveStack()
		}
	}
	b.targets = ids[2:4]
	copied := g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), true)
	if copied == nil || !slices.Equal(copied.Targets, b.targets) {
		t.Fatal("copy failed to retain X or retarget")
	}
	clone := g.Clone()
	g.ResolveTopOfStack()
	for _, id := range ids {
		p := g.FindPermanent(id)
		if p.Damage != 2 || !p.Tapped {
			t.Fatal("copy changed original binding")
		}
	}
	g.ResolveStack()
	want := []int{2, 4, 4, 2}
	for i, id := range ids {
		if g.FindPermanent(id).Damage != want[i] {
			t.Fatalf("instances lost independent collections: target %d damage %d want %d", i, g.FindPermanent(id).Damage, want[i])
		}
	}
	for i, id := range ids {
		want := 0
		if i < 2 {
			want = 2
		}
		if clone.FindPermanent(id).Damage != want {
			t.Fatal("copy resolution changed clone")
		}
	}
	clone.ResolveStack()
	for i, id := range ids {
		if clone.FindPermanent(id).Damage != want[i] {
			t.Fatal("clone lost copied/original frame")
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Gap",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}
	if result, err := LoadFS(files, "*.ron"); err == nil || result != nil {
		t.Fatal("invalid collection published partial registry")
	}
}

func TestRONCollectionDamageSimultaneousAndPrevention(t *testing.T) {
	for _, name := range []string{"Sandstorm", "Winter Blast"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			g.SetStep(core.PrecombatMain)
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 8, mage.WithKeyword(core.Flying)), a.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 8, mage.WithKeyword(core.Flying)), a.PlayerID())
			set := "LEG"
			amount := 2
			x := 2
			ids := []uuid.UUID{first.ID(), second.ID()}
			if name == "Sandstorm" {
				set = "ARN"
				amount = 1
				x = 0
				ids = nil
				first.RevokeBaseAttr(core.AttrSummonSick)
				second.RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{first.ID(), second.ID()})
			}
			card := repositoryCard(t, set, name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Green, 1)
			a.ManaPool().Add(core.Colorless, x)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), ids, x); err != nil {
				t.Fatal(err)
			}
			g.AddPreventionShield(first.ID(), 1)
			notifications := 0
			g.SetOnDamageDealt(func(source, target string, damage int, combat bool) {
				notifications++
				if source != name || combat || g.FindPermanent(first.ID()).Damage != amount-1 || g.FindPermanent(second.ID()).Damage != amount {
					t.Error("notification saw incomplete batch or wrong source")
				}
			})
			g.ResolveStack()
			want := 2
			if amount == 1 {
				want = 1
			}
			if notifications != want || a.Life() != 20 || b.Life() != 20 {
				t.Fatal("prevention or recipient categories incorrect")
			}
		})
	}
}

func TestRONCollectionDamageSourceIdentity(t *testing.T) {
	for _, change := range []string{"departed", "returned", "phased", "control", "copy", "prevented"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := permanentScopeFactory(t, "Artifact", `Lifelink,Activated(TapPermanent(ThisPermanent),ActionList([PermanentDealsDamage(ThisPermanent,Integer(2),EachPermanent(IsCardtype(Creature)))]))`)()
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			first := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 10), b.PlayerID())
			second := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 10), b.PlayerID())
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "departed", "returned":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "returned" {
					g.PutOnBattlefield(card, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(source.ID())
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			case "prevented":
				g.AddDamagePreventionRule(mage.WithFrom(mage.IsArtifact), mage.WithTo(mage.IsCreature))
			}
			clone := g.Clone()
			g.ResolveStack()
			amount := 2
			if change == "copy" {
				amount = 4
			}
			if change == "prevented" {
				amount = 0
			}
			wantA, wantB := 20+2*amount, 20
			if change == "control" {
				wantA, wantB = 20, 24
			}
			if g.FindPermanent(first.ID()).Damage != amount || g.FindPermanent(second.ID()).Damage != amount || a.Life() != wantA || b.Life() != wantB {
				t.Fatal("source LKI, lifelink or prevention failed")
			}
			if clone.FindPermanent(first.ID()).Damage != 0 || clone.GetPlayer(a.PlayerID()).Life() != 20 {
				t.Fatal("pending clone changed")
			}
			clone.ResolveStack()
			if clone.FindPermanent(first.ID()).Damage != amount || clone.GetPlayer(a.PlayerID()).Life() != wantA || clone.GetPlayer(b.PlayerID()).Life() != wantB {
				t.Fatal("clone lost original source")
			}
		})
	}
}

func TestRONCollectionDamageSupportedShapes(t *testing.T) {
	for _, tc := range []struct {
		recipient string
		ground    bool
	}{
		{`EachPermanent(Ref_TargetPermanents)`, true},
		{`EachPermanent(And([HasAbility(Flying),Ref_TargetPermanents]))`, false},
		{`EachPermanent(And([Ref_TargetPermanents,Or([HasAbility(Flying),IsTapped])]))`, false},
		{`EachPermanent(Or([HasAbility(Flying),And([IsTapped,IsCardtype(Creature)])]))`, false},
	} {
		for _, amount := range []int{-1, 0, 2} {
			t.Run(fmt.Sprintf("%s/%d", tc.recipient, amount), func(t *testing.T) {
				g, a, b := ronGame()
				card := permanentScopeFactory(t, "Instant", fmt.Sprintf(`SpellActions(Targeted([NumberTargetPermanents(Integer(2),IsCardtype(Creature))],ActionList([SpellDealsDamage(ThisSpell,Integer(%d),%s)])))`, amount, tc.recipient))()
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				first := g.PutOnBattlefield(mage.NewCreature("Flyer", "", 2, 8, mage.WithKeyword(core.Flying)), b.PlayerID())
				second := g.PutOnBattlefield(mage.NewCreature("Ground", "", 2, 8), b.PlayerID())
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{first.ID(), second.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				want := max(0, amount)
				if first.Damage != want {
					t.Fatal("fixed amount or flying member lost")
				}
				if !tc.ground {
					want = 0
				}
				if second.Damage != want {
					t.Fatal("collection filter ignored")
				}
			})
		}
	}
}

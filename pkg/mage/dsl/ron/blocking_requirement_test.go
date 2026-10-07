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

func TestRONBlockingRequirementCards(t *testing.T) {
	for _, name := range []string{"Lure", "Marble Priest"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			set := "LEA"
			if name == "Marble Priest" {
				set = "LEG"
			}
			card := repositoryCard(t, set, name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 3, 8), a.PlayerID())
			var targets []uuid.UUID
			if name == "Lure" {
				targets = []uuid.UUID{host.ID()}
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err == nil || g.StackSize() != 0 || len(a.Hand()) != 1 {
				t.Fatal("unpaid cast changed state")
			}
			a.ManaPool().Add(core.Green, 2)
			a.ManaPool().Add(core.Colorless, 5)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if name == "Marble Priest" {
				host = g.FindPermanent(card.ID())
			}
			wall := g.PutOnBattlefield(mage.NewCreature("Wall", "", 4, 8, mage.WithSubTypes("Wall")), b.PlayerID())
			plain := g.PutOnBattlefield(mage.NewCreature("Plain", "", 2, 8), b.PlayerID())
			host.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{host.ID()})
			g.ExecuteBlockers(nil)
			if !g.GetCombat().IsBlocking(wall.ID()) || g.GetCombat().IsBlocking(plain.ID()) != (name == "Lure") {
				t.Fatal("filtered required blockers")
			}
			clone := g.Clone()
			g.ExecuteCombatDamage()
			clone.ExecuteCombatDamage()
			for _, game := range []*mage.Game{g, clone} {
				want := 6
				if name == "Marble Priest" {
					want = 0
				}
				if game.FindPermanent(host.ID()).Damage != want || game.GetPlayer(b.PlayerID()).Life() != 20 {
					t.Fatal("complete card combat rules")
				}
			}
		})
	}
}

func TestRONBlockingRequirementInvalid(t *testing.T) {
	for _, rule := range []string{
		`PermanentRuleEffect(ThisPermanent,[AllCreaturesMustBlockIt])`,
		`PermanentRuleEffect(ThisPermanent,[AllCreaturesMustBlockIt()])`,
		`PermanentRuleEffect(ThisPermanent,[AllCreaturesMustBlockIt(IsCardtype(Creature),AnyPermanent)])`,
		`PermanentRuleEffect(Ref_TargetPermanent,[AllCreaturesMustBlockIt(IsCardtype(Creature))])`,
		`PermanentRuleEffect(HostPermanent,[AllCreaturesMustBlockIt(IsCardtype(Creature))])`,
		`PermanentRuleEffect(ThisPermanent,[AllCreaturesMustBlockIt(Ref_TargetPlayer)])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONBlockingRequirementLureLifetime(t *testing.T) {
	for _, change := range []string{"live", "control", "source-left", "source-return", "source-phased", "phase-cycle", "source-abilities", "host-abilities", "host-return", "reattach", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEA", "Lure")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			hostCard := mage.NewCreature("Host", "", 3, 8)
			hostCard.SetOwner(a.PlayerID())
			host := g.PutOnBattlefield(hostCard, a.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 8), a.PlayerID())
			blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 8), b.PlayerID())
			g.Attach(source.ID(), host.ID())
			switch change {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "source-left", "source-return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "source-return" {
					source = g.PutOnBattlefield(card, a.PlayerID())
					g.Attach(source.ID(), other.ID())
				}
			case "source-phased":
				g.PhaseOut(source.ID())
			case "phase-cycle":
				g.PhaseIn(g.PhaseOut(host.ID()))
			case "source-abilities":
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
			case "host-abilities":
				g.AddContinuousEffect(mage.RemoveAllAbilities(host.ID(), core.Indefinite))
			case "host-return":
				g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
				host = g.PutOnBattlefield(hostCard, a.PlayerID())
			case "reattach":
				g.Attach(source.ID(), other.ID())
			}
			host = g.FindPermanent(host.ID())
			host.RevokeBaseAttr(core.AttrSummonSick)
			other.RevokeBaseAttr(core.AttrSummonSick)
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{host.ID(), other.ID()})
			if change == "clone" {
				original := g
				g = g.Clone()
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if len(original.CompleteBlockAssignments(b.PlayerID(), nil)) != 1 {
					t.Fatal("clone removal changed original")
				}
			}
			result := g.CompleteBlockAssignments(b.PlayerID(), nil)
			expect := change == "live" || change == "control" || change == "phase-cycle" || change == "host-abilities" || change == "reattach" || change == "source-return"
			if (len(result) == 1) != expect {
				t.Fatalf("requirement live=%v, assignments=%v", expect, result)
			}
			if expect {
				target := host.ID()
				if change == "reattach" || change == "source-return" {
					target = other.ID()
				}
				if result[0].BlockerID != blocker.ID() || result[0].AttackerID != target {
					t.Fatal("stale source or host identity")
				}
			}
		})
	}
}

func TestRONBlockingRequirementLureAnnouncement(t *testing.T) {
	for _, change := range []string{"land", "shroud", "departed", "returned", "phased", "copy"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := mage.NewCreature("Host", "", 2, 8)
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			card := repositoryCard(t, "LEA", "Lure")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Green, 3)
			target := host.ID()
			if change == "land" {
				target = g.PutOnBattlefield(mage.NewLand("Land"), b.PlayerID()).ID()
			}
			if change == "shroud" {
				host.GrantBaseAttr(core.Shroud)
			}
			err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0)
			if change == "land" || change == "shroud" {
				if err == nil || a.ManaPool().TotalMana() != 3 || len(a.Hand()) != 1 || g.StackSize() != 0 {
					t.Fatal("invalid aura target did not roll back")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "departed", "returned":
				g.RemoveFromBattlefield(host)
				if change == "returned" {
					g.PutOnBattlefield(hostCard, b.PlayerID())
				}
			case "phased":
				g.PhaseOut(host.ID())
			case "copy":
				if g.CopySpellOnStack(card.ID(), b.PlayerID(), false) == nil {
					t.Fatal("copy failed")
				}
			}
			clone := g.Clone()
			g.ResolveStack()
			clone.ResolveStack()
			for _, game := range []*mage.Game{g, clone} {
				if change == "copy" {
					var count int
					for _, p := range game.AllBattlefield() {
						if p.Card.Name() == "Lure" && p.AttachedTo == host.ID() {
							count++
						}
					}
					if count != 2 {
						t.Fatal("copied aura declarations")
					}
				} else if game.FindPermanent(card.ID()) != nil {
					t.Fatal("aura resolved on illegal incarnation")
				}
			}
		})
	}
}

func TestRONBlockingRequirementStrictRecordsAndInventory(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "3ED", "CED", "CEI", "LEG"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if !strings.Contains(line, `Name: "Lure"`) && !strings.Contains(line, `Name: "Marble Priest"`) {
				continue
			}
			nodes, err := Parse(strings.NewReader(line), set+".ron")
			if err != nil || len(nodes) != 1 {
				t.Fatal(err)
			}
			files[set+".ron"] = &fstest.MapFile{Data: []byte(line)}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(files) != 7 || len(registry.RegisteredCardNames()) != 2 {
		t.Fatalf("strict reprints: %v", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil || report.Compiled != 7 || report.UniqueCards != 2 {
		t.Fatalf("inventory: %v", err)
	}
	found := false
	for _, signature := range report.Signatures {
		if signature.Name == "IsCreatureType" && signature.Role == "permanent_filter" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing requirement filter role")
	}
	first, _ := registry.CreateCard("Lure")
	second, _ := registry.CreateCard("Lure")
	g, a, b := ronGame()
	p := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 8), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 8), a.PlayerID())
	firstSource := g.PutOnBattlefield(first, a.PlayerID())
	secondSource := g.PutOnBattlefield(second, b.PlayerID())
	g.Attach(firstSource.ID(), p.ID())
	g.Attach(secondSource.ID(), q.ID())
	blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 8), b.PlayerID())
	p.RevokeBaseAttr(core.AttrSummonSick)
	q.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{p.ID(), q.ID()})
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(secondSource.ID()))
	result := g.CompleteBlockAssignments(b.PlayerID(), []mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: q.ID()}})
	if len(result) != 1 || result[0].AttackerID != p.ID() {
		t.Fatal("factory instances shared requirements")
	}
	result = clone.CompleteBlockAssignments(b.PlayerID(), []mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: q.ID()}})
	if len(result) != 1 || result[0].AttackerID != q.ID() {
		t.Fatal("clone or competing proposal isolation")
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[Unknown])`)}
	if LoadInto(registry, files, "*.ron") == nil || len(registry.RegisteredCardNames()) != 2 {
		t.Fatal("failed collection changed registry")
	}
}

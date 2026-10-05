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

func TestRONFilteredBlockingRealCards(t *testing.T) {
	for _, entry := range []struct{ set, name string }{
		{"LEA", "Invisibility"}, {"LEB", "Invisibility"}, {"2ED", "Invisibility"}, {"CED", "Invisibility"}, {"CEI", "Invisibility"}, {"LEG", "Seeker"}, {"LEG", "Elven Riders"},
	} {
		t.Run(entry.set+"/"+entry.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, entry.set, entry.name)
			if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(card, a.PlayerID())
			attacker := source
			if entry.name != "Elven Riders" {
				attacker = g.PutOnBattlefield(mage.NewCreature("Host", "", 3, 3), a.PlayerID())
				g.Attach(source.ID(), attacker.ID())
			}
			for _, kind := range []string{"plain", "wall", "flying", "reach", "white", "artifact", "white-land"} {
				c := mage.NewCreature(kind, "", 2, 4)
				switch kind {
				case "wall":
					c.AddSubType("Wall")
				case "flying":
					c = mage.NewCreature(kind, "", 2, 4, mage.WithKeyword(core.Flying))
				case "reach":
					c = mage.NewCreature(kind, "", 2, 4, mage.WithKeyword(core.Reach))
				case "white":
					c.SetColorOverride([]core.Color{core.White})
				case "artifact":
					c.AddType(core.TypeArtifact)
				case "white-land":
					c = mage.NewLand(kind)
					c.SetColorOverride([]core.Color{core.White})
				}
				blocker := g.PutOnBattlefield(c, b.PlayerID())
				want := kind == "wall"
				if entry.name == "Seeker" {
					want = kind == "white" || kind == "artifact"
				}
				if entry.name == "Elven Riders" {
					want = kind == "wall" || kind == "flying"
				}
				if mage.CanBlock(blocker, g.FindPermanent(attacker.ID()), g) != want {
					t.Fatalf("%s: filter", kind)
				}
			}
		})
	}
}

func TestRONFilteredBlockingInvalid(t *testing.T) {
	for _, tc := range []struct{ types, subtypes, subject, operation string }{
		{"Artifact", "", "HostPermanent", "CantBeBlockedExceptByDefenders(AnyPermanent)"},
		{"Enchantment", "Aura", "HostPermanent()", "CantBeBlockedExceptByDefenders(AnyPermanent)"},
		{"Artifact", "", "Ref_TargetPermanent", "CantBeBlockedExceptByDefenders(AnyPermanent)"},
		{"Artifact", "", "ThisPermanent", "CantBeBlockedExceptByDefenders"},
		{"Artifact", "", "ThisPermanent", "CantBeBlockedExceptByDefenders()"},
		{"Artifact", "", "ThisPermanent", "CantBeBlockedExceptByDefenders(AnyPermanent,AnyPermanent)"},
		{"Artifact", "", "ThisPermanent", "CantBeBlockedExceptByDefenders(And(AnyPermanent))"},
		{"Artifact", "", "ThisPermanent", "CantBeBlockedExceptByDefenders(IsCreatureType(Wall)),CantAttack"},
		{"Instant", "", "ThisPermanent", "CantBeBlockedExceptByDefenders(AnyPermanent)"},
	} {
		src := fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[%s],Subtypes:[%s]),Rules:[PermanentRuleEffect(%s,[%s])])`, tc.types, tc.subtypes, tc.subject, tc.operation)
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", src)
		}
	}
}

func TestRONFilteredBlockingLifecycle(t *testing.T) {
	for _, name := range []string{"Invisibility", "Seeker", "Elven Riders"} {
		for _, change := range []string{"none", "source-phase", "phase-cycle", "control", "source-left", "source-return", "source-removed", "recipient-removed", "reattach", "host-return"} {
			if name == "Elven Riders" && (change == "reattach" || change == "host-return" || change == "source-left" || change == "source-return" || change == "source-phase") {
				continue
			}
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				set := "LEG"
				if name == "Invisibility" {
					set = "LEA"
				}
				card := repositoryCard(t, set, name)
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				hostCard := mage.NewCreature("Host", "", 3, 3)
				hostCard.SetOwner(a.PlayerID())
				host := source
				if name != "Elven Riders" {
					host = g.PutOnBattlefield(hostCard, a.PlayerID())
					g.Attach(source.ID(), host.ID())
				}
				blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 2, 2), b.PlayerID())
				other := g.PutOnBattlefield(mage.NewCreature("Other", "", 3, 3), a.PlayerID())
				switch change {
				case "source-phase":
					g.PhaseOut(source.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(host.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "source-left", "source-return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if change == "source-return" {
						source = g.PutOnBattlefield(card, a.PlayerID())
						g.Attach(source.ID(), other.ID())
					}
				case "source-removed":
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
				case "recipient-removed":
					g.AddContinuousEffect(mage.RemoveAllAbilities(host.ID(), core.Indefinite))
				case "reattach":
					g.Attach(source.ID(), other.ID())
				case "host-return":
					g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
					host = g.PutOnBattlefield(hostCard, a.PlayerID())
				}
				g.ApplyContinuousEffects()
				allowed := change == "source-phase" || change == "source-left" || change == "source-return" || change == "source-removed" || change == "reattach" || change == "host-return" || name == "Elven Riders" && change == "recipient-removed"
				if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(host.ID()), g) != allowed {
					t.Fatal("source lifetime or host membership")
				}
				clone := g.Clone()
				if name != "Elven Riders" && clone.FindPermanent(source.ID()) != nil {
					clone.Attach(source.ID(), other.ID())
				}
				clone.ApplyContinuousEffects()
				if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(host.ID()), g) != allowed {
					t.Fatal("clone changed attachment or rule")
				}
				if change == "reattach" || change == "source-return" {
					if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(other.ID()), g) {
						t.Fatal("new host did not receive the restriction")
					}
				}
			})
		}
	}
}

func TestRONFilteredBlockingCurrentFiltersAndIntersection(t *testing.T) {
	g, a, b := ronGame()
	seeker := g.PutOnBattlefield(repositoryCard(t, "LEG", "Seeker"), a.PlayerID())
	invisibility := g.PutOnBattlefield(repositoryCard(t, "LEA", "Invisibility"), a.PlayerID())
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 3, 3), a.PlayerID())
	blocker := g.PutOnBattlefield(mage.NewCreature("Wall", "", 0, 4, mage.WithSubTypes("Wall")), b.PlayerID())
	g.Attach(seeker.ID(), host.ID())
	g.Attach(invisibility.ID(), host.ID())
	if mage.CanBlock(blocker, host, g) {
		t.Fatal("all restrictions must pass")
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.SetColors(mage.This, core.White)), blocker.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(host.ID()), g) {
		t.Fatal("filter did not read current color")
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Flying)), host.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(host.ID()), g) {
		t.Fatal("filter bypassed flying")
	}
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.GrantKeywordModification(mage.This, core.Reach)), blocker.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(host.ID()), g) {
		t.Fatal("legal blocker rejected")
	}
	g.RunStepWithPriority(core.Cleanup)
	if mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(host.ID()), g) {
		t.Fatal("filter cached temporary white color")
	}
}

func TestRONFilteredBlockingNativeCombat(t *testing.T) {
	for _, entry := range []struct{ set, name string }{{"LEA", "Invisibility"}, {"LEG", "Seeker"}, {"LEG", "Elven Riders"}} {
		t.Run(entry.name, func(t *testing.T) {
			g, a, b := ronGame()
			c := repositoryCard(t, entry.set, entry.name)
			c.SetOwner(a.PlayerID())
			var source, host *mage.Permanent
			if entry.name == "Elven Riders" {
				source = g.PutOnBattlefield(c, a.PlayerID())
				host = source
			} else {
				host = g.PutOnBattlefield(mage.NewCreature("Host", "", 3, 3), a.PlayerID())
				a.AddToHand(c)
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
					t.Fatal("unpaid aura accepted")
				}
				if g.StackSize() != 0 || len(a.Hand()) != 1 || g.FindPermanent(c.ID()) != nil {
					t.Fatal("failed cost changed state")
				}
				if entry.name == "Seeker" {
					a.ManaPool().Add(core.White, 4)
				} else {
					a.ManaPool().Add(core.Blue, 2)
				}
				invalid := g.PutOnBattlefield(mage.NewLand("Invalid"), b.PlayerID())
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{invalid.ID()}, 0); err == nil {
					t.Fatal("noncreature enchanted")
				}
				if a.ManaPool().TotalMana() != map[string]int{"Seeker": 4, "Invisibility": 2}[entry.name] || g.StackSize() != 0 {
					t.Fatal("invalid target consumed cost")
				}
				if err := g.CastSpellByID(a.PlayerID(), c.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				source = g.FindPermanent(c.ID())
				if source == nil || source.AttachedTo != host.ID() {
					t.Fatal("aura casting")
				}
			}
			host = g.FindPermanent(host.ID())
			host.RevokeBaseAttr(core.AttrSummonSick)
			legalCard := mage.NewCreature("Legal", "", 0, 6, mage.WithSubTypes("Wall"))
			if entry.name == "Seeker" {
				legalCard.SetColorOverride([]core.Color{core.White})
			}
			legal := g.PutOnBattlefield(legalCard, b.PlayerID())
			illegal := g.PutOnBattlefield(mage.NewCreature("Illegal", "", 2, 4), b.PlayerID())
			g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{host.ID()})
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: illegal.ID(), AttackerID: host.ID()}})
			if g.GetCombat().IsBlocking(illegal.ID()) {
				t.Fatal("illegal declaration accepted")
			}
			g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: legal.ID(), AttackerID: host.ID()}})
			if !g.GetCombat().IsBlocking(legal.ID()) {
				t.Fatal("legal declaration rejected")
			}
			if entry.name != "Elven Riders" {
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
			} else {
				g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
			}
			if !g.GetCombat().IsBlocking(legal.ID()) {
				t.Fatal("restriction change removed an existing block")
			}
			g.ExecuteCombatDamage()
			if b.Life() != 20 || g.FindPermanent(legal.ID()).Damage != 3 {
				t.Fatal("combat did not use the declared block")
			}
		})
	}
}

func TestRONFilteredBlockingInventory(t *testing.T) {
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:2),Rules:[PermanentRuleEffect(ThisPermanent,[CantBeBlockedExceptByDefenders(Or([IsCreatureType(Wall),And([IsCardtype(Creature),HasAbility(Flying)])]))])])`
	report, err := InspectFS(fstest.MapFS{"test.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v", err)
	}
	for _, signature := range report.Signatures {
		switch signature.Name {
		case "Or", "And", "IsCreatureType", "IsCardtype", "HasAbility":
			if signature.Role != "permanent_filter" {
				t.Fatalf("%s: %s", signature.Name, signature.Role)
			}
		case "Flying":
			if signature.Role != "keyword" {
				t.Fatal("keyword role")
			}
		}
	}
}

func TestRONFilteredBlockingInstancesAndStrictLoad(t *testing.T) {
	files := fstest.MapFS{}
	for _, set := range []string{"LEA", "LEB", "2ED", "CED", "CEI", "LEG"} {
		data, err := os.ReadFile("../../../../cards/mtgish/" + set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		nodes, err := Parse(strings.NewReader(string(data)), set+".ron")
		if err != nil {
			t.Fatal(err)
		}
		for _, node := range nodes {
			name := node.Fields["Name"].String
			if name != "Invisibility" && name != "Seeker" && name != "Elven Riders" {
				continue
			}
			for line := range strings.SplitSeq(string(data), "\n") {
				if strings.Contains(line, `Name: "`+name+`"`) {
					files[set+name+".ron"] = &fstest.MapFile{Data: []byte(line)}
				}
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil || len(files) != 7 || len(registry.RegisteredCardNames()) != 3 {
		t.Fatalf("strict records: %v", err)
	}
	data, err := os.ReadFile("../../../../cards/mtgish/LEA.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEA.ron")
	if err != nil {
		t.Fatal(err)
	}
	var factory mage.CardFactory
	for _, node := range nodes {
		if node.Fields["Name"].String == "Invisibility" {
			factory, err = Compile(node)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if factory == nil {
		t.Fatal("missing factory")
	}
	g, a, b := ronGame()
	first := g.PutOnBattlefield(factory(), a.PlayerID())
	second := g.PutOnBattlefield(factory(), b.PlayerID())
	p := g.PutOnBattlefield(mage.NewCreature("First", "", 2, 2), a.PlayerID())
	q := g.PutOnBattlefield(mage.NewCreature("Second", "", 2, 2), b.PlayerID())
	blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 2, 2), b.PlayerID())
	g.Attach(first.ID(), p.ID())
	g.Attach(second.ID(), q.ID())
	clone := g.Clone()
	g.RemoveFromBattlefield(g.FindPermanent(first.ID()))
	if !mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(p.ID()), g) || mage.CanBlock(g.FindPermanent(blocker.ID()), g.FindPermanent(q.ID()), g) {
		t.Fatal("instances shared source state")
	}
	if mage.CanBlock(clone.FindPermanent(blocker.ID()), clone.FindPermanent(p.ID()), clone) {
		t.Fatal("source removal changed clone")
	}
}

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

func controlHost(name string) mage.Card {
	if name == "Steal Artifact" {
		return mage.NewArtifact("Host", "")
	}
	return mage.NewCreature("Host", "", 2, 3)
}

func TestRONStaticControlLifecycle(t *testing.T) {
	for _, name := range []string{"Control Magic", "Steal Artifact"} {
		for _, scenario := range []string{"source-control", "source-left", "reattach", "phase-source", "phase-host", "host-return", "source-return", "competing", "cant-change", "host-abilities", "source-abilities"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				hostCard := controlHost(name)
				hostCard.SetOwner(b.PlayerID())
				host := g.PutOnBattlefield(hostCard, b.PlayerID())
				other := g.PutOnBattlefield(controlHost(name), b.PlayerID())
				card := repositoryCard(t, "LEB", name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				source := g.PutOnBattlefield(card, a.PlayerID())
				if scenario == "cant-change" {
					host.GrantBaseAttr(core.AttrCantChangeControl)
				}
				g.Attach(source.ID(), host.ID())
				want := a.PlayerID()
				if scenario == "cant-change" {
					want = b.PlayerID()
				}
				if g.FindPermanent(host.ID()).ControllerID() != want || host.Card.Owner() != b.PlayerID() {
					t.Fatal("initial control or ownership")
				}
				if name == "Control Magic" && !host.HasAttr(core.AttrSummonSick) {
					t.Fatal("control change did not cause summoning sickness")
				}
				clone := g.Clone()
				switch scenario {
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: source.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
					want = b.PlayerID()
				case "source-left":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					want = b.PlayerID()
				case "reattach":
					g.Attach(source.ID(), other.ID())
					want = b.PlayerID()
					if g.FindPermanent(other.ID()).ControllerID() != a.PlayerID() {
						t.Fatal("new attachment was not controlled")
					}
				case "phase-source":
					ids := g.PhaseOut(source.ID())
					g.ApplyContinuousEffects()
					if g.FindPermanent(host.ID()).ControllerID() != b.PlayerID() {
						t.Fatal("phased Aura still controls host")
					}
					g.PhaseIn(ids)
					g.ApplyContinuousEffects()
				case "phase-host":
					ids := g.PhaseOut(host.ID())
					if g.FindPermanent(host.ID()) != nil || g.FindPermanent(source.ID()) != nil {
						t.Fatal("host did not phase with Aura")
					}
					g.PhaseIn(ids)
					g.ApplyContinuousEffects()
				case "host-return":
					g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
					host = g.PutOnBattlefield(hostCard, b.PlayerID())
					want = b.PlayerID()
				case "source-return":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					source = g.PutOnBattlefield(card, a.PlayerID())
					g.Attach(source.ID(), other.ID())
					want = b.PlayerID()
					if g.FindPermanent(other.ID()).ControllerID() != a.PlayerID() {
						t.Fatal("returned Aura did not control new attachment")
					}
				case "competing":
					rivalCard := repositoryCard(t, "LEB", name)
					rivalCard.SetOwner(b.PlayerID())
					rival := g.PutOnBattlefield(rivalCard, b.PlayerID())
					g.Attach(rival.ID(), host.ID())
					if g.FindPermanent(host.ID()).ControllerID() != b.PlayerID() {
						t.Fatal("later attachment timestamp lost")
					}
					g.RemoveFromBattlefield(g.FindPermanent(rival.ID()))
				case "source-abilities":
					g.AddContinuousEffect(mage.RemoveAllAbilities(source.ID(), core.Indefinite))
					g.ApplyContinuousEffects()
				case "host-abilities":
					g.AddContinuousEffect(mage.RemoveAllAbilities(host.ID(), core.Indefinite))
				}
				if g.FindPermanent(host.ID()).ControllerID() != want {
					t.Fatalf("controller=%s want=%s", g.FindPermanent(host.ID()).ControllerID(), want)
				}
				if clone.FindPermanent(hostCard.ID()).ControllerID() != func() uuid.UUID {
					if scenario == "cant-change" {
						return b.PlayerID()
					}
					return a.PlayerID()
				}() {
					t.Fatal("clone control state changed")
				}
			})
		}
	}
}

func TestRONStaticControlNativeCasting(t *testing.T) {
	for _, name := range []string{"Control Magic", "Steal Artifact"} {
		for _, scenario := range []string{"legal", "unpaid", "wrong-type", "shroud", "departed", "phased", "lost-type"} {
			t.Run(name+"/"+scenario, func(t *testing.T) {
				g, a, b := ronGame()
				hostCard := controlHost(name)
				if scenario == "wrong-type" {
					hostCard = mage.NewLand("Wrong")
				}
				if scenario == "shroud" {
					hostCard = mage.NewCreature("Host", "", 2, 3, mage.WithKeyword(core.Shroud))
					if name == "Steal Artifact" {
						hostCard = mage.NewArtifact("Host", "", mage.WithKeyword(core.Shroud))
					}
				}
				hostCard.SetOwner(b.PlayerID())
				host := g.PutOnBattlefield(hostCard, b.PlayerID())
				card := repositoryCard(t, "LEB", name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				a.ManaPool().Add(core.Blue, 2)
				if scenario != "unpaid" {
					a.ManaPool().Add(core.Colorless, 2)
				}
				before := a.ManaPool().TotalMana()
				err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0)
				if scenario == "unpaid" || scenario == "wrong-type" || scenario == "shroud" {
					if err == nil {
						t.Fatal("illegal cast accepted")
					}
					if a.ManaPool().TotalMana() != before || g.StackSize() != 0 || len(a.Hand()) != 1 || g.FindPermanent(host.ID()).ControllerID() != b.PlayerID() {
						t.Fatal("failed cast changed state")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if g.StackSize() != 1 || a.ManaPool().TotalMana() != 0 || g.FindPermanent(host.ID()).ControllerID() != b.PlayerID() {
					t.Fatal("control changed before resolution or incorrect payment")
				}
				switch scenario {
				case "departed":
					g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
				case "phased":
					g.PhaseOut(host.ID())
				case "lost-type":
					attr := core.AttrIsCreature
					if name == "Steal Artifact" {
						attr = core.AttrIsArtifact
					}
					g.MutablePermanent(host.ID()).RevokeBaseAttr(attr)
				}
				clone := g.Clone()
				clone.ResolveStack()
				if g.StackSize() != 1 || g.FindPermanent(card.ID()) != nil {
					t.Fatal("clone resolution changed original")
				}
				g.ResolveStack()
				if scenario != "legal" {
					if g.FindPermanent(card.ID()) != nil || len(a.Graveyard()) != 1 {
						t.Fatal("illegal target did not stop Aura entry")
					}
					return
				}
				if g.FindPermanent(card.ID()).AttachedTo != host.ID() || g.FindPermanent(host.ID()).ControllerID() != a.PlayerID() {
					t.Fatal("Aura did not attach and take control")
				}
				if name == "Control Magic" && g.FindPermanent(host.ID()).CanDeclareAsAttacker(g) {
					t.Fatal("newly controlled creature attacked without haste")
				}
				g.BouncePermanentToHand(g.FindPermanent(card.ID()))
				if g.FindPermanent(host.ID()).ControllerID() != b.PlayerID() {
					t.Fatal("control did not revert after Aura departure")
				}
			})
		}
	}
}

func TestRONStaticControlInvalid(t *testing.T) {
	for _, rule := range []string{
		`PermanentLayerEffect(HostPermanent, [SetController()])`,
		`PermanentLayerEffect(HostPermanent, [SetController(You, You)])`,
		`PermanentLayerEffect(HostPermanent, [SetController(You())])`,
		`PermanentLayerEffect(HostPermanent, [SetController(Ref_TargetPlayer)])`,
		`PermanentLayerEffect(ThisPermanent, [SetController(You)])`,
		`EachPermanentLayerEffect(AnyPermanent, [SetController(You)])`,
		`Activated(PayMana([]), ActionList([SetController(You)]))`,
		`PermanentLayerEffect(HostPermanent, [SetController(You), UnknownModification])`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(AnyPermanent),%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[PermanentLayerEffect(HostPermanent,[SetController(You)])])`), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Compile(nodes[0]); err == nil {
		t.Fatal("non-Aura accepted")
	}
}

func TestRONStaticControlFactories(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEB.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEB.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes {
		name := node.Fields["Name"].String
		if name != "Control Magic" && name != "Steal Artifact" {
			continue
		}
		factory, err := Compile(node)
		if err != nil {
			t.Fatal(err)
		}
		g, a, b := ronGame()
		first := g.PutOnBattlefield(controlHost(name), b.PlayerID())
		second := g.PutOnBattlefield(controlHost(name), a.PlayerID())
		c1, c2 := factory(), factory()
		c1.SetOwner(a.PlayerID())
		c2.SetOwner(b.PlayerID())
		p1 := g.PutOnBattlefield(c1, a.PlayerID())
		p2 := g.PutOnBattlefield(c2, b.PlayerID())
		g.Attach(p1.ID(), first.ID())
		g.Attach(p2.ID(), second.ID())
		if g.FindPermanent(first.ID()).ControllerID() != a.PlayerID() || g.FindPermanent(second.ID()).ControllerID() != b.PlayerID() {
			t.Fatal("factory instances shared control source")
		}
		g.RemoveFromBattlefield(g.FindPermanent(p1.ID()))
		if g.FindPermanent(first.ID()).ControllerID() != b.PlayerID() || g.FindPermanent(second.ID()).ControllerID() != b.PlayerID() {
			t.Fatal("one instance changed another")
		}
	}
}

func TestRONStaticControlStrictLoadAndInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/LEB.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "LEB.ron")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	lines := strings.Split(string(data), "\n")
	for _, node := range nodes {
		name := node.Fields["Name"].String
		if name == "Control Magic" || name == "Steal Artifact" {
			files[name+".ron"] = &fstest.MapFile{Data: []byte(lines[node.Position.Line-1])}
		}
	}
	if len(files) != 2 {
		t.Fatal("missing real records")
	}
	collection, err := LoadFS(files, "*.ron")
	if err != nil || collection == nil {
		t.Fatalf("strict subset: %v", err)
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 2 || len(report.Issues) != 0 {
		t.Fatal("subset compilation")
	}
	seen := false
	for _, entry := range report.Signatures {
		if entry.Name == "You" {
			seen = true
			if entry.Role != "player_reference" {
				t.Fatalf("controller role: %s", entry.Role)
			}
		}
	}
	if !seen {
		t.Fatal("missing static player reference")
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(AnyPermanent),PermanentLayerEffect(HostPermanent,[SetController(You())])])`)}
	if bad, err := LoadFS(files, "*.ron"); err == nil || bad != nil {
		t.Fatal("invalid collection accepted")
	}
}

func TestRONStaticControlCombinedModifications(t *testing.T) {
	nodes, err := Parse(strings.NewReader(`Card(Name:"Combined",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),PermanentLayerEffect(HostPermanent,[SetController(You),AdjustPT(1,2),AddAbility([Flying])])])`), "combined.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	card := factory()
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 3), b.PlayerID())
	source := g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(source.ID(), host.ID())
	current := g.FindPermanent(host.ID())
	if current.ControllerID() != a.PlayerID() || current.CurrentPower(g) != 3 || current.CurrentToughness(g) != 5 || !current.HasAttr(core.Flying) {
		t.Fatal("combined layer modifications were lost")
	}
	g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
	current = g.FindPermanent(host.ID())
	if current.ControllerID() != b.PlayerID() || current.CurrentPower(g) != 2 || current.CurrentToughness(g) != 3 || current.HasAttr(core.Flying) {
		t.Fatal("combined modifications outlived Aura")
	}
}

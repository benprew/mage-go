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

var preventionCards = []struct{ set, name string }{
	{"ATQ", "Argothian Pixies"}, {"ARN", "Desert Nomads"}, {"DRK", "Uncle Istvan"},
}

func TestRONStaticPreventionCards(t *testing.T) {
	for _, tc := range preventionCards {
		for _, kind := range []string{"artifact-creature", "creature", "artifact", "desert"} {
			t.Run(tc.name+"/"+kind, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
				victim := g.PutOnBattlefield(card, a.PlayerID())
				other := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), b.PlayerID())
				sourceCard := mage.NewCreature("Source", "", 1, 10)
				switch kind {
				case "artifact-creature":
					sourceCard.AddType(core.TypeArtifact)
				case "artifact":
					sourceCard = mage.NewArtifact("Source", "")
				case "desert":
					sourceCard = mage.NewLand("Source", mage.WithSubTypes("Desert"))
				}
				source := g.PutOnBattlefield(sourceCard, b.PlayerID())
				captured := g.CaptureDamageSource(source.ID())
				match := tc.name == "Argothian Pixies" && kind == "artifact-creature" || tc.name == "Desert Nomads" && kind == "desert" || tc.name == "Uncle Istvan" && (kind == "creature" || kind == "artifact-creature")
				want := 1
				if match {
					want = 0
				}
				for range 2 {
					g.DealSimultaneousDamage(mage.PermanentDamageAssignment(g.FindPermanent(victim.ID()), 1, captured), mage.PermanentDamageAssignment(g.FindPermanent(other.ID()), 1, captured), mage.PlayerDamageAssignment(a, 1, captured))
				}
				if g.FindPermanent(victim.ID()).Damage != 2*want || g.FindPermanent(other.ID()).Damage != 2*want || a.Life() != 18 {
					t.Fatal("source filter, repeated prevention, or recipient scope")
				}
			})
		}
	}
}

func TestRONStaticPreventionLifecycle(t *testing.T) {
	for _, tc := range preventionCards {
		for _, change := range []string{"departed-source", "returned-source", "phased-source", "source-characteristics", "recipient-phase-cycle", "recipient-control", "recipient-return", "ability-removal", "clone"} {
			if change == "source-characteristics" && tc.name == "Desert Nomads" {
				continue
			}
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				victim := g.PutOnBattlefield(card, a.PlayerID())
				sourceCard := mage.NewLand("Source")
				sourceCard.AddSubType("Desert")
				source := g.PutOnBattlefield(sourceCard, b.PlayerID())
				source.GrantBaseAttr(core.AttrIsCreature)
				source.GrantBaseAttr(core.AttrIsArtifact)
				captured := g.CaptureDamageSource(source.ID())
				want := 0
				switch change {
				case "departed-source", "returned-source":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if change == "returned-source" {
						g.PutOnBattlefield(sourceCard, a.PlayerID())
					}
				case "phased-source":
					g.PhaseOut(source.ID())
				case "source-characteristics":
					p := g.MutablePermanent(source.ID())
					p.RevokeBaseAttr(core.AttrIsCreature)
					p.RevokeBaseAttr(core.AttrIsArtifact)
					want = 1
				case "recipient-phase-cycle":
					g.PhaseIn(g.PhaseOut(victim.ID()))
				case "recipient-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: victim.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "recipient-return":
					g.RemoveFromBattlefield(g.FindPermanent(victim.ID()))
					victim = g.PutOnBattlefield(card, b.PlayerID())
				case "ability-removal":
					removal := mage.RemoveAllAbilities(victim.ID(), core.WhileOnBattlefield)
					removal.SetSourceID(source.ID())
					g.AddContinuousEffect(removal)
					want = 1
				case "clone":
					clone := g.Clone()
					removal := mage.RemoveAllAbilities(victim.ID(), core.WhileOnBattlefield)
					removal.SetSourceID(source.ID())
					clone.AddContinuousEffect(removal)
					clone.DealSimultaneousDamage(mage.PermanentDamageAssignment(clone.FindPermanent(victim.ID()), 1, captured))
					if clone.FindPermanent(victim.ID()).Damage != 1 || g.FindPermanent(victim.ID()).Damage != 0 {
						t.Fatal("clone isolation")
					}
				}
				g.DealSimultaneousDamage(mage.PermanentDamageAssignment(g.FindPermanent(victim.ID()), 1, captured))
				if g.FindPermanent(victim.ID()).Damage != want {
					t.Fatalf("damage=%d want=%d", g.FindPermanent(victim.ID()).Damage, want)
				}
			})
		}
	}
}

func TestRONStaticPreventionOtherRules(t *testing.T) {
	g, a, b := ronGame()
	pixies := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Argothian Pixies"), a.PlayerID())
	artifact := mage.NewCreature("Artifact creature", "", 1, 4)
	artifact.AddType(core.TypeArtifact)
	blocker := g.PutOnBattlefield(artifact, b.PlayerID())
	plain := g.PutOnBattlefield(mage.NewCreature("Plain", "", 1, 4), b.PlayerID())
	if mage.CanBlock(blocker, pixies, g) || !mage.CanBlock(plain, pixies, g) {
		t.Fatal("Pixies blocking restriction")
	}
	nomads := g.PutOnBattlefield(repositoryCard(t, "ARN", "Desert Nomads"), a.PlayerID())
	if mage.HasLandwalkEvasion(nomads, b.PlayerID(), g) {
		t.Fatal("landwalk without Desert")
	}
	g.PutOnBattlefield(mage.NewLand("Desert", mage.WithSubTypes("Desert")), b.PlayerID())
	if !mage.HasLandwalkEvasion(nomads, b.PlayerID(), g) {
		t.Fatal("Desert landwalk")
	}
	for _, p := range []*mage.Permanent{pixies, nomads} {
		p.RevokeBaseAttr(core.AttrSummonSick)
	}
	g.ExecuteAttackers(a.PlayerID(), []uuid.UUID{pixies.ID(), nomads.ID()})
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: pixies.ID()}})
	if g.GetCombat().IsBlocking(blocker.ID()) {
		t.Fatal("illegal native blocks")
	}
	g.ExecuteCombatDamage()
	if b.Life() != 16 {
		t.Fatal("unblocked native damage")
	}
}

func TestRONStaticPreventionInvalid(t *testing.T) {
	valid := `PreventDamage(DamageWouldBeDealtByAPermanentToRecipient(IsCardtype(Creature),Permanent(ThisPermanent)),[PreventThatDamage])`
	for _, rule := range []string{
		`PreventDamage`, `PreventDamage()`, `PreventDamage(DamageWouldBeDealtByAPermanentToRecipient(IsCardtype(Creature),Permanent(ThisPermanent)))`,
		strings.ReplaceAll(valid, "ThisPermanent", "ThisPermanent()"), strings.ReplaceAll(valid, "ThisPermanent", "Ref_TargetPermanent"), strings.ReplaceAll(valid, "ThisPermanent", "HostPermanent"), strings.ReplaceAll(valid, "ThisPermanent", "Trigger_ThatPermanent"),
		strings.ReplaceAll(valid, "Permanent(ThisPermanent)", "Player(You)"), strings.ReplaceAll(valid, "Permanent(ThisPermanent)", "ThisPermanent"),
		strings.ReplaceAll(valid, "[PreventThatDamage]", "[]"), strings.ReplaceAll(valid, "[PreventThatDamage]", "[PreventThatDamage,PreventThatDamage]"), strings.ReplaceAll(valid, "PreventThatDamage", "PreventThatDamage()"), strings.ReplaceAll(valid, "PreventThatDamage", "DrawCards(You,Integer(1))"),
		strings.ReplaceAll(valid, "IsCardtype(Creature)", "Other(ThisPermanent)"), strings.ReplaceAll(valid, "IsCardtype(Creature)", "IsCardtype()"),
		`Activated(TapPermanent(ThisPermanent),ActionList([` + valid + `]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rule+`])`), "invalid-prevention.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
	nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid spell",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[`+valid+`])`), "invalid-prevention.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Compile(nodes[0]); err == nil {
		t.Fatal("accepted static prevention on instant")
	}
	report, err := InspectFS(fstest.MapFS{"prevention.ron": {Data: []byte(fmt.Sprintf(`Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[%s])`, valid))}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range report.Signatures {
		expected := map[string]string{"IsCardtype": "permanent_filter", "ThisPermanent": "permanent_reference", "PreventThatDamage": "prevention_operation", "Permanent": "prevention_recipient"}[entry.Name]
		if expected != "" && entry.Role != expected {
			t.Fatalf("%s role=%s want=%s", entry.Name, entry.Role, expected)
		}
	}
}

func TestRONStaticPreventionPendingActivation(t *testing.T) {
	for _, tc := range []struct{ set, name, sourceSet, sourceName string }{
		{"ATQ", "Argothian Pixies", "ATQ", "Grapeshot Catapult"},
		{"DRK", "Uncle Istvan", "LEB", "Prodigal Sorcerer"},
	} {
		for _, change := range []string{"present", "departed", "returned", "phased", "copy"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				victim := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
				victim.GrantBaseAttr(core.Flying)
				card := repositoryCard(t, tc.sourceSet, tc.sourceName)
				source := g.PutOnBattlefield(card, b.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "departed", "returned":
					g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
					if change == "returned" {
						g.PutOnBattlefield(card, a.PlayerID())
					}
				case "phased":
					g.PhaseOut(source.ID())
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("copy failed")
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				clone.ResolveStack()
				if g.FindPermanent(victim.ID()).Damage != 0 || clone.FindPermanent(victim.ID()).Damage != 0 {
					t.Fatal("pending RON damage lost prevention or source identity")
				}
			})
		}
	}
}

func TestRONStaticPreventionCombatAndSpell(t *testing.T) {
	for _, tc := range preventionCards {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			victim := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
			sourceCard := mage.NewCreature("Attacker", "", 4, 10)
			sourceCard.AddType(core.TypeArtifact)
			sourceCard.AddType(core.TypeLand)
			sourceCard.AddSubType("Desert")
			attacker := g.PutOnBattlefield(sourceCard, b.PlayerID())
			g.GetCombat().AddAttacker(attacker.ID(), a.PlayerID())
			g.GetCombat().AddBlocker(victim.ID(), attacker.ID())
			g.ExecuteCombatDamage()
			if g.FindPermanent(victim.ID()) == nil || g.FindPermanent(victim.ID()).Damage != 0 || g.FindPermanent(attacker.ID()).Damage != victim.CurrentPower(g) {
				t.Fatal("combat prevention replaced wrong packet")
			}
			bolt := repositoryCard(t, "LEB", "Lightning Bolt")
			bolt.SetOwner(b.PlayerID())
			b.AddToHand(bolt)
			b.ManaPool().Add(core.Red, 1)
			if err := g.CastSpellByID(b.PlayerID(), bolt.ID(), []uuid.UUID{victim.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(victim.ID()) != nil {
				t.Fatal("permanent-source filter prevented spell damage")
			}
		})
	}
}

func TestRONStaticPreventionReplacementOrder(t *testing.T) {
	for mode := range 2 {
		a := mage.NewBasePlayer("A")
		b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B"), mode: mode}
		g := mage.NewGame(a, b)
		victim := g.PutOnBattlefield(repositoryCard(t, "DRK", "Uncle Istvan"), b.PlayerID())
		source := g.PutOnBattlefield(mage.NewCreature("Source", "", 1, 10), a.PlayerID())
		g.SetCreatureDamageRedirect(victim.ID(), b.PlayerID())
		g.DealSimultaneousDamage(mage.PermanentDamageAssignment(victim, 1, g.CaptureDamageSource(source.ID())))
		if g.FindPermanent(victim.ID()).Damage != 0 || b.prompts != 1 {
			t.Fatal("replacement choice or recheck")
		}
		want := 20
		if mode == 0 {
			want = 19
		}
		if b.Life() != want {
			t.Fatalf("life=%d want=%d", b.Life(), want)
		}
	}
}

func TestRONStaticPreventionStrictLoadAndInstances(t *testing.T) {
	files := fstest.MapFS{}
	for _, tc := range preventionCards {
		data, err := os.ReadFile("../../../../cards/mtgish/" + tc.set + ".lines.ron")
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, `Name: "`+tc.name+`"`) {
				files[tc.name+".ron"] = &fstest.MapFile{Data: []byte(line)}
			}
		}
	}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 3 {
		t.Fatal("strict collection count")
	}
	for _, tc := range preventionCards {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			first, err := registry.CreateCard(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			second, err := registry.CreateCard(tc.name)
			if err != nil {
				t.Fatal(err)
			}
			left := g.PutOnBattlefield(first, a.PlayerID())
			right := g.PutOnBattlefield(second, b.PlayerID())
			source := g.PutOnBattlefield(mage.NewLand("Source", mage.WithSubTypes("Desert")), a.PlayerID())
			source.GrantBaseAttr(core.AttrIsArtifact)
			source.GrantBaseAttr(core.AttrIsCreature)
			captured := g.CaptureDamageSource(source.ID())
			removal := mage.RemoveAllAbilities(left.ID(), core.WhileOnBattlefield)
			removal.SetSourceID(source.ID())
			g.AddContinuousEffect(removal)
			g.DealSimultaneousDamage(mage.PermanentDamageAssignment(left, 1, captured), mage.PermanentDamageAssignment(right, 1, captured))
			if g.FindPermanent(left.ID()).Damage != 1 || g.FindPermanent(right.ID()).Damage != 0 {
				t.Fatal("factory instances share prevention state")
			}
		})
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Rejected",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}
	if got, err := LoadFS(files, "*.ron"); err == nil || got != nil {
		t.Fatal("partial registry published")
	}
}

type ronPreventionBlocker struct {
	*mage.BasePlayer
	blocks []mage.BlockAssignment
}

func (p *ronPreventionBlocker) DeclareBlockers(*mage.Game) []mage.BlockAssignment { return p.blocks }

func TestRONStaticPreventionNativeLandwalk(t *testing.T) {
	for _, desert := range []bool{false, true} {
		a := mage.NewBasePlayer("A")
		b := &ronPreventionBlocker{BasePlayer: mage.NewBasePlayer("B")}
		g := mage.NewGame(a, b)
		g.SetActivePlayerIndex(0)
		g.SetTurn(1)
		nomads := g.PutOnBattlefield(repositoryCard(t, "ARN", "Desert Nomads"), a.PlayerID())
		blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 1, 10), b.PlayerID())
		if desert {
			g.PutOnBattlefield(mage.NewLand("Desert", mage.WithSubTypes("Desert")), b.PlayerID())
		}
		g.GetCombat().AddAttacker(nomads.ID(), b.PlayerID())
		b.blocks = []mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: nomads.ID()}}
		g.RunStepWithPriority(core.DeclareBlockers)
		if g.GetCombat().IsBlocking(blocker.ID()) == desert {
			t.Fatal("native landwalk legality")
		}
	}
}

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

func TestRONStaticSourceTypePreventionTreefolk(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ATQ", "Argothian Treefolk")
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil {
		t.Fatal("unpaid cast succeeded")
	}
	a.ManaPool().Add(core.Green, 5)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if g.FindPermanent(card.ID()) != nil {
		t.Fatal("early entry")
	}
	g.ResolveStack()
	victim := g.FindPermanent(card.ID())
	if victim.CurrentPower(g) != 3 || victim.CurrentToughness(g) != 5 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("metadata or payment")
	}
	source := g.PutOnBattlefield(mage.NewArtifact("Source", ""), b.PlayerID())
	for range 2 {
		g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 2, source.ID())
	}
	if g.FindPermanent(victim.ID()).Damage != 0 {
		t.Fatal("artifact damage")
	}
	plain := g.PutOnBattlefield(mage.NewCreature("Plain", "", 1, 5), b.PlayerID())
	g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, plain.ID())
	if g.FindPermanent(victim.ID()).Damage != 1 {
		t.Fatal("nonartifact damage")
	}
}

func TestRONStaticSourceTypePreventionInvalid(t *testing.T) {
	valid := `PreventDamage(DamageWouldBeDealtByASourceToRecipient(IsCardtype(Artifact),Permanent(ThisPermanent)),[PreventThatDamage])`
	for _, rule := range []string{
		strings.ReplaceAll(valid, "ThisPermanent", "ThisPermanent()"), strings.ReplaceAll(valid, "ThisPermanent", "HostPermanent"), strings.ReplaceAll(valid, "ThisPermanent", "Ref_TargetPermanent"), strings.ReplaceAll(valid, "ThisPermanent", "Trigger_ThatPermanent"),
		strings.ReplaceAll(valid, "IsCardtype(Artifact)", "IsCardtype(Artifact())"), strings.ReplaceAll(valid, "IsCardtype(Artifact)", "IsCardtype(Unknown)"), strings.ReplaceAll(valid, "IsCardtype(Artifact)", "IsCardtype(Artifact,Creature)"), strings.ReplaceAll(valid, "IsCardtype(Artifact)", "IsAttacking"),
		strings.ReplaceAll(valid, "Permanent(ThisPermanent)", "Player(You)"), strings.ReplaceAll(valid, "[PreventThatDamage]", "[]"), strings.ReplaceAll(valid, "PreventThatDamage]", "PreventThatDamage()]"),
		`Activated(PayMana([]),ActionList([` + valid + `]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONStaticSourceTypePreventionSpellsAndActivations(t *testing.T) {
	for _, change := range []string{"normal", "depart", "return", "phase", "copy", "clone"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			victim := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Argothian Treefolk"), a.PlayerID())
			victim.GrantBaseAttr(core.Flying)
			card := repositoryCard(t, "ATQ", "Grapeshot Catapult")
			source := g.PutOnBattlefield(card, b.PlayerID())
			source.RevokeBaseAttr(core.AttrSummonSick)
			if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{victim.ID()}); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "depart", "return":
				g.RemoveFromBattlefield(g.FindPermanent(source.ID()))
				if change == "return" {
					returned := g.PutOnBattlefield(card, b.PlayerID())
					returned.RevokeBaseAttr(core.AttrIsArtifact)
				}
			case "phase":
				g.PhaseOut(source.ID())
			case "copy":
				if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
					t.Fatal("copy")
				}
			}
			clone := g.Clone()
			clone.ResolveStack()
			g.ResolveStack()
			want := 0
			for _, game := range []*mage.Game{g, clone} {
				if game.FindPermanent(victim.ID()).Damage != want {
					t.Fatal("activation source identity")
				}
			}
			spellFactory := permanentScopeFactory(t, "Artifact, Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([SpellDealsDamage(ThisSpell,Integer(1),Permanent(Ref_TargetPermanent))])))`)
			spell := spellFactory()
			spell.SetOwner(b.PlayerID())
			b.AddToHand(spell)
			if err := g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{victim.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(victim.ID()).Damage != want {
				t.Fatal("artifact spell damage")
			}
		})
	}
}

func TestRONStaticSourceTypePreventionHost(t *testing.T) {
	for _, change := range []string{"normal", "reattach", "aura-phase", "aura-return", "host-return", "host-phase", "host-removal", "aura-removal", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			nodes, err := Parse(strings.NewReader(`Card(Name:"Host prevention fixture",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(IsCardtype(Creature)),PreventDamage(DamageWouldBeDealtByASourceToRecipient(IsCardtype(Artifact),Permanent(HostPermanent)),[PreventThatDamage])])`), "host.ron")
			if err != nil {
				t.Fatal(err)
			}
			factory, err := Compile(nodes[0])
			if err != nil {
				t.Fatal(err)
			}
			card := factory()
			card.AddSubType("Aura")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			hostCard := mage.NewCreature("Host", "", 2, 10)
			host := g.PutOnBattlefield(hostCard, a.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 10), b.PlayerID())
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			aura := g.FindPermanent(card.ID())
			switch change {
			case "reattach":
				g.Attach(aura.ID(), other.ID())
			case "aura-phase":
				g.PhaseOut(aura.ID())
			case "aura-return":
				g.RemoveFromBattlefield(aura)
				g.PutOnBattlefield(card, a.PlayerID())
			case "host-return":
				g.RemoveFromBattlefield(host)
				host = g.PutOnBattlefield(hostCard, a.PlayerID())
			case "host-phase":
				g.PhaseOut(host.ID())
			case "host-removal", "aura-removal":
				id := host.ID()
				if change == "aura-removal" {
					id = aura.ID()
				}
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), id, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: aura.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			source := g.PutOnBattlefield(mage.NewArtifact("Source", ""), b.PlayerID())
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.DealSimultaneousDamage(mage.PermanentDamageAssignment(game.FindPermanentIncludingPhased(host.ID()), 1, game.CaptureDamageSource(source.ID())), mage.PermanentDamageAssignment(game.FindPermanent(other.ID()), 1, game.CaptureDamageSource(source.ID())))
				wantHost, wantOther := 0, 1
				if change == "reattach" {
					wantHost, wantOther = 1, 0
				}
				if change == "aura-phase" || change == "aura-return" || change == "host-return" || change == "aura-removal" {
					wantHost = 1
				}
				if game.FindPermanentIncludingPhased(host.ID()).Damage != wantHost || game.FindPermanent(other.ID()).Damage != wantOther {
					t.Fatal("live host scope")
				}
			}
		})
	}
}

func TestRONStaticSourceTypePreventionOrdering(t *testing.T) {
	for mode := range 2 {
		a := mage.NewBasePlayer("A")
		b := &ronModePlayer{BasePlayer: mage.NewBasePlayer("B"), mode: mode}
		g := mage.NewGame(a, b)
		victim := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Argothian Treefolk"), b.PlayerID())
		source := g.PutOnBattlefield(mage.NewArtifact("Source", ""), a.PlayerID())
		g.SetCreatureDamageRedirect(victim.ID(), b.PlayerID())
		g.DealDamageToPermanent(g.FindPermanent(victim.ID()), 1, source.ID())
		want := 20
		if mode == 0 {
			want = 19
		}
		if b.prompts != 1 || b.Life() != want || g.FindPermanent(victim.ID()).Damage != 0 {
			t.Fatal("affected player ordering or redirect recheck")
		}
	}
}

func TestRONStaticSourceTypePreventionStrictInventory(t *testing.T) {
	data, err := os.ReadFile("../../../../cards/mtgish/ATQ.lines.ron")
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := Parse(strings.NewReader(string(data)), "ATQ.ron")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	for _, node := range nodes {
		name := node.Fields["Name"].String
		if name == "Argothian Treefolk" || name == "Artifact Ward" {
			line := strings.Split(string(data), "\n")[node.Position.Line-1]
			files[name+".ron"] = &fstest.MapFile{Data: []byte(line)}
		}
	}
	report, err := InspectFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if report.Compiled != 1 || len(report.Issues) != 1 || report.Issues[0].Card != "Artifact Ward" || !strings.Contains(report.Issues[0].Message, "unsupported self untap") {
		t.Fatal("unexpected complete record support", report.Issues)
	}
	found := false
	for _, sig := range report.Signatures {
		if sig.Name == "IsCardtype" && sig.Role == "damage_source_filter" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing damage source role")
	}
	registry := mage.NewRegistry()
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 0 {
		t.Fatal("partial registry installed")
	}
	delete(files, "Artifact Ward.ron")
	files["reprint.ron"] = files["Argothian Treefolk.ron"]
	if err := LoadInto(registry, files, "*.ron"); err != nil {
		t.Fatal(err)
	}
	if len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("reprint count")
	}
	for range 2 {
		card, err := registry.CreateCard("Argothian Treefolk")
		if err != nil {
			t.Fatal(err)
		}
		if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
			t.Fatal(err)
		}
	}
	files["invalid.ron"] = &fstest.MapFile{Data: []byte(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Artifact],Subtypes:[]),Rules:[UnknownRule])`)}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("atomic registry changed")
	}
}

func TestRONStaticSourceTypePreventionCombat(t *testing.T) {
	g, a, b := ronGame()
	victim := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Argothian Treefolk"), a.PlayerID())
	artifact := mage.NewCreature("Artifact creature", "", 2, 10)
	artifact.AddType(core.TypeArtifact)
	attacker := g.PutOnBattlefield(artifact, b.PlayerID())
	attacker.RevokeBaseAttr(core.AttrSummonSick)
	g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID()})
	g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: victim.ID(), AttackerID: attacker.ID()}})
	g.ExecuteCombatDamage()
	if g.FindPermanent(victim.ID()).Damage != 0 || g.FindPermanent(attacker.ID()).Damage != 3 {
		t.Fatal("native combat prevention")
	}
}

func TestRONStaticSourceTypePreventionInstances(t *testing.T) {
	g, a, b := ronGame()
	first := repositoryCard(t, "ATQ", "Argothian Treefolk")
	second := repositoryCard(t, "ATQ", "Argothian Treefolk")
	if first.Abilities()[0] == second.Abilities()[0] {
		t.Fatal("shared static ability")
	}
	left := g.PutOnBattlefield(first, a.PlayerID())
	right := g.PutOnBattlefield(second, b.PlayerID())
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), left.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(mage.NewArtifact("Source", ""), a.PlayerID())
	saved := g.CaptureDamageSource(source.ID())
	g.DealSimultaneousDamage(mage.PermanentDamageAssignment(g.FindPermanent(left.ID()), 1, saved), mage.PermanentDamageAssignment(g.FindPermanent(right.ID()), 1, saved))
	if g.FindPermanent(left.ID()).Damage != 1 || g.FindPermanent(right.ID()).Damage != 0 {
		t.Fatal("instance rule removal")
	}
}

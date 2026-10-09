package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONAbilityTargetRestrictionArtifactWard(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "ATQ", "Artifact Ward")
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 10), a.PlayerID())
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
		t.Fatal("unpaid Aura cast")
	}
	a.ManaPool().Add(core.White, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	if len(host.Attachments) != 0 {
		t.Fatal("early attachment")
	}
	g.ResolveStack()
	artifact := mage.NewCreature("Artifact", "", 2, 10)
	artifact.AddType(core.TypeArtifact)
	source := g.PutOnBattlefield(artifact, b.PlayerID())
	plain := g.PutOnBattlefield(mage.NewCreature("Plain", "", 2, 10), b.PlayerID())
	if mage.CanBlock(source, g.FindPermanent(host.ID()), g) || !mage.CanBlock(plain, g.FindPermanent(host.ID()), g) {
		t.Fatal("blocking restriction")
	}
	g.DealDamageToPermanent(g.FindPermanent(host.ID()), 2, source.ID())
	g.DealDamageToPermanent(g.FindPermanent(host.ID()), 1, plain.ID())
	if g.FindPermanent(host.ID()).Damage != 1 {
		t.Fatal("damage prevention")
	}
	if g.FindPermanent(host.ID()).CanBeTargetedBy(source.Card, b.PlayerID(), g) || !g.FindPermanent(host.ID()).CanBeTargetedBy(plain.Card, b.PlayerID(), g) {
		t.Fatal("ability restriction")
	}
	spell := artifact.Copy()
	if !g.FindPermanent(host.ID()).CanBeTargetedBy(mage.SpellTargetSource(spell), b.PlayerID(), g) {
		t.Fatal("artifact spell restricted")
	}
}

func TestRONAbilityTargetRestrictionLifecycle(t *testing.T) {
	for _, change := range []string{"none", "reattach", "aura-phase", "host-phase", "aura-left", "aura-return", "host-return", "aura-removal", "host-removal", "control"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := mage.NewCreature("Host", "", 2, 10)
			host := g.PutOnBattlefield(hostCard, a.PlayerID())
			other := g.PutOnBattlefield(mage.NewCreature("Other", "", 2, 10), b.PlayerID())
			card := repositoryCard(t, "ATQ", "Artifact Ward")
			aura := g.PutOnBattlefield(card, a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			artifact := g.PutOnBattlefield(mage.NewArtifact("Source", ""), b.PlayerID())
			clone := g.Clone()
			switch change {
			case "reattach":
				g.Attach(aura.ID(), other.ID())
			case "aura-phase":
				g.PhaseOut(aura.ID())
			case "host-phase":
				g.PhaseOut(host.ID())
			case "aura-left", "aura-return":
				g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
				if change == "aura-return" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			case "host-return":
				g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
				host = g.PutOnBattlefield(hostCard, a.PlayerID())
			case "aura-removal", "host-removal":
				id := aura.ID()
				if change == "host-removal" {
					id = host.ID()
				}
				if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), id, a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: aura.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			}
			want := change == "reattach" || change == "aura-phase" || change == "aura-left" || change == "aura-return" || change == "host-return" || change == "aura-removal"
			if g.FindPermanentIncludingPhased(host.ID()).CanBeTargetedBy(artifact.Card, b.PlayerID(), g) != want {
				t.Fatal("host lifecycle")
			}
			if g.FindPermanent(other.ID()).CanBeTargetedBy(artifact.Card, b.PlayerID(), g) != (change != "reattach") {
				t.Fatal("other host")
			}
			if clone.FindPermanent(host.ID()).CanBeTargetedBy(artifact.Card, b.PlayerID(), clone) {
				t.Fatal("clone changed")
			}
			if change == "aura-phase" {
				g.PhaseIn([]uuid.UUID{aura.ID()})
				if g.FindPermanent(host.ID()).CanBeTargetedBy(artifact.Card, b.PlayerID(), g) {
					t.Fatal("phase restoration")
				}
			}
			if change == "aura-removal" {
				g.GetEffects().RemoveEndOfTurn()
				g.ApplyContinuousEffects()
				if g.FindPermanent(host.ID()).CanBeTargetedBy(artifact.Card, b.PlayerID(), g) {
					t.Fatal("ability restoration")
				}
			}
		})
	}
}

func TestRONAbilityTargetRestrictionInvalid(t *testing.T) {
	valid := `PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfAbilities(AbilityOfASource(IsCardtype(Artifact)))])`
	for _, rule := range []string{
		strings.Replace(valid, "ThisPermanent", "HostPermanent", 1), strings.Replace(valid, "ThisPermanent", "ThisPermanent()", 1), strings.Replace(valid, "ThisPermanent", "Ref_TargetPermanent", 1),
		strings.Replace(valid, "IsCardtype(Artifact)", "IsCardtype(Artifact())", 1), strings.Replace(valid, "IsCardtype(Artifact)", "IsCardtype(Unknown)", 1), strings.Replace(valid, "IsCardtype(Artifact)", "IsColor(White())", 1), strings.Replace(valid, "IsCardtype(Artifact)", "IsColor(Unknown)", 1), strings.Replace(valid, "IsCardtype(Artifact)", "IsColor(Colorless)", 1),
		strings.Replace(valid, "AbilityOfASource(IsCardtype(Artifact))", "AbilityOfASource()", 1), strings.Replace(valid, "AbilityOfASource(IsCardtype(Artifact))", "AnyAbility", 1), strings.Replace(valid, "AbilityOfASource(IsCardtype(Artifact))", "CanTargetOnly(IsCreatureType(Wall))", 1),
		strings.Replace(valid, "IsCardtype(Artifact)", "And([])", 1), strings.Replace(valid, "IsCardtype(Artifact)", "AnySpell", 1), strings.Replace(valid, "IsCardtype(Artifact)", "IsEnchantmentType(Aura)", 1),
		strings.Replace(valid, "CantBeTheTargetOfAbilities(AbilityOfASource(IsCardtype(Artifact)))", "CantBeTheTargetOfAbilities(AbilityOfASource(IsCardtype(Artifact)),AnyAbility)", 1),
		`Activated(PayMana([]),ActionList([` + valid + `]))`,
		strings.Replace(valid, "CantBeTheTargetOfAbilities(AbilityOfASource(IsCardtype(Artifact)))", "CantBeTheTargetOfAbilities(AbilityOfASource(IsCardtype(Artifact))),CantBeEnchantedByAnEnchantment(And([Other(ThisPermanent),IsEnchantmentType(Aura)]))", 1),
		strings.Replace(valid, "CantBeTheTargetOfAbilities(AbilityOfASource(IsCardtype(Artifact)))", "CantBeTheTargetOfAbilities(AbilityOfASource(IsCardtype(Artifact))),CantBeTheTargetOfAbilities(AbilityOfASource(IsColor(White)))", 1),
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:2,Toughness:5),Rules:[`+rule+`])`), "bad.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONAbilityTargetRestrictionRaidingParty(t *testing.T) {
	record := staticBaseRecord(t, "FEM", "Raiding Party")
	if _, err := Compile(record); err == nil || !strings.Contains(err.Error(), "TapAnyNumberOfPermanents") {
		t.Fatal("complete unsupported procedure was accepted", err)
	}
	factory := staticBaseSubrule(t, record, record.Fields["Rules"].Args[0])
	g, a, b := ronGame()
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	white := mage.NewCreature("White", "{W}", 1, 1)
	red := mage.NewCreature("Red", "{R}", 1, 1)
	for _, c := range []mage.Card{white, red} {
		for _, source := range []mage.Card{c, mage.SpellTargetSource(c)} {
			if p.CanBeTargetedBy(source, b.PlayerID(), g) != (c == red) {
				t.Fatal("source color")
			}
		}
	}
	report, err := InspectFS(fstest.MapFS{"rule.ron": {Data: []byte(`Card(Name:"Rule",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfAbilities(AbilityOfASource(IsColor(White)))])])`)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatal(err, report.Issues)
	}
	for name, role := range map[string]string{"CantBeTheTargetOfAbilities": "permanent_rule_operation", "AbilityOfASource": "ability_filter", "IsColor": "ability_source_filter", "White": "color"} {
		found := false
		for _, s := range report.Signatures {
			if s.Name == name && s.Role == role {
				found = true
			}
		}
		if !found {
			t.Fatal("missing role", name, role)
		}
	}
}

func TestRONAbilityTargetRestrictionActivationAndResolution(t *testing.T) {
	for _, mode := range []string{"blocked", "allowed-spell", "resolution", "resolution-copy"} {
		t.Run(mode, func(t *testing.T) {
			g, a, b := ronGame()
			host := g.PutOnBattlefield(mage.NewCreature("Host", "", 2, 10), a.PlayerID())
			auraCard := repositoryCard(t, "ATQ", "Artifact Ward")
			aura := g.PutOnBattlefield(auraCard, a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([ManaCostGeneric(1)]),Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AdjustPT(+1,+1)],UntilEndOfTurn)])))`)
			source := g.PutOnBattlefield(factory(), b.PlayerID())
			b.ManaPool().Add(core.Colorless, 1)
			if mode == "allowed-spell" {
				spell := permanentScopeFactory(t, "Artifact, Instant", `SpellActions(Targeted([TargetPermanent(IsCardtype(Creature))],ActionList([CreatePermanentLayerEffectUntil(Ref_TargetPermanent,[AdjustPT(+3,+3)],UntilEndOfTurn)])))`)()
				spell.SetOwner(b.PlayerID())
				b.AddToHand(spell)
				b.ManaPool().Add(core.Green, 1)
				if err := g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if g.FindPermanent(host.ID()).CurrentPower(g) != 5 {
					t.Fatal("ordinary spell did not resolve")
				}
				return
			}
			if mode == "resolution" || mode == "resolution-copy" {
				g.RemoveFromBattlefield(g.FindPermanent(aura.ID()))
			}
			err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{host.ID()})
			if mode == "blocked" {
				if err == nil || b.ManaPool().TotalMana() != 1 || g.StackSize() != 0 {
					t.Fatal("illegal ability spent resources")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if mode == "resolution-copy" && g.CopyStackObjectDirect(g.StackPeek(), a.PlayerID(), false) == nil {
				t.Fatal("copy")
			}
			returned := g.PutOnBattlefield(auraCard, a.PlayerID())
			g.Attach(returned.ID(), host.ID())
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveStack()
				if game.FindPermanent(host.ID()).CurrentPower(game) != 2 {
					t.Fatal("restricted target received boost")
				}
			}
			if b.ManaPool().TotalMana() != 0 {
				t.Fatal("resolution refunded payment")
			}
		})
	}
}

func TestRONAbilityTargetRestrictionIndependentAndFilters(t *testing.T) {
	for _, filter := range []string{`IsCardtype(Artifact)`, `IsColor(White)`, `And([IsColor(White),Or([IsCardtype(Artifact),IsCardtype(Creature)])])`} {
		g, a, b := ronGame()
		rule := `PermanentRuleEffect(ThisPermanent,[CantBeTheTargetOfAbilities(AbilityOfASource(` + filter + `))])`
		factory := permanentScopeFactory(t, "Artifact", rule)
		first, second := factory(), factory()
		if first.Abilities()[0] == second.Abilities()[0] {
			t.Fatal("shared declaration")
		}
		left := g.PutOnBattlefield(first, a.PlayerID())
		right := g.PutOnBattlefield(second, b.PlayerID())
		card := mage.NewCreature("Source", "{W}", 1, 3)
		card.AddType(core.TypeArtifact)
		source := g.PutOnBattlefield(card, b.PlayerID())
		if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), left.ID(), a.PlayerID(), nil); err != nil {
			t.Fatal(err)
		}
		if !g.FindPermanent(left.ID()).CanBeTargetedBy(source.Card, b.PlayerID(), g) || g.FindPermanent(right.ID()).CanBeTargetedBy(source.Card, b.PlayerID(), g) {
			t.Fatal("sibling declaration changed")
		}
	}
}

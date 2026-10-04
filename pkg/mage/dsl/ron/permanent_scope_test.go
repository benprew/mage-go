package ron

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func permanentScopeFactory(t *testing.T, types, rules string) mage.CardFactory {
	t.Helper()
	src := fmt.Sprintf(`Card(Name: "Scope", Typeline: (Supertypes: [], Cardtypes: [%s], Subtypes: []), Rules: [%s])`, types, rules)
	nodes, err := Parse(strings.NewReader(src), "permanent-scope.ron")
	if err != nil {
		t.Fatal(err)
	}
	factory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	return factory
}

func TestRONPermanentActionsDeclareTypedReads(t *testing.T) {
	for _, action := range []string{"DestroyPermanent", "DestroyPermanentNoRegen", "TapPermanent", "UntapPermanent", "RegeneratePermanent", "SacrificePermanent"} {
		t.Run(action, func(t *testing.T) {
			factory := permanentScopeFactory(t, "Instant", fmt.Sprintf(`SpellActions(Targeted([TargetPermanent(AnyPermanent)], ActionList([%s(Ref_TargetPermanent)])))`, action))
			first, second := factory(), factory()
			if first.Abilities()[0].(*mage.ActionDefinition).Targets()[0] == second.Abilities()[0].(*mage.ActionDefinition).Targets()[0] {
				t.Fatal("shared declaration")
			}
			for _, card := range []mage.Card{first, second} {
				if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
					t.Fatal(err)
				}
			}
			reads := 0
			var walk func(mage.DefinitionMetadata)
			walk = func(n mage.DefinitionMetadata) {
				if n.Role == "permanent-ref" && n.Text == "Ref_TargetPermanent" {
					reads++
				}
				for _, child := range n.Children {
					walk(child)
				}
			}
			walk(mage.InspectDefinition(first.Abilities()[0]))
			if reads != 1 {
				t.Fatalf("expected one typed permanent read, got %d", reads)
			}
			action := first.Abilities()[0].(*mage.ActionDefinition)
			invalid := mage.NewInstant("Missing declaration", "", mage.WithAbility(mage.NewAction(mage.ActionSpell, mage.WithEffects(action.Effects()...))))
			if mage.ValidateCardDefinition(invalid.Name(), invalid) == nil {
				t.Fatal("missing declaration accepted")
			}
		})
	}
}

func TestRONPermanentScopeRejectsInvalidReads(t *testing.T) {
	for _, rule := range []string{
		`Activated(PayMana([]), ActionList([TapPermanent(Ref_TargetPermanent)]))`,
		`Activated(PayMana([]), Targeted([TargetPlayer(AnyPlayer)], ActionList([TapPermanent(Ref_TargetPermanent)])))`,
		`Activated(PayMana([]), Targeted([TargetPermanent(AnyPermanent)], ActionList([TapPermanent(Ref_TargetPermanent())])))`,
		`Activated(PayMana([]), ActionList([RegeneratePermanent(HostPermanent)]))`,
		`Activated(PayMana([]), Targeted([TargetPermanent(AnyPermanent)], ActionList([]))), Activated(PayMana([]), ActionList([TapPermanent(Ref_TargetPermanent)]))`,
		`Activated(PayMana([]), ActionList([DestroyPermanent(Trigger_ThatPermanent)]))`,
	} {
		nodes, err := Parse(strings.NewReader(fmt.Sprintf(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Artifact], Subtypes: []), Rules: [%s])`, rule)), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("invalid scope compiled: %s", rule)
		}
	}
	nodes, err := Parse(strings.NewReader(`Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Instant], Subtypes: []), Rules: [SpellActions(ActionList([DestroyPermanent(ThisPermanent)]))])`), "invalid.ron")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(nodes[0]); err == nil {
		t.Fatal("spell has permanent source")
	}
}

func TestRONHostRegenerationAndSacrificeLKI(t *testing.T) {
	for _, tc := range []struct {
		set, name     string
		color         core.Color
		amount, power int
	}{
		{"LEB", "Regeneration", core.Green, 1, 4},
		{"LEG", "The Brute", core.Red, 3, 5},
		{"FEM", "Thrull Retainer", core.Black, 0, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := repositoryCard(t, "LEB", "Air Elemental")
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			aura := repositoryCard(t, tc.set, tc.name)
			aura.SetOwner(a.PlayerID())
			if err := mage.ValidateCardDefinition(aura.Name(), aura); err != nil {
				t.Fatal(err)
			}
			source := g.PutOnBattlefield(aura, a.PlayerID())
			g.Attach(source.ID(), host.ID())
			if g.FindPermanent(host.ID()).CurrentPower(g) != tc.power {
				t.Fatal("host layer changed")
			}
			index := 0
			if tc.name != "Regeneration" {
				index = 1
			}
			if tc.amount > 0 {
				if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, nil); err == nil {
					t.Fatal("unpaid activation succeeded")
				}
				if g.FindPermanent(source.ID()) == nil || !g.GetStack().IsEmpty() {
					t.Fatal("failed payment changed state")
				}
				a.ManaPool().Add(tc.color, tc.amount)
			}
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), index, nil); err != nil {
				t.Fatal(err)
			}
			if g.HasRegenerationShield(host.ID()) {
				t.Fatal("shield installed before resolution")
			}
			if tc.name == "Thrull Retainer" && (g.FindPermanent(source.ID()) != nil || len(a.Graveyard()) != 1) {
				t.Fatal("source sacrifice was not paid")
			}
			clone := g.Clone()
			clone.ResolveStack()
			if !clone.HasRegenerationShield(host.ID()) || g.HasRegenerationShield(host.ID()) {
				t.Fatal("clone lost or shared host reference")
			}
			g.ResolveStack()
			if !g.HasRegenerationShield(host.ID()) {
				t.Fatal("wrong host or lost departure attachment")
			}
			g.DestroyPermanent(g.FindPermanent(host.ID()))
			if g.FindPermanent(host.ID()) == nil || g.HasRegenerationShield(host.ID()) {
				t.Fatal("regeneration failed")
			}
		})
	}
}

func TestRONPermanentTargetCopiesClonesAndLegality(t *testing.T) {
	g, a, b := ronGame()
	land := repositoryCard(t, "LEB", "Forest")
	land.SetOwner(b.PlayerID())
	original := g.PutOnBattlefield(land, b.PlayerID())
	otherCard := repositoryCard(t, "LEB", "Forest")
	otherCard.SetOwner(a.PlayerID())
	other := g.PutOnBattlefield(otherCard, a.PlayerID())
	spell := repositoryCard(t, "LEB", "Ice Storm")
	spell.SetOwner(a.PlayerID())
	a.AddToHand(spell)
	a.ManaPool().Add(core.Green, 1)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{original.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	clone := g.Clone()
	phased := g.PhaseOut(original.ID())
	if copied := g.CopySpellOnStack(spell.ID(), b.PlayerID(), true); copied == nil || len(copied.Targets) != 1 || copied.Targets[0] != other.ID() {
		t.Fatal("copy did not retarget")
	}
	g.PhaseIn(phased)
	g.BouncePermanentToHand(g.FindPermanent(original.ID()))
	b.RemoveFromHand(land.ID())
	returned := g.PutOnBattlefield(land, b.PlayerID())
	g.ResolveStack()
	if g.FindPermanent(other.ID()) != nil || g.FindPermanent(returned.ID()) == nil {
		t.Fatal("copy or incarnation binding failed")
	}
	if clone.FindPermanent(other.ID()) == nil || clone.FindPermanent(original.ID()) == nil {
		t.Fatal("live actions changed clone")
	}
	clone.ResolveStack()
	if clone.FindPermanent(original.ID()) != nil || clone.FindPermanent(other.ID()) == nil {
		t.Fatal("clone changed targets")
	}
}

func TestRONPermanentActionsPhasingAndSourceDeparture(t *testing.T) {
	for _, phaseIn := range []bool{false, true} {
		t.Run(fmt.Sprint(phaseIn), func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEB", "Icy Manipulator")
			card.SetOwner(a.PlayerID())
			source := g.PutOnBattlefield(card, a.PlayerID())
			targetCard := repositoryCard(t, "LEB", "Air Elemental")
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			a.ManaPool().Add(core.Colorless, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
				t.Fatal(err)
			}
			if !g.FindPermanent(source.ID()).Tapped || g.FindPermanent(target.ID()).Tapped {
				t.Fatal("tap timing changed")
			}
			g.DestroyPermanent(g.FindPermanent(source.ID()))
			phased := g.PhaseOut(target.ID())
			if phaseIn {
				g.PhaseIn(phased)
			}
			g.ResolveStack()
			if g.FindPermanentIncludingPhased(target.ID()).Tapped != phaseIn {
				t.Fatal("phasing legality failed")
			}
		})
	}
}

func TestRONHostReferenceReadsCurrentAttachmentAndRejectsNewIncarnation(t *testing.T) {
	for _, change := range []string{"reattach", "returned-host", "phased-host", "returned-source"} {
		t.Run(change, func(t *testing.T) {
			g, a, b := ronGame()
			hostCard := repositoryCard(t, "LEB", "Air Elemental")
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			otherCard := repositoryCard(t, "LEB", "Air Elemental")
			otherCard.SetOwner(b.PlayerID())
			other := g.PutOnBattlefield(otherCard, b.PlayerID())
			auraCard := repositoryCard(t, "LEB", "Regeneration")
			auraCard.SetOwner(a.PlayerID())
			aura := g.PutOnBattlefield(auraCard, a.PlayerID())
			g.Attach(aura.ID(), host.ID())
			a.ManaPool().Add(core.Green, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), aura.ID(), 0, nil); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "reattach":
				g.Attach(aura.ID(), other.ID())
			case "returned-host":
				g.BouncePermanentToHand(g.FindPermanent(host.ID()))
				b.RemoveFromHand(hostCard.ID())
				g.PutOnBattlefield(hostCard, b.PlayerID())
			case "phased-host":
				g.PhaseOut(host.ID())
			case "returned-source":
				g.BouncePermanentToHand(g.FindPermanent(aura.ID()))
				a.RemoveFromHand(auraCard.ID())
				returned := g.PutOnBattlefield(auraCard, a.PlayerID())
				g.Attach(returned.ID(), other.ID())
			}
			g.ResolveStack()
			wantOld := change == "returned-source"
			wantOther := change == "reattach"
			if g.HasRegenerationShield(host.ID()) != wantOld || g.HasRegenerationShield(other.ID()) != wantOther {
				t.Fatal("host reference followed wrong attachment incarnation")
			}
		})
	}
}

func TestRONUnaryPermanentActionsUseSharedRules(t *testing.T) {
	for _, action := range []string{"DestroyPermanent", "DestroyPermanentNoRegen", "TapPermanent", "UntapPermanent", "RegeneratePermanent", "SacrificePermanent"} {
		for _, own := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", action, own), func(t *testing.T) {
				g, a, b := ronGame()
				owner := b
				if own {
					owner = a
				}
				hostCard := repositoryCard(t, "LEB", "Air Elemental")
				hostCard.SetOwner(owner.PlayerID())
				host := g.PutOnBattlefield(hostCard, owner.PlayerID())
				if action == "UntapPermanent" {
					g.TapPermanent(host)
				}
				if strings.HasPrefix(action, "Destroy") {
					g.AddRegenerationShield(host.ID())
				}
				factory := permanentScopeFactory(t, "Instant", fmt.Sprintf(`SpellActions(Targeted([TargetPermanent(AnyPermanent)], ActionList([%s(Ref_TargetPermanent)])))`, action))
				card := factory()
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				p := g.FindPermanent(host.ID())
				switch action {
				case "DestroyPermanent":
					if p == nil || !p.Tapped || g.HasRegenerationShield(host.ID()) {
						t.Fatal("destroy bypassed regeneration")
					}
				case "DestroyPermanentNoRegen":
					if p != nil {
						t.Fatal("regeneration stopped no-regeneration destruction")
					}
				case "TapPermanent":
					if p == nil || !p.Tapped {
						t.Fatal("tap failed")
					}
				case "UntapPermanent":
					if p == nil || p.Tapped {
						t.Fatal("untap failed")
					}
				case "RegeneratePermanent":
					if !g.HasRegenerationShield(host.ID()) {
						t.Fatal("regenerate failed")
					}
				case "SacrificePermanent":
					if (p == nil) != own {
						t.Fatal("sacrifice ignored resolving controller")
					}
				}
			})
		}
	}
}

func TestRONSourceAndTargetReadsStayDistinct(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `Activated(PayMana([]), Targeted([TargetPermanent(AnyPermanent)], ActionList([UntapPermanent(ThisPermanent), TapPermanent(Ref_TargetPermanent)])))`)
	g, a, b := ronGame()
	sourceCard := factory()
	sourceCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	g.TapPermanent(source)
	card := repositoryCard(t, "LEB", "Air Elemental")
	card.SetOwner(b.PlayerID())
	target := g.PutOnBattlefield(card, b.PlayerID())
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	g.BouncePermanentToHand(g.FindPermanent(source.ID()))
	a.RemoveFromHand(sourceCard.ID())
	returned := g.PutOnBattlefield(sourceCard, a.PlayerID())
	g.TapPermanent(returned)
	g.ResolveStack()
	if !g.FindPermanent(returned.ID()).Tapped || !g.FindPermanent(target.ID()).Tapped {
		t.Fatal("source followed returned incarnation or target lost identity")
	}
}

func TestRONRealPermanentRemovalAndUntap(t *testing.T) {
	for _, tc := range []struct {
		set, name string
		color     core.Color
		amount    int
		creature  bool
	}{
		{"LEB", "Shatter", core.Red, 1, false},
		{"DRK", "Fissure", core.Red, 3, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			targetName := "Icy Manipulator"
			if tc.creature {
				targetName = "Air Elemental"
			}
			targetCard := repositoryCard(t, "LEB", targetName)
			targetCard.SetOwner(b.PlayerID())
			target := g.PutOnBattlefield(targetCard, b.PlayerID())
			if tc.creature {
				g.AddRegenerationShield(target.ID())
			}
			spell := repositoryCard(t, tc.set, tc.name)
			spell.SetOwner(a.PlayerID())
			a.AddToHand(spell)
			colored := 1
			if tc.name == "Fissure" {
				colored = 2
			}
			a.ManaPool().Add(tc.color, colored)
			a.ManaPool().Add(core.Colorless, tc.amount)
			if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{target.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(target.ID()) != nil || len(a.Graveyard()) != 1 || len(b.Graveyard()) != 1 {
				t.Fatal("removal or zone handling failed")
			}
		})
	}
	g, a, b := ronGame()
	land := repositoryCard(t, "LEB", "Forest")
	land.SetOwner(b.PlayerID())
	target := g.PutOnBattlefield(land, b.PlayerID())
	g.TapPermanent(target)
	druidCard := repositoryCard(t, "LEB", "Ley Druid")
	druidCard.SetOwner(a.PlayerID())
	source := g.PutOnBattlefield(druidCard, a.PlayerID())
	source.RevokeBaseAttr(core.AttrSummonSick)
	if err := g.ActivateAbilityByIndex(a.PlayerID(), source.ID(), 0, []uuid.UUID{target.ID()}); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(target.ID()).Tapped || !g.FindPermanent(source.ID()).Tapped {
		t.Fatal("target untap changed tap cost")
	}
}

func TestRONPermanentTriggerBindingAndHostEntryTiming(t *testing.T) {
	factory := permanentScopeFactory(t, "Artifact", `TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([TapPermanent(Ref_TargetPermanent)])))`)
	g, a, b := ronGame()
	creature := repositoryCard(t, "LEB", "Air Elemental")
	creature.SetOwner(b.PlayerID())
	target := g.PutOnBattlefield(creature, b.PlayerID())
	sourceCard := factory()
	sourceCard.SetOwner(a.PlayerID())
	if err := mage.ValidateCardDefinition(sourceCard.Name(), sourceCard); err != nil {
		t.Fatal(err)
	}
	source := g.PutOnBattlefield(sourceCard, a.PlayerID())
	if g.FindPermanent(target.ID()).Tapped {
		t.Fatal("trigger acted at placement")
	}
	g.DestroyPermanent(g.FindPermanent(source.ID()))
	g.ResolveStack()
	if !g.FindPermanent(target.ID()).Tapped {
		t.Fatal("trigger lost target after source departure")
	}

	src := `Card(Name: "Host Entry", Typeline: (Supertypes: [], Cardtypes: [Enchantment], Subtypes: [Aura]), Rules: [EnchantPermanent(IsCardtype(Creature)), TriggerA(WhenAPermanentEntersTheBattlefield(SinglePermanent(ThisPermanent)), ActionList([TapPermanent(HostPermanent)]))])`
	nodes, err := Parse(strings.NewReader(src), "host-entry.ron")
	if err != nil {
		t.Fatal(err)
	}
	auraFactory, err := Compile(nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	g2, player, opponent := ronGame()
	creature2 := repositoryCard(t, "LEB", "Air Elemental")
	creature2.SetOwner(opponent.PlayerID())
	host := g2.PutOnBattlefield(creature2, opponent.PlayerID())
	aura := auraFactory()
	aura.SetOwner(player.PlayerID())
	player.AddToHand(aura)
	if err := g2.CastSpellByID(player.PlayerID(), aura.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g2.ResolveStack()
	if !g2.FindPermanent(host.ID()).Tapped {
		t.Fatal("entry action did not use the attachment established by Aura resolution")
	}
}

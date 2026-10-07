package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONAuraRestrictionCards(t *testing.T) {
	for _, name := range []string{"Anti-Magic Aura", "Consecrate Land"} {
		for _, change := range []string{"none", "source-leaves", "source-control", "remove-source-abilities", "remove-host-abilities", "phase-cycle", "clone", "reattach"} {
			t.Run(name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				set := "LEG"
				hostCard := mage.NewCreature("Host", "", 2, 5)
				otherCard := mage.NewCreature("Other", "", 2, 5)
				if name == "Consecrate Land" {
					set = "LEB"
					hostCard = mage.NewLand("Host")
					otherCard = mage.NewLand("Other")
				}
				hostCard.SetOwner(b.PlayerID())
				otherCard.SetOwner(a.PlayerID())
				host := g.PutOnBattlefield(hostCard, b.PlayerID())
				other := g.PutOnBattlefield(otherCard, a.PlayerID())
				card := repositoryCard(t, set, name)
				if err := mage.ValidateCardDefinition(name, card); err != nil {
					t.Fatal(err)
				}
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err == nil || len(a.Hand()) != 1 {
					t.Fatal("unpaid casting did not roll back")
				}
				a.ManaPool().Add(core.Blue, 3)
				a.ManaPool().Add(core.White, 1)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
					t.Fatal(err)
				}
				g.ResolveStack()
				if p := g.FindPermanent(card.ID()); p == nil || p.AttachedTo != host.ID() {
					t.Fatal("granting Aura could not enchant its subject")
				}
				candidate := mage.NewAura("Candidate", "{W}", mage.WithCastTarget(mage.TargetPermanent(mage.AnyPermanent)))
				candidate.SetOwner(a.PlayerID())
				a.AddToHand(candidate)
				a.ManaPool().Add(core.White, 1)
				mana := a.ManaPool().TotalMana()
				if err := g.CastSpellByID(a.PlayerID(), candidate.ID(), []uuid.UUID{host.ID()}, 0); err == nil || a.ManaPool().TotalMana() != mana || g.StackSize() != 0 {
					t.Fatal("restricted Aura cast or payment rollback")
				}
				clone := g.Clone()
				allowed := false
				switch change {
				case "source-leaves":
					g.BouncePermanentToHand(g.FindPermanent(card.ID()))
					allowed = true
				case "source-control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: other.ID(), TargetID: card.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "remove-source-abilities":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), card.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
					allowed = true
				case "remove-host-abilities":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), host.ID(), b.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(host.ID()))
				case "clone":
					clone.BouncePermanentToHand(clone.FindPermanent(card.ID()))
					if err := clone.CastSpellByID(a.PlayerID(), candidate.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
						t.Fatal(err)
					}
				case "reattach":
					g.Attach(card.ID(), other.ID())
					allowed = true
				}
				if err := g.CastSpellByID(a.PlayerID(), candidate.ID(), []uuid.UUID{host.ID()}, 0); (err == nil) != allowed {
					t.Fatalf("allowed=%v err=%v", allowed, err)
				}
				if allowed {
					g.ResolveStack()
				}
				if name == "Consecrate Land" && !allowed && change != "remove-host-abilities" {
					g.DestroyPermanent(g.FindPermanent(host.ID()))
					if g.FindPermanent(host.ID()) == nil {
						t.Fatal("indestructible land was destroyed")
					}
				}
				if name == "Anti-Magic Aura" && !allowed {
					spell := repositoryCard(t, "LEB", "Giant Growth")
					spell.SetOwner(a.PlayerID())
					a.AddToHand(spell)
					a.ManaPool().Add(core.Green, 1)
					if err := g.CastSpellByID(a.PlayerID(), spell.ID(), []uuid.UUID{host.ID()}, 0); err == nil {
						t.Fatal("non-Aura spell targeted protected creature")
					}
					abilityTarget := mage.DeclarePermanentTarget("victim", mage.AnyPermanent)
					actor := g.PutOnBattlefield(mage.NewArtifact("Actor", "", mage.WithAbility(mage.Activate(mage.Costs(), mage.TapSubject(abilityTarget), mage.WithTargets(abilityTarget)))), a.PlayerID())
					if err := g.ActivateAbilityByIndex(a.PlayerID(), actor.ID(), 0, []uuid.UUID{host.ID()}); err != nil {
						t.Fatal(err)
					}
					g.ResolveStack()
					if !g.FindPermanent(host.ID()).Tapped {
						t.Fatal("ability target was restricted")
					}
				}
			})
		}
	}
}

func TestRONAuraRestrictionInvalid(t *testing.T) {
	for _, rule := range []string{
		`PermanentRuleEffect(Ref_TargetPermanent, [CantBeEnchantedByAnEnchantment(And([Other(ThisPermanent), IsEnchantmentType(Aura)]))])`,
		`PermanentRuleEffect(HostPermanent(), [CantBeEnchantedByAnEnchantment(And([Other(ThisPermanent), IsEnchantmentType(Aura)]))])`,
		`PermanentRuleEffect(HostPermanent, [CantBeEnchantedByAnEnchantment(IsEnchantmentType(Aura))])`,
		`PermanentRuleEffect(HostPermanent, [CantBeEnchantedByAnEnchantment(And([Other(HostPermanent), IsEnchantmentType(Aura)]))])`,
		`PermanentRuleEffect(HostPermanent, [CantBeEnchantedByAnEnchantment(And([Other(ThisPermanent), IsEnchantmentType(Aura), IsColor(Red)]))])`,
		`PermanentRuleEffect(HostPermanent, [CantBeEnchantedByAnEnchantment(And([Other(ThisPermanent), IsEnchantmentType(Aura())]))])`,
		`PermanentRuleEffect(HostPermanent, [CantBeEnchantedByAnEnchantment(And([Other(ThisPermanent), Other(ThisPermanent)]))])`,
		`PermanentRuleEffect(HostPermanent, [CantBeEnchantedByAnEnchantment(And([Other(ThisPermanent), IsEnchantmentType(Aura)])), UnknownRule])`,
		`PermanentRuleEffect(HostPermanent, [])`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(AnyPermanent),`+rule+`])`), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", rule)
		}
	}
}

func TestRONAuraRestrictionAtomicLoadAndInventory(t *testing.T) {
	src := `Card(Name:"Rule",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[Aura]),Rules:[EnchantPermanent(AnyPermanent),PermanentRuleEffect(HostPermanent,[CantBeTheTargetOfSpells(AnySpell),CantBeEnchantedByAnEnchantment(And([IsEnchantmentType(Aura),Other(ThisPermanent)]))])])`
	files := fstest.MapFS{"cards.ron": {Data: []byte(src)}}
	registry, err := LoadFS(files, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	files["bad.ron"] = &fstest.MapFile{Data: []byte(strings.Replace(src, `Name:"Rule"`, `Name:"Bad"`, 1) + strings.Replace(src, `Other(ThisPermanent)`, `Other(HostPermanent)`, 1))}
	if err := LoadInto(registry, files, "*.ron"); err == nil || len(registry.RegisteredCardNames()) != 1 {
		t.Fatal("invalid collection changed registry")
	}
	report, err := InspectFS(fstest.MapFS{"cards.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil || report.Compiled != 1 {
		t.Fatalf("inventory: %v", err)
	}
	for name, role := range map[string]string{"And": "aura_restriction_filter", "Other": "aura_restriction_filter", "ThisPermanent": "permanent_reference", "IsEnchantmentType": "aura_restriction_filter", "Aura": "subtype"} {
		found := false
		for _, signature := range report.Signatures {
			if signature.Name == name && signature.Role == role {
				found = true
			}
		}
		if !found {
			t.Errorf("%s lacks %s role", name, role)
		}
	}
}

func TestRONAuraRestrictionCopiesAndResolution(t *testing.T) {
	for _, name := range []string{"Anti-Magic Aura", "Consecrate Land"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			set := "LEG"
			hostCard := mage.NewCreature("Host", "", 2, 5)
			if name == "Consecrate Land" {
				set = "LEB"
				hostCard = mage.NewLand("Host")
			}
			hostCard.SetOwner(b.PlayerID())
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			card := repositoryCard(t, set, name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 3)
			a.ManaPool().Add(core.White, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
				t.Fatal("copy failed")
			}
			clone := g.Clone()
			for _, game := range []*mage.Game{g, clone} {
				game.ResolveTopOfStack()
				p := game.FindPermanent(host.ID())
				if len(p.Attachments) != 1 {
					t.Fatal("copy did not attach")
				}
				copyID := p.Attachments[0]
				if copyID == card.ID() || !game.FindPermanent(copyID).IsToken {
					t.Fatal("copy identity is not independent")
				}
				game.ResolveStack()
				if game.FindPermanent(card.ID()) != nil || len(game.GetPlayer(a.PlayerID()).Graveyard()) != 1 {
					t.Fatal("original ignored new restriction at resolution")
				}
				if game.FindPermanent(copyID) == nil {
					t.Fatal("copy excluded itself")
				}
			}
		})
	}
}

func TestRONAuraRestrictionExistingAuras(t *testing.T) {
	for _, name := range []string{"Anti-Magic Aura", "Consecrate Land"} {
		t.Run(name, func(t *testing.T) {
			g, a, b := ronGame()
			set := "LEG"
			hostCard := mage.NewCreature("Host", "", 2, 5)
			if name == "Consecrate Land" {
				set = "LEB"
				hostCard = mage.NewLand("Host")
			}
			host := g.PutOnBattlefield(hostCard, b.PlayerID())
			oldCard := mage.NewAura("Old", "", mage.WithKeyword(core.Indestructible), mage.WithCastTarget(mage.TargetPermanent(mage.AnyPermanent)))
			oldCard.SetOwner(b.PlayerID())
			old := g.PutOnBattlefield(oldCard, b.PlayerID())
			g.Attach(old.ID(), host.ID())
			card := repositoryCard(t, set, name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.Blue, 3)
			a.ManaPool().Add(core.White, 1)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{host.ID()}, 0); err != nil {
				t.Fatal(err)
			}
			g.ResolveStack()
			if g.FindPermanent(old.ID()) != nil || len(b.Graveyard()) != 1 || g.FindPermanent(card.ID()) == nil {
				t.Fatal("existing illegal Aura or granting exception")
			}
		})
	}
}

package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONStaticRedirectCards(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"ATQ", "Martyrs of Korlis"}, {"LEA", "Veteran Bodyguard"}} {
		t.Run(tc.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, tc.set, tc.name)
			p := g.PutOnBattlefield(card, a.PlayerID())
			source := g.PutOnBattlefield(mage.NewCreature("Source", "", 1, 20, mage.WithKeyword(core.AttrIsArtifact)), b.PlayerID())
			if tc.name == "Veteran Bodyguard" {
				g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrSummonSick)
				g.SetActivePlayerIndex(1)
				g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{source.ID()})
				g.ExecuteBlockers(nil)
			}
			g.DealDamageToPlayer(a, 2, source.ID())
			if a.Life() != 20 || g.FindPermanent(p.ID()).Damage != 2 {
				t.Fatal("matching damage was not redirected")
			}
		})
	}
}

func TestRONStaticRedirectArtifactSpell(t *testing.T) {
	g, a, b := ronGame()
	p := g.PutOnBattlefield(repositoryCard(t, "ATQ", "Martyrs of Korlis"), a.PlayerID())
	spell := permanentScopeFactory(t, "Artifact, Instant", `SpellActions(Targeted([TargetPlayer(AnyPlayer)],ActionList([SpellDealsDamage(ThisSpell,Integer(2),Player(Ref_TargetPlayer))])))`)()
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	if err := g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{a.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("copy failed")
	}
	clone := g.Clone()
	g.ResolveStack()
	clone.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.FindPermanent(p.ID()).Damage != 4 {
			t.Fatal("artifact spell/copy damage was not redirected")
		}
	}
}

func TestRONStaticRedirectLifecycle(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"ATQ", "Martyrs of Korlis"}, {"LEA", "Veteran Bodyguard"}} {
		for _, change := range []string{"live", "tap", "phase", "phase-cycle", "control", "remove-abilities", "departure", "return", "nonmatching", "source-return", "source-phase", "source-phase-cycle", "blocked", "before-blockers"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				card := repositoryCard(t, tc.set, tc.name)
				card.SetOwner(a.PlayerID())
				a.AddToHand(card)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || g.StackSize() != 0 {
					t.Fatal("unpaid cast changed state")
				}
				a.ManaPool().Add(core.White, 5)
				if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
					t.Fatal(err)
				}
				if g.FindPermanent(card.ID()) != nil || a.ManaPool().TotalMana() != 0 {
					t.Fatal("entry timing/payment")
				}
				g.ResolveStack()
				p := g.FindPermanent(card.ID())
				if p.CurrentPower(g) != map[string]int{"Martyrs of Korlis": 1, "Veteran Bodyguard": 2}[tc.name] || p.CurrentToughness(g) != map[string]int{"Martyrs of Korlis": 6, "Veteran Bodyguard": 5}[tc.name] {
					t.Fatal("metadata")
				}
				sourceCard := mage.NewCreature("Source", "", 1, 20, mage.WithKeyword(core.AttrIsArtifact), mage.WithKeyword(core.Lifelink))
				source := g.PutOnBattlefield(sourceCard, b.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{source.ID()})
				if change == "blocked" {
					blocker := g.PutOnBattlefield(mage.NewCreature("Blocker", "", 0, 20), a.PlayerID())
					g.ExecuteBlockers([]mage.BlockAssignment{{BlockerID: blocker.ID(), AttackerID: source.ID()}})
				} else if change != "before-blockers" {
					g.ExecuteBlockers(nil)
				}
				information := g.CaptureDamageSource(source.ID())
				switch change {
				case "tap":
					g.TapPermanent(p)
				case "phase":
					g.PhaseOut(p.ID())
				case "phase-cycle":
					g.PhaseIn(g.PhaseOut(p.ID()))
				case "control":
					g.AddControlEffect(mage.ControlEffectSpec{SourceID: source.ID(), TargetID: p.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
				case "remove-abilities":
					if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.RemoveAbilities(mage.This)), p.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				case "departure", "return":
					g.RemoveFromBattlefield(p)
					if change == "return" {
						g.PutOnBattlefield(card, a.PlayerID())
						g.TapPermanent(g.FindPermanent(card.ID()))
					}
				case "nonmatching":
					if tc.name == "Martyrs of Korlis" {
						g.MutablePermanent(source.ID()).RevokeBaseAttr(core.AttrIsArtifact)
					} else {
						g.RemoveFromCombat(source.ID())
					}
				case "source-return":
					g.RemoveFromBattlefield(source)
					g.PutOnBattlefield(sourceCard, b.PlayerID()).RevokeBaseAttr(core.AttrIsArtifact)
				case "source-phase":
					g.PhaseOut(source.ID())
				case "source-phase-cycle":
					g.PhaseIn(g.PhaseOut(source.ID()))
				}
				clone := g.Clone()
				matches := change == "live" || change == "phase-cycle" || change == "source-return" || change == "source-phase" || tc.name == "Martyrs of Korlis" && (change == "source-phase-cycle" || change == "blocked" || change == "before-blockers")
				for _, game := range []*mage.Game{g, clone} {
					game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 2, information)
					life := 18
					if matches {
						life = 20
					}
					if game.GetPlayer(a.PlayerID()).Life() != life || game.GetPlayer(b.PlayerID()).Life() != 22 {
						t.Fatal("recipient/source lifecycle or lifelink")
					}
					if matches && game.FindPermanent(p.ID()).Damage != 2 {
						t.Fatal("wrong damage recipient")
					}
					if change == "control" {
						game.DealDamageToPlayerFromSource(game.GetPlayer(b.PlayerID()), 1, information)
						if game.FindPermanent(p.ID()).Damage != 1 {
							t.Fatal("controller change")
						}
					}
				}
			})
		}
	}
}

func TestRONStaticRedirectCombat(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"ATQ", "Martyrs of Korlis"}, {"LEA", "Veteran Bodyguard"}} {
		g, a, b := ronGame()
		p := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
		source := g.PutOnBattlefield(mage.NewCreature("Attacker", "", 2, 10, mage.WithKeyword(core.AttrIsArtifact)), b.PlayerID())
		source.RevokeBaseAttr(core.AttrSummonSick)
		g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{source.ID()})
		g.ExecuteBlockers(nil)
		g.ExecuteCombatDamage()
		if a.Life() != 20 || g.FindPermanent(p.ID()).Damage != 2 {
			t.Fatal(tc.name, "native combat redirection")
		}
	}
}

func TestRONStaticRedirectInvalid(t *testing.T) {
	valid := `If(PermanentPassesFilter(ThisPermanent,IsUntapped),[ReplaceWouldDealDamage(DamageWouldBeDealtByAPermanentToRecipient(And([IsUnblocked,IsCardtype(Creature)]),Player(You)),[DealToCreatureOrPlaneswalkerInstead(ThisPermanent)])])`
	for _, rule := range []string{
		strings.Replace(valid, "Player(You)", "Player(Ref_TargetPlayer)", 1), strings.Replace(valid, "Player(You)", "Permanent(ThisPermanent)", 1),
		strings.Replace(valid, "Instead(ThisPermanent)", "Instead(HostPermanent)", 1), strings.Replace(valid, "Instead(ThisPermanent)", "Instead(Ref_TargetPermanent)", 1), strings.Replace(valid, "Instead(ThisPermanent)", "Instead(ThisPermanent())", 1),
		strings.Replace(valid, "IsUnblocked", "IsUnblocked()", 1), strings.Replace(valid, "IsUnblocked", "ControlledByPlayer(You)", 1), strings.Replace(valid, "And([IsUnblocked,IsCardtype(Creature)])", "And([])", 1),
		strings.Replace(valid, "[DealToCreatureOrPlaneswalkerInstead(ThisPermanent)]", "[]", 1), strings.Replace(valid, "[DealToCreatureOrPlaneswalkerInstead(ThisPermanent)]", "[DealToCreatureOrPlaneswalkerInstead(ThisPermanent),PreventThatDamage]", 1),
		strings.Replace(valid, "DamageWouldBeDealtByAPermanentToRecipient", "DamageWouldBeDealtByASourceToRecipient", 1),
		strings.TrimSuffix(strings.TrimPrefix(valid, `If(PermanentPassesFilter(ThisPermanent,IsUntapped),[`), "])"),
		`Activated(PayMana([]),ActionList([` + valid + `]))`,
	} {
		nodes, err := Parse(strings.NewReader(`Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[`+rule+`])`), "invalid-redirect.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatal("accepted", rule)
		}
	}
}

func TestRONStaticRedirectInventory(t *testing.T) {
	source := `Card(Name:"Redirect",Typeline:(Supertypes:[],Cardtypes:[Creature],Subtypes:[]),CardPT:(Power:1,Toughness:1),Rules:[If(PermanentPassesFilter(ThisPermanent,IsUntapped),[ReplaceWouldDealDamage(DamageWouldBeDealtByAPermanentToRecipient(IsUnblocked,Player(You)),[DealToCreatureOrPlaneswalkerInstead(ThisPermanent)])])])`
	report, err := InspectFS(fstest.MapFS{"a.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Issues) != 0 {
		t.Fatal(report.Issues)
	}
	for _, sig := range report.Signatures {
		if sig.Role == "unknown" {
			t.Fatal("unknown role", sig.Name)
		}
	}
	registry, err := LoadFS(fstest.MapFS{"a.ron": {Data: []byte(source)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	first, _ := registry.CreateCard("Redirect")
	second, _ := registry.CreateCard("Redirect")
	if first.ID() == second.ID() {
		t.Fatal("factories shared identity")
	}
	if err := LoadInto(registry, fstest.MapFS{"bad.ron": {Data: []byte(strings.Replace(source, "Player(You)", "Player(Ref_TargetPlayer)", 1))}}, "*.ron"); err == nil {
		t.Fatal("invalid collection loaded")
	}
	if _, err := registry.CreateCard("Redirect"); err != nil {
		t.Fatal("failed load replaced registry")
	}
}

func TestRONStaticRedirectPendingAbility(t *testing.T) {
	for _, tc := range []struct{ set, name, source string }{{"ATQ", "Martyrs of Korlis", "Rod of Ruin"}, {"LEA", "Veteran Bodyguard", "Prodigal Sorcerer"}} {
		for _, change := range []string{"live", "departure", "return", "phase", "copy", "clone"} {
			t.Run(tc.name+"/"+change, func(t *testing.T) {
				g, a, b := ronGame()
				p := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
				card := repositoryCard(t, "LEA", tc.source)
				source := g.PutOnBattlefield(card, b.PlayerID())
				source.RevokeBaseAttr(core.AttrSummonSick)
				if tc.source == "Prodigal Sorcerer" {
					source.GrantBaseAttr(core.Vigilance)
					g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{source.ID()})
					g.ExecuteBlockers(nil)
				} else {
					b.ManaPool().Add(core.Colorless, 3)
				}
				if err := g.ActivateAbilityByIndex(b.PlayerID(), source.ID(), 0, []uuid.UUID{a.PlayerID()}); err != nil {
					t.Fatal(err)
				}
				switch change {
				case "departure", "return":
					g.RemoveFromBattlefield(source)
					if change == "return" {
						g.PutOnBattlefield(card, b.PlayerID())
					}
				case "phase":
					g.PhaseOut(source.ID())
				case "copy":
					if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
						t.Fatal("ability copy")
					}
				}
				clone := g.Clone()
				g.ResolveStack()
				if clone.FindPermanent(p.ID()).Damage != 0 {
					t.Fatal("pending clone shared state")
				}
				clone.ResolveStack()
				want := 1
				if change == "copy" {
					want = 2
				}
				for _, game := range []*mage.Game{g, clone} {
					if game.GetPlayer(a.PlayerID()).Life() != 20 || game.FindPermanent(p.ID()).Damage != want {
						t.Fatal("pending ability source information")
					}
				}
			})
		}
	}
}

func TestRONStaticRedirectIndependentInstances(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"ATQ", "Martyrs of Korlis"}, {"LEA", "Veteran Bodyguard"}} {
		g, a, b := ronGame()
		first := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), a.PlayerID())
		second := g.PutOnBattlefield(repositoryCard(t, tc.set, tc.name), b.PlayerID())
		source := g.PutOnBattlefield(mage.NewCreature("Source", "", 1, 20, mage.WithKeyword(core.AttrIsArtifact)), b.PlayerID())
		source.RevokeBaseAttr(core.AttrSummonSick)
		g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{source.ID()})
		g.ExecuteBlockers(nil)
		g.DealSimultaneousDamage(mage.PlayerDamageAssignment(a, 1, g.CaptureDamageSource(source.ID())), mage.PlayerDamageAssignment(b, 2, g.CaptureDamageSource(source.ID())))
		if a.Life() != 20 || b.Life() != 20 || g.FindPermanent(first.ID()).Damage != 1 || g.FindPermanent(second.ID()).Damage != 2 {
			t.Fatal("instances shared source/controller")
		}
	}
}

func TestRONStaticRedirectReprintsAndHost(t *testing.T) {
	for _, set := range []string{"ATQ", "LEA", "LEB", "2ED", "3ED", "CED", "CEI"} {
		name := "Veteran Bodyguard"
		if set == "ATQ" {
			name = "Martyrs of Korlis"
		}
		card := repositoryCard(t, set, name)
		if err := mage.ValidateCardDefinition(name, card); err != nil {
			t.Fatal(set, err)
		}
	}
	g, a, b := ronGame()
	host := g.PutOnBattlefield(mage.NewCreature("Host", "", 1, 20), b.PlayerID())
	source := g.PutOnBattlefield(mage.NewArtifact("Source", ""), b.PlayerID())
	factory := staticPermanentFixture(t, "Enchantment", "Aura", `EnchantPermanent(IsCardtype(Creature)),Unless(PermanentPassesFilter(ThisPermanent,IsTapped),[ReplaceWouldDealDamage(DamageWouldBeDealtByASourceToRecipient(IsCardtype(Artifact),Player(You)),[DealToCreatureOrPlaneswalkerInstead(HostPermanent)])])`)
	card := factory()
	if err := mage.ValidateCardDefinition(card.Name(), card); err != nil {
		t.Fatal(err)
	}
	aura := g.PutOnBattlefield(card, a.PlayerID())
	g.Attach(aura.ID(), host.ID())
	g.DealDamageToPlayer(a, 2, source.ID())
	if a.Life() != 20 || g.FindPermanent(host.ID()).Damage != 2 {
		t.Fatal("conditional host context")
	}
	g.TapPermanent(aura)
	g.DealDamageToPlayer(a, 1, source.ID())
	if a.Life() != 19 {
		t.Fatal("negative static condition")
	}
}

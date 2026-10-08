package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONTemporaryRedirectGameplay(t *testing.T) {
	for _, state := range []string{"live", "control", "phase", "return", "illegal-target", "source-departure", "source-return-before", "source-phase-before"} {
		t.Run(state, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEG", "Shimian Night Stalker")
			card.SetOwner(a.PlayerID())
			absorber := g.PutOnBattlefield(card, a.PlayerID())
			attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "{R}", 1, 20), b.PlayerID())
			absorber.RevokeBaseAttr(core.AttrSummonSick)
			attacker.RevokeBaseAttr(core.AttrSummonSick)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), absorber.ID(), 0, []uuid.UUID{attacker.ID()}); err == nil {
				t.Fatal("nonattacking target accepted")
			}
			g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID()})
			if err := g.ActivateAbilityByIndex(a.PlayerID(), absorber.ID(), 0, []uuid.UUID{attacker.ID()}); err == nil || g.FindPermanent(absorber.ID()).Tapped {
				t.Fatal("failed payment changed source")
			}
			a.ManaPool().Add(core.Black, 1)
			if err := g.ActivateAbilityByIndex(a.PlayerID(), absorber.ID(), 0, []uuid.UUID{attacker.ID()}); err != nil {
				t.Fatal(err)
			}
			if len(g.WaitingRedirectionPermanentSources()) != 0 {
				t.Fatal("installed before resolution")
			}
			if state == "illegal-target" {
				g.GetCombat().Reset()
			}
			if state == "source-departure" || state == "source-return-before" {
				g.RemoveFromBattlefield(absorber)
				if state == "source-return-before" {
					g.PutOnBattlefield(card, a.PlayerID())
				}
			}
			if state == "source-phase-before" {
				g.PhaseOut(absorber.ID())
			}
			g.ResolveStack()
			if state == "illegal-target" || state == "source-departure" || state == "source-return-before" || state == "source-phase-before" {
				wantRefs := 2
				if state == "illegal-target" {
					wantRefs = 0
				}
				if len(g.WaitingRedirectionPermanentSources()) != wantRefs {
					t.Fatal("resolution lost original destination reference")
				}
				g.DealDamageToPlayer(a, 1, attacker.ID())
				if a.Life() != 19 {
					t.Fatal("invalid subject redirected")
				}
				return
			}
			saved := g.CaptureDamageSource(attacker.ID())
			switch state {
			case "control":
				g.AddControlEffect(mage.ControlEffectSpec{SourceID: b.PlayerID(), TargetID: absorber.ID(), ControllerID: b.PlayerID(), Duration: core.Indefinite})
			case "phase":
				g.PhaseOut(absorber.ID())
			case "return":
				g.RemoveFromBattlefield(absorber)
				g.PutOnBattlefield(card, a.PlayerID())
			}
			baseline := g.Clone()
			for _, game := range []*mage.Game{g, g.Clone()} {
				refs := game.WaitingRedirectionPermanentSources()
				if len(refs) != 2 {
					t.Fatalf("waiting references=%d want 2", len(refs))
				}
				game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 1, saved)
				want := 20
				if state == "phase" || state == "return" {
					want = 19
				}
				if game.GetPlayer(a.PlayerID()).Life() != want {
					t.Fatal("wrong saved recipient or destination")
				}
				if want == 20 {
					if game.FindPermanent(absorber.ID()).Damage != 1 {
						t.Fatal("damage did not reach absorber")
					}
					game.DealDamageToPlayerFromSource(game.GetPlayer(a.PlayerID()), 1, saved)
					if game.FindPermanent(absorber.ID()).Damage != 2 {
						t.Fatal("rule consumed after first damage")
					}
				}
				game.GetEffects().ClearReplacementsEndOfTurn()
				if len(game.WaitingRedirectionPermanentSources()) != 0 {
					t.Fatal("cleanup retained rule")
				}
			}
			if len(baseline.WaitingRedirectionPermanentSources()) != 2 {
				t.Fatal("clone shared cleanup")
			}
		})
	}
}

func TestRONTemporaryRedirectInvalid(t *testing.T) {
	const valid = `CreateReplaceWouldDealDamageUntil(DamageWouldBeDealtByPermanentToRecipient(Ref_TargetPermanent, Player(You)), [DealToCreatureOrPlaneswalkerInstead(ThisPermanent)], UntilEndOfTurn)`
	for _, expr := range []string{
		strings.Replace(valid, "UntilEndOfTurn", "UntilEndOfCombat", 1),
		strings.Replace(valid, "Player(You)", "Permanent(ThisPermanent)", 1),
		strings.Replace(valid, "ThisPermanent", "Ref_TargetPermanent_2", 1),
		strings.Replace(valid, "[DealToCreatureOrPlaneswalkerInstead(ThisPermanent)]", "[]", 1),
		strings.Replace(valid, "Player(You)", "Player(You, You)", 1),
		strings.Replace(valid, "Ref_TargetPermanent", "TheChosenDamageSource", 1),
	} {
		record := parseTemporaryRedirectCard(t, `Card(Name: "Invalid", Typeline: (Supertypes: [], Cardtypes: [Creature], Subtypes: []), Rules: [Activated(TapPermanent(ThisPermanent), Targeted([TargetPermanent(IsCardtype(Creature))], ActionList([`+expr+`])))], CardPT: (Power: 1, Toughness: 1))`)
		if _, err := Compile(record); err == nil {
			t.Fatal("invalid declaration accepted", expr)
		}
	}
}

func parseTemporaryRedirectCard(t *testing.T, text string) Node {
	t.Helper()
	nodes, err := Parse(strings.NewReader(text), "redirect.ron")
	if err != nil {
		t.Fatal(err)
	}
	return nodes[0]
}

func TestRONTemporaryRedirectCopyAndInstances(t *testing.T) {
	g, a, b := ronGame()
	first := g.PutOnBattlefield(repositoryCard(t, "LEG", "Shimian Night Stalker"), a.PlayerID())
	second := g.PutOnBattlefield(repositoryCard(t, "LEG", "Shimian Night Stalker"), a.PlayerID())
	attacker := g.PutOnBattlefield(mage.NewCreature("Attacker", "{R}", 1, 20), b.PlayerID())
	for _, p := range []*mage.Permanent{first, second, attacker} {
		p.RevokeBaseAttr(core.AttrSummonSick)
	}
	g.ExecuteAttackers(b.PlayerID(), []uuid.UUID{attacker.ID()})
	a.ManaPool().Add(core.Black, 2)
	for _, p := range []*mage.Permanent{first, second} {
		if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, []uuid.UUID{attacker.ID()}); err != nil {
			t.Fatal(err)
		}
	}
	if g.CopyStackObjectDirect(g.StackPeek(), b.PlayerID(), false) == nil {
		t.Fatal("ability copy failed")
	}
	g.ResolveStack()
	if len(g.WaitingRedirectionPermanentSources()) != 3 || a.ManaPool().TotalMana() != 0 {
		t.Fatal("instances shared identities or copy paid costs")
	}
	g.DealDamageToPlayer(b, 1, attacker.ID())
	if b.Life() != 20 || g.FindPermanent(second.ID()).Damage != 1 {
		t.Fatal("copy did not save its own controller")
	}
	g.RemoveFromBattlefield(second)
	g.DealDamageToPlayer(a, 1, attacker.ID())
	if a.Life() != 20 || g.FindPermanent(first.ID()).Damage != 1 {
		t.Fatal("independent original rule lost its binding")
	}
}

package ron

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronPlayerRedirectChooser struct {
	*mage.BasePlayer
	accept bool
	offers int
}

func (p *ronPlayerRedirectChooser) ChooseMayAbility(string) bool { p.offers++; return p.accept }

func TestRONTemporaryPlayerRedirectBlood(t *testing.T) {
	a := &ronPlayerRedirectChooser{BasePlayer: mage.NewBasePlayer("A"), accept: true}
	b := mage.NewBasePlayer("B")
	g := mage.NewGame(a, b)
	g.SetStep(core.PrecombatMain)
	g.SetTurn(1)
	card := repositoryCard(t, "DRK", "Blood of the Martyr")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	victim := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), b.PlayerID())
	source := g.PutOnBattlefield(mage.NewCreature("Source", "", 1, 20), b.PlayerID())
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || g.StackSize() != 0 {
		t.Fatal("unpaid cast")
	}
	a.ManaPool().Add(core.White, 3)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	g.DealDamageToPermanent(victim, 1, source.ID())
	if victim.Damage != 1 || a.Life() != 20 {
		t.Fatal("early redirect")
	}
	g.ResolveStack()
	g.DealDamageToPermanent(victim, 2, source.ID())
	if a.Life() != 18 || victim.Damage != 1 {
		t.Fatal("damage not redirected")
	}
	g.GetEffects().ClearReplacementsEndOfTurn()
	g.DealDamageToPermanent(victim, 1, source.ID())
	if a.Life() != 18 || victim.Damage != 2 {
		t.Fatal("cleanup retained rule")
	}
}

func TestRONTemporaryPlayerRedirectReverberation(t *testing.T) {
	g, b, a := ronGame()
	spell := repositoryCard(t, "LEB", "Earthquake")
	spell.SetOwner(b.PlayerID())
	b.AddToHand(spell)
	b.ManaPool().Add(core.Red, 1)
	b.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(b.PlayerID(), spell.ID(), nil, 2); err != nil {
		t.Fatal(err)
	}
	target := g.StackPeek().ID
	card := repositoryCard(t, "LEG", "Reverberation")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Blue, 2)
	a.ManaPool().Add(core.Colorless, 2)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target}, 0); err != nil {
		t.Fatal(err)
	}
	victim := g.PutOnBattlefield(mage.NewCreature("Victim", "", 1, 20), a.PlayerID())
	clone := g.Clone()
	clone.ResolveStack()
	g.ResolveStack()
	for _, game := range []*mage.Game{g, clone} {
		if game.GetPlayer(a.PlayerID()).Life() != 20 || game.GetPlayer(b.PlayerID()).Life() != 14 || game.FindPermanent(victim.ID()).Damage != 0 {
			t.Fatal("Earthquake packets not redirected to spell controller")
		}
	}
}

func TestRONTemporaryPlayerRedirectInvalid(t *testing.T) {
	valid := `CreateReplaceWouldDealDamageUntil(DamageWouldBeDealtToARecipient(APermanent(IsCardtype(Creature))),[MayActions([DealToPlayerInstead(You)])],UntilEndOfTurn)`
	for _, expr := range []string{
		strings.Replace(valid, "UntilEndOfTurn", "UntilEndOfCombat", 1),
		strings.Replace(valid, "You", "WouldDealDamage_ControllerOfDamageSource", 1),
		strings.Replace(valid, "MayActions([DealToPlayerInstead(You)])", "DealToPlayerInstead(You)", 1),
		strings.Replace(valid, "Creature", "Creature()", 1),
		strings.Replace(valid, "You", "You()", 1),
		strings.Replace(valid, "APermanent(IsCardtype(Creature))", "Player(You)", 1),
		`CreateReplaceWouldDealDamageUntil(DamageWouldBeDealtBySpell(Ref_TargetSpell),[DealToPlayerInstead(WouldDealDamage_ControllerOfDamageSource)],UntilEndOfTurn)`,
	} {
		n := parseTemporaryRedirectCard(t, `Card(Name:"Bad",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(ActionList([`+expr+`]))])`)
		if _, err := Compile(n); err == nil {
			t.Fatal("invalid accepted", expr)
		}
	}
}

func TestRONTemporaryPlayerRedirectBloodChoicesAndCopies(t *testing.T) {
	for _, state := range []string{"decline", "copy", "late-entry", "phase", "player-prevention"} {
		t.Run(state, func(t *testing.T) {
			a := &ronPlayerRedirectChooser{BasePlayer: mage.NewBasePlayer("A"), accept: state != "decline"}
			b := &ronPlayerRedirectChooser{BasePlayer: mage.NewBasePlayer("B"), accept: true}
			g := mage.NewGame(a, b)
			g.SetStep(core.PrecombatMain)
			g.SetTurn(1)
			card := repositoryCard(t, "DRK", "Blood of the Martyr")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			a.ManaPool().Add(core.White, 3)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
				t.Fatal(err)
			}
			if state == "copy" {
				original := g.StackPeek()
				copied := g.CopyStackObjectDirect(original, b.PlayerID(), false)
				if copied == nil {
					t.Fatal("copied failed")
				}
				g.ResolveStackObject(copied)
				g.GetStack().RemoveByID(copied.ID)
				g.GetStack().RemoveByID(original.ID)
			}
			if state != "copy" {
				g.ResolveStack()
			}
			source := g.PutOnBattlefield(mage.NewCreature("Source", "", 1, 20), b.PlayerID())
			victim := g.PutOnBattlefield(repositoryCard(t, "LEB", "Air Elemental"), b.PlayerID())
			if state == "phase" {
				g.PhaseOut(victim.ID())
			}
			if state == "player-prevention" {
				g.AddPreventionShield(a.PlayerID(), 1)
			}
			g.DealDamageToPermanent(victim, 2, source.ID())
			wantA, wantB := 18, 20
			if state == "decline" || state == "phase" {
				wantA = 20
			}
			if state == "copy" {
				wantA, wantB = 20, 18
			}
			if state == "player-prevention" {
				wantA = 19
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("lives=%d/%d want %d/%d", a.Life(), b.Life(), wantA, wantB)
			}
			if state == "decline" && victim.Damage != 2 {
				t.Fatal("decline prevented damage")
			}
		})
	}
}

func TestRONTemporaryPlayerRedirectSpellIdentity(t *testing.T) {
	for _, state := range []string{"copy-source", "remove-target", "controller", "cleanup", "copy-rule"} {
		t.Run(state, func(t *testing.T) {
			g, b, a := ronGame()
			source := repositoryCard(t, "LEB", "Earthquake")
			source.SetOwner(b.PlayerID())
			b.AddToHand(source)
			b.ManaPool().Add(core.Red, 3)
			if err := g.CastSpellByID(b.PlayerID(), source.ID(), nil, 2); err != nil {
				t.Fatal(err)
			}
			target := g.StackPeek()
			card := repositoryCard(t, "LEG", "Reverberation")
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID}, 0); err == nil {
				t.Fatal("unpaid rule")
			}
			a.ManaPool().Add(core.Blue, 4)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{target.ID}, 0); err != nil {
				t.Fatal(err)
			}
			rule := g.StackPeek()
			if state == "copy-rule" {
				copied := g.CopyStackObjectDirect(rule, b.PlayerID(), false)
				if copied == nil {
					t.Fatal("copied rule")
				}
				g.ResolveStackObject(copied)
				g.GetStack().RemoveByID(copied.ID)
			}
			if state == "remove-target" {
				g.GetStack().RemoveByID(target.ID)
			}
			g.ResolveStackObject(rule)
			g.GetStack().RemoveByID(rule.ID)
			if state == "controller" {
				target.Controller = a.PlayerID()
			}
			if state == "cleanup" {
				g.GetEffects().ClearReplacementsEndOfTurn()
			}
			if state == "copy-source" {
				copied := g.CopyStackObjectDirect(target, b.PlayerID(), false)
				if copied == nil {
					t.Fatal("copied source")
				}
				g.ResolveStackObject(copied)
				g.GetStack().RemoveByID(copied.ID)
			}
			g.ResolveStack()
			wantA, wantB := 20, 16
			switch state {
			case "copy-source":
				wantA, wantB = 18, 14
			case "remove-target":
				wantA, wantB = 20, 20
			case "controller":
				wantA, wantB = 16, 20
			case "cleanup":
				wantA, wantB = 18, 18
			}
			if a.Life() != wantA || b.Life() != wantB {
				t.Fatalf("lives=%d/%d want %d/%d", a.Life(), b.Life(), wantA, wantB)
			}
		})
	}
}

func TestRONTemporaryPlayerRedirectInventory(t *testing.T) {
	for _, tc := range []struct{ set, name string }{{"DRK", "Blood of the Martyr"}, {"LEG", "Reverberation"}} {
		record := repositoryRecord(t, tc.set, tc.name, tc.set+".ron")
		factory, err := Compile(record)
		if err != nil {
			t.Fatal(err)
		}
		if err := mage.ValidateCardDefinition(tc.name, factory()); err != nil {
			t.Fatal(err)
		}
	}
	src := `Card(Name:"Inventory",Typeline:(Supertypes:[],Cardtypes:[Instant],Subtypes:[]),Rules:[SpellActions(Targeted([TargetSpell(IsCardtype(Sorcery))],ActionList([CreateReplaceWouldDealDamageUntil(DamageWouldBeDealtBySpell(Ref_TargetSpell),[DealToPlayerInstead(WouldDealDamage_ControllerOfDamageSource)],UntilEndOfTurn)])))])`
	report, err := InspectFS(fstest.MapFS{"rule.ron": {Data: []byte(src)}}, "*.ron")
	if err != nil {
		t.Fatal(err)
	}
	for name, role := range map[string]string{"DamageWouldBeDealtBySpell": "temporary_redirection_event", "Ref_TargetSpell": "spell_reference", "DealToPlayerInstead": "temporary_redirection_operation", "WouldDealDamage_ControllerOfDamageSource": "redirection_player_reference"} {
		found := false
		for _, entry := range report.Signatures {
			if entry.Name == name && entry.Role == role {
				found = true
				if entry.Support != "unverified" {
					t.Fatal("inventory claims gameplay")
				}
			}
		}
		if !found {
			t.Fatal("missing role", name, role)
		}
	}
}

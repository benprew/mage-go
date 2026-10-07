package ron

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

type ronSpellIdentityProbe struct {
	source mage.DamageSource
	seen   int
}

func (*ronSpellIdentityProbe) SourceID() uuid.UUID             { return uuid.Nil }
func (*ronSpellIdentityProbe) GetDuration() core.Duration      { return core.Indefinite }
func (*ronSpellIdentityProbe) IsActive(mage.GameReader) bool   { return true }
func (p *ronSpellIdentityProbe) Clone() mage.ReplacementEffect { clone := *p; return &clone }
func (p *ronSpellIdentityProbe) Matches(action mage.Action, _ mage.GameReader) bool {
	damage, ok := action.(*mage.DamageToPlayerAction)
	return ok && damage.DamageSource().SameObject(p.source)
}
func (p *ronSpellIdentityProbe) Replace(action mage.Action, _ *mage.Game) mage.Action {
	p.seen++
	return action
}

func TestRONSpellDamageIdentityNativeResolution(t *testing.T) {
	for _, spec := range []struct {
		name                                      string
		color                                     core.Color
		mana, x, selfDamage, enemyDamage, packets int
	}{
		{"Lightning Bolt", core.Red, 1, 0, 0, 3, 1},
		{"Psionic Blast", core.Blue, 3, 0, 2, 4, 2},
		{"Earthquake", core.Red, 3, 2, 2, 2, 2},
		{"Hurricane", core.Green, 3, 2, 2, 2, 2},
	} {
		t.Run(spec.name, func(t *testing.T) {
			g, a, b := ronGame()
			card := repositoryCard(t, "LEB", spec.name)
			card.SetOwner(a.PlayerID())
			a.AddToHand(card)
			var targets []uuid.UUID
			if spec.x == 0 {
				targets = []uuid.UUID{b.PlayerID()}
			}
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, spec.x); err == nil || len(a.Hand()) != 1 || g.StackSize() != 0 {
				t.Fatal("unpaid casting changed live state")
			}
			a.ManaPool().Add(spec.color, spec.mana)
			if err := g.CastSpellByID(a.PlayerID(), card.ID(), targets, spec.x); err != nil {
				t.Fatal(err)
			}
			obj := g.StackPeek()
			source := g.CaptureSpellDamageSource(obj.ID)
			probe := &ronSpellIdentityProbe{source: source}
			g.AddReplacementEffect(probe)
			clone := g.Clone()
			g.ResolveStack()
			if probe.seen != spec.packets || a.Life() != 20-spec.selfDamage || b.Life() != 20-spec.enemyDamage || len(a.Graveyard()) != 1 {
				t.Fatalf("native damage identity: matched %d packets; life %d/%d", probe.seen, a.Life(), b.Life())
			}
			if clone.GetPlayer(a.PlayerID()).Life() != 20 || clone.GetPlayer(b.PlayerID()).Life() != 20 || clone.StackSize() != 1 {
				t.Fatal("resolution changed the pending clone")
			}
			clone.ResolveStack()
			if clone.GetPlayer(a.PlayerID()).Life() != a.Life() || clone.GetPlayer(b.PlayerID()).Life() != b.Life() {
				t.Fatal("cloned spell damage changed")
			}
			if g.CaptureSpellDamageSource(obj.ID).Valid() || source.StackID() != obj.ID {
				t.Fatal("live capture revived the departed spell or lost its saved identity")
			}
		})
	}
}

func TestRONSpellDamageIdentityCopyAndRecast(t *testing.T) {
	g, a, b := ronGame()
	card := repositoryCard(t, "LEB", "Lightning Bolt")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	original := g.StackPeek()
	source := g.CaptureSpellDamageSource(original.ID)
	probe := &ronSpellIdentityProbe{source: source}
	g.AddReplacementEffect(probe)
	copied := g.CopyStackObjectDirect(original, b.PlayerID(), false)
	if copied == nil || source.SameObject(g.CaptureSpellDamageSource(copied.ID)) {
		t.Fatal("copied spell reused the original damage identity")
	}
	if !g.ChangeObjectColor(original.ID, core.Green) {
		t.Fatal("recolor failed")
	}
	g.ResolveStack()
	if probe.seen != 1 || b.Life() != 14 || source.HasColor(core.Green) {
		t.Fatal("copy/original identity or frozen information was lost")
	}
	a.RemoveFromGraveyard(card.ID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{b.PlayerID()}, 0); err != nil {
		t.Fatal(err)
	}
	if source.SameObject(g.CaptureSpellDamageSource(g.StackPeek().ID)) {
		t.Fatal("recast matched old source identity")
	}
	g.ResolveStack()
	if probe.seen != 1 || b.Life() != 11 {
		t.Fatal("later cast used or changed the old identity")
	}
}

func TestRONSpellDamageIdentityEventSnapshot(t *testing.T) {
	g, a, b := ronGame()
	saved := mage.BindDamageSource("spell event source")
	observer := mage.NewCreature("Observer", "", 1, 5, mage.WithAbility(mage.WhenEvent(
		mage.ReceivesDamage(mage.This, mage.BindNumber("amount")).WithSource(saved),
		mage.DealDamageFrom(mage.FromDamageSource(saved), mage.You, mage.Fixed(1)))))
	if err := mage.ValidateCardDefinition(observer.Name(), observer); err != nil {
		t.Fatal(err)
	}
	observer.SetOwner(b.PlayerID())
	permanent := g.PutOnBattlefield(observer, b.PlayerID())
	card := repositoryCard(t, "LEB", "Lightning Bolt")
	card.SetOwner(a.PlayerID())
	a.AddToHand(card)
	a.ManaPool().Add(core.Red, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), []uuid.UUID{permanent.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	obj := g.StackPeek()
	probe := &ronSpellIdentityProbe{source: g.CaptureSpellDamageSource(obj.ID)}
	g.AddReplacementEffect(probe)
	g.ResolveTopOfStack()
	if permanent.Damage != 3 || b.Life() != 20 || len(a.Graveyard()) != 1 || g.CaptureSpellDamageSource(obj.ID).Valid() {
		t.Fatal("damage event or spell departure timing")
	}
	g.PutTriggersOnStack()
	if g.StackSize() != 1 {
		t.Fatal("damage event trigger was not queued")
	}
	clone := g.Clone()
	g.ResolveStack()
	if probe.seen != 1 || b.Life() != 19 || clone.GetPlayer(b.PlayerID()).Life() != 20 {
		t.Fatal("event snapshot lost spell identity or changed clone")
	}
	clone.ResolveStack()
	if clone.GetPlayer(b.PlayerID()).Life() != 19 {
		t.Fatal("cloned trigger lost its departed spell source")
	}
}

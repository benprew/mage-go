package mage

import (
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestPermanentEnchantGrantForwardLink(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	guest := NewCreature("Guest", "", 3, 4)
	guest.SetOwner(b.PlayerID())
	b.AddToGraveyard(guest)
	card := NewAura("Aura", "{B}", WithCastTarget(GraveyardCreatureEnchantTarget()))
	card.SetOwner(a.PlayerID())
	aura := g.PutOnBattlefield(card, a.PlayerID())
	other := g.PutOnBattlefield(NewCreature("Other", "", 2, 2), b.PlayerID())
	hosts := PermanentsEnteredWithSource(This, MatchingPermanents(IsCreature))
	effect := Until(core.Indefinite, RemoveGraveyardCreatureEnchant(This), GrantPermanentEnchant(This, hosts))
	if err := ApplyEffect(g, effect, aura.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if auraHostIsLegal(card, other, g) {
		t.Fatal("empty forward query permitted an unrelated creature")
	}
	if len(card.CastTargets()) != 1 || graveyardEnchantTarget(card) == nil {
		t.Fatal("printed enchant changed")
	}
	if err := ApplyEffect(g, ReturnCardToBattlefield(EnchantedGraveyardCard(This), You, RecordEntryFrom(This)), aura.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	host := g.FindPermanent(guest.ID())
	if host == nil || !auraHostIsLegal(card, host, g) {
		t.Fatal("live enchant did not find the later entry")
	}
	host.GrantBaseAttr(core.Shroud)
	host.GrantBaseAttr(core.Hexproof)
	g.Attach(aura.ID(), host.ID())
	if aura.AttachedTo != host.ID() || aura.enchantedCard.id != uuid.Nil {
		t.Fatal("untargeted attachment failed")
	}
	g.Attach(aura.ID(), other.ID())
	if aura.AttachedTo != host.ID() {
		t.Fatal("unrelated creature replaced the linked host")
	}
	g.CheckStateBasedActions()
	if g.FindPermanent(aura.ID()) == nil {
		t.Fatal("state action used the printed graveyard enchant")
	}
	clone := g.Clone()
	g.PhaseOut(aura.ID())
	g.PhaseIn([]uuid.UUID{aura.ID()})
	g.ApplyContinuousEffects()
	if !auraHostIsLegal(card, g.FindPermanent(host.ID()), g) {
		t.Fatal("phase cycle lost the grant")
	}
	g.RemoveFromBattlefield(g.FindPermanent(host.ID()))
	g.PutOnBattlefield(guest, b.PlayerID())
	if auraHostIsLegal(card, g.FindPermanent(guest.ID()), g) {
		t.Fatal("returned host inherited the relation")
	}
	if !auraHostIsLegal(card, clone.FindPermanent(host.ID()), clone) {
		t.Fatal("original changed clone legality")
	}
}

func TestPermanentEnchantGrantDeclarations(t *testing.T) {
	for _, modification := range []ContinuousModification{
		GrantPermanentEnchant(nil, MatchingPermanents(AnyPermanent)),
		GrantPermanentEnchant(This, nil),
		GrantPermanentEnchant(This, grantRecipientSet{From(BindPermanent("donor"))}),
		RemoveGraveyardCreatureEnchant(nil),
		RemoveGraveyardCreatureEnchant(From(BindPermanent("missing"))),
	} {
		card := NewArtifact("Bad", "", WithAbility(Activate(Costs(), Until(core.Indefinite, modification))))
		if ValidateCardDefinition(card.Name(), card) == nil {
			t.Fatal("invalid declaration accepted")
		}
	}
}

func TestPermanentEnchantGrantLifecycle(t *testing.T) {
	for _, state := range []string{"normal", "expiry", "recipient-return", "donor-return", "phase", "clone", "remove-after", "remove-before", "other-enchant", "protection"} {
		t.Run(state, func(t *testing.T) {
			a, b := NewBasePlayer("A"), NewBasePlayer("B")
			g := NewGame(a, b)
			host := g.PutOnBattlefield(NewLand("Land"), b.PlayerID())
			creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), b.PlayerID())
			donor := g.PutOnBattlefield(NewArtifact("Donor", ""), a.PlayerID())
			options := []CardOption{WithCastTarget(CreatureEnchantTarget())}
			if state == "other-enchant" {
				options = append(options, func(c *BaseCard) { c.castTargets = append(c.castTargets, TargetPermanent(IsArtifact)) })
			}
			card := NewAura("Aura", "{B}", options...)
			card.SetOwner(a.PlayerID())
			aura := g.PutOnBattlefield(card, a.PlayerID())
			recipient := BindPermanent("recipient")
			ctx := &EffectContext{Game: g, SourceID: donor.ID(), Controller: a.PlayerID()}
			g.resolution.permanentBindings = map[*PermanentBinding]capturedPermanent{recipient: {aura.ID(), aura.incarnationID}}
			effect := Until(core.EndOfTurn, RemoveCreatureEnchant(From(recipient)), GrantPermanentEnchant(From(recipient), MatchingPermanents(IsLand)))
			if err := effect.Apply(ctx); err != nil {
				t.Fatal(err)
			}
			g.ApplyContinuousEffects()
			clone := g.Clone()
			switch state {
			case "expiry":
				g.effects.RemoveEndOfTurn()
			case "recipient-return":
				g.RemoveFromBattlefield(aura)
				aura = g.PutOnBattlefield(card, a.PlayerID())
			case "donor-return":
				g.RemoveFromBattlefield(donor)
				g.PutOnBattlefield(donor.Card, b.PlayerID())
			case "phase":
				g.PhaseIn(g.PhaseOut(aura.ID()))
			case "clone":
				clone.effects.RemoveEndOfTurn()
				clone.ApplyContinuousEffects()
				if auraHostIsLegal(card, clone.FindPermanent(host.ID()), clone) {
					t.Fatal("clone retained expired grant")
				}
			case "remove-after", "remove-before":
				if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveAbilities(This)), aura.ID(), a.PlayerID(), nil); err != nil {
					t.Fatal(err)
				}
				if state == "remove-before" {
					if err := ApplyEffect(g, Until(core.EndOfTurn, GrantPermanentEnchant(This, MatchingPermanents(IsLand))), aura.ID(), a.PlayerID(), nil); err != nil {
						t.Fatal(err)
					}
				}
			case "protection":
				host.Card.AddAbility(ProtectionFromColors(core.Black))
				host.RuntimeAbilities = append(host.RuntimeAbilities, ProtectionFromColors(core.Black))
			}
			g.ApplyContinuousEffects()
			aura = g.FindPermanent(aura.ID())
			wantLand := state != "expiry" && state != "recipient-return" && state != "other-enchant"
			if auraHostIsLegal(card, g.FindPermanent(host.ID()), g) != wantLand {
				t.Fatal("wrong computed land restriction")
			}
			wantCreature := state == "expiry" || state == "recipient-return" || state == "remove-after"
			if auraHostIsLegal(card, g.FindPermanent(creature.ID()), g) != wantCreature {
				t.Fatal("wrong computed creature restriction")
			}
			g.Attach(aura.ID(), host.ID())
			if (aura.AttachedTo == host.ID()) != (wantLand && state != "protection") {
				t.Fatal("attachment ignored computed enchant or protection")
			}
			if state != "other-enchant" && state != "protection" {
				if auraHostIsLegal(card, clone.FindPermanent(host.ID()), clone) != (state != "clone") {
					t.Fatal("original changed clone enchant state")
				}
			}
		})
	}
}

func TestPermanentEnchantGrantIntersections(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	land := g.PutOnBattlefield(NewLand("Land"), a.PlayerID())
	creature := g.PutOnBattlefield(NewCreature("Creature", "", 2, 2), a.PlayerID())
	card := NewAura("Aura", "", WithCastTarget(CreatureEnchantTarget()))
	p := g.PutOnBattlefield(card, a.PlayerID())
	if err := ApplyEffect(g, Until(core.EndOfTurn, GrantPermanentEnchant(This, MatchingPermanents(IsLand))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if auraHostIsLegal(card, land, g) || auraHostIsLegal(card, creature, g) {
		t.Fatal("enchant abilities did not intersect")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveCreatureEnchant(This)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !auraHostIsLegal(card, land, g) {
		t.Fatal("fixed removal erased an acquired enchant")
	}
	if err := ApplyEffect(g, Until(core.EndOfTurn, GrantPermanentEnchant(This, MatchingPermanents(IsArtifact))), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if auraHostIsLegal(card, land, g) {
		t.Fatal("acquired enchant abilities did not intersect")
	}
}

func TestPermanentEnchantGrantRecipientContext(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	left := g.PutOnBattlefield(NewCreature("Left", "", 2, 2), a.PlayerID())
	right := g.PutOnBattlefield(NewCreature("Right", "", 2, 2), b.PlayerID())
	card := NewAura("Aura", "", WithCastTarget(CreatureEnchantTarget()))
	aura := g.PutOnBattlefield(card, a.PlayerID())
	if err := ApplyEffect(g, Until(core.Indefinite, GrantPermanentEnchant(This, ControlledPermanents(IsCreature))), aura.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !auraHostIsLegal(card, left, g) || auraHostIsLegal(card, right, g) {
		t.Fatal("enchant used the resolving controller")
	}
	if err := ApplyEffect(g, Until(core.Indefinite, ChangeController(This, You)), aura.ID(), b.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if auraHostIsLegal(card, g.FindPermanent(left.ID()), g) || !auraHostIsLegal(card, g.FindPermanent(right.ID()), g) {
		t.Fatal("enchant did not read the recipient's current controller")
	}
}

func TestPermanentEnchantGrantGraveyardRemovalAndExpiry(t *testing.T) {
	a, b := NewBasePlayer("A"), NewBasePlayer("B")
	g := NewGame(a, b)
	guest := NewCreature("Guest", "", 2, 2)
	guest.SetOwner(b.PlayerID())
	b.AddToGraveyard(guest)
	card := NewAura("Aura", "", WithCastTarget(GraveyardCreatureEnchantTarget()))
	aura := g.PutOnBattlefield(card, a.PlayerID())
	if err := ApplyEffect(g, Until(core.EndOfTurn, RemoveGraveyardCreatureEnchant(This), GrantPermanentEnchant(This, MatchingPermanents(IsCreature))), aura.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if _, ok := g.graveyardEnchantHost(aura); ok {
		t.Fatal("permanent enchant still permits a graveyard host")
	}
	ctx := &EffectContext{Game: g, SourceID: aura.ID(), Controller: a.PlayerID()}
	if c, ok := EnchantedGraveyardCard(This).resolveCard(ctx); !ok || c.ID() != guest.ID() {
		t.Fatal("removal erased the saved attachment")
	}
	clone := g.Clone()
	g.effects.RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	if _, ok := g.graveyardEnchantHost(g.FindPermanent(aura.ID())); !ok {
		t.Fatal("expiry did not restore graveyard enchant")
	}
	clone.CheckStateBasedActions()
	if clone.FindPermanent(aura.ID()) != nil || g.FindPermanent(aura.ID()) == nil {
		t.Fatal("state action or clone isolation ignored removed graveyard enchant")
	}
}

func TestPermanentEnchantGrantBindingIsolation(t *testing.T) {
	target := DeclarePermanentTarget("donor target", IsCreature)
	card := NewArtifact("Invalid", "", WithAbility(Activate(Costs(), Until(core.Indefinite, GrantPermanentEnchant(This, grantRecipientSet{target})), WithTargets(target))))
	if ValidateCardDefinition(card.Name(), card) == nil {
		t.Fatal("granted enchant read a donor target")
	}
}

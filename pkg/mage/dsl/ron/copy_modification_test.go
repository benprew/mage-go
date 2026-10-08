package ron

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/core"
)

func TestRONCopyTypeModificationGameplay(t *testing.T) {
	g, a, b := ronGame()
	chooser := &ronEntryCopyPlayer{Player: a, accept: true}
	g.SetPlayerAt(0, chooser)
	donor := g.PutOnBattlefield(repositoryCard(t, "LEB", "Juggernaut"), b.PlayerID())
	donor.GrantBaseAttr(core.Shroud)
	card := repositoryCard(t, "LEB", "Copy Artifact")
	a.AddToHand(card)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err == nil || chooser.offers != 0 || len(a.Hand()) != 1 {
		t.Fatal("failed payment changed entry")
	}
	a.ManaPool().Add(core.Blue, 1)
	a.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(a.PlayerID(), card.ID(), nil, 0); err != nil {
		t.Fatal(err)
	}
	if chooser.offers != 0 {
		t.Fatal("copy choice at announcement")
	}
	clone := g.Clone()
	decline := &ronEntryCopyPlayer{Player: clone.GetPlayer(a.PlayerID()), accept: false}
	clone.SetPlayerAt(0, decline)
	g.ResolveStack()
	p := g.FindPermanent(card.ID())
	if p == nil || p.Name() != "Juggernaut" || !p.HasType(core.TypeArtifact) || !p.HasType(core.TypeCreature) || !p.HasType(core.TypeEnchantment) || p.CurrentPower(g) != 5 || p.CurrentToughness(g) != 3 || p.HasAttr(core.Shroud) || p.Card.ManaCost().CMC() != 4 {
		t.Fatal("copy exception or characteristics")
	}
	wall := g.PutOnBattlefield(repositoryCard(t, "LEB", "Wall of Wood"), b.PlayerID())
	if !p.HasAttr(core.MustAttack) || mage.CanBlock(wall, p, g) {
		t.Fatal("copied combat rules")
	}
	g.PhaseIn(g.PhaseOut(p.ID()))
	if chooser.offers != 1 {
		t.Fatal("phase repeated entry")
	}
	chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return p }
	next := g.PutOnBattlefield(repositoryCard(t, "LEB", "Clone"), a.PlayerID())
	if next == nil || !next.HasType(core.TypeEnchantment) || next.Name() != "Juggernaut" {
		t.Fatal("subsequent copy lost exception")
	}
	g.BouncePermanentToHand(p)
	if a.Hand()[0].Name() != "Copy Artifact" || !a.Hand()[0].HasType(core.TypeEnchantment) || a.Hand()[0].HasType(core.TypeArtifact) || g.LKI(card.ID()).Name != "Juggernaut" {
		t.Fatal("physical rules or LKI")
	}
	if !next.HasType(core.TypeEnchantment) {
		t.Fatal("copy followed donor departure")
	}
	returned := a.Hand()[0]
	a.RemoveFromHand(returned.ID())
	chooser.accept = false
	again := g.PutOnBattlefield(returned, a.PlayerID())
	if again == nil || again.Name() != "Copy Artifact" || again.HasType(core.TypeArtifact) {
		t.Fatal("returned card retained copy")
	}
	clone.ResolveStack()
	other := clone.FindPermanent(card.ID())
	if other == nil || other.Name() != "Copy Artifact" || other.HasType(core.TypeArtifact) || decline.offers != 1 {
		t.Fatal("clone choice isolation")
	}
}

func TestRONCopyTypeModificationManaAndLegality(t *testing.T) {
	g, a, b := ronGame()
	chooser := &ronEntryCopyPlayer{Player: a, accept: true}
	g.SetPlayerAt(0, chooser)
	source := g.PutOnBattlefield(repositoryCard(t, "LEB", "Sol Ring"), b.PlayerID())
	p := g.PutOnBattlefield(repositoryCard(t, "LEB", "Copy Artifact"), a.PlayerID())
	if p == nil || !p.HasType(core.TypeEnchantment) || !p.HasType(core.TypeArtifact) || p.HasType(core.TypeCreature) {
		t.Fatal("noncreature copy types")
	}
	if err := g.ActivateAbilityByIndex(a.PlayerID(), p.ID(), 0, nil); err != nil {
		t.Fatal(err)
	}
	if !p.Tapped || source.Tapped || a.ManaPool().TotalMana() != 2 {
		t.Fatal("copied mana ability source")
	}
	spell := repositoryCard(t, "LEB", "Disenchant")
	b.AddToHand(spell)
	b.ManaPool().Add(core.White, 1)
	b.ManaPool().Add(core.Colorless, 1)
	if err := g.CastSpellByID(b.PlayerID(), spell.ID(), []uuid.UUID{p.ID()}, 0); err != nil {
		t.Fatal(err)
	}
	g.ResolveStack()
	if g.FindPermanent(p.ID()) != nil || a.Graveyard()[0].Name() != "Copy Artifact" || g.FindPermanent(source.ID()) == nil {
		t.Fatal("target legality or printed graveyard rules")
	}
}

func TestRONCopyTypeModificationInvalid(t *testing.T) {
	for _, form := range []string{`CopyEffects`, `CopyEffects()`, `CopyEffects([])`, `CopyEffects(AddCardtypes([Enchantment]))`, `CopyEffects([AddCardtypes])`, `CopyEffects([AddCardtypes()])`, `CopyEffects([AddCardtypes([])])`, `CopyEffects([AddCardtypes([Enchantment()])])`, `CopyEffects([AddCardtypes([Instant])])`, `CopyEffects([AddCardtypes([Unknown])])`, `CopyEffects([KeepColor])`, `CopyEffects([AddCardtypes([Enchantment]),KeepColor])`, `CopyEffects([AddCardtypes([Enchantment]),AddAbility([])])`} {
		src := `Card(Name:"Invalid",Typeline:(Supertypes:[],Cardtypes:[Enchantment],Subtypes:[]),Rules:[AsPermanentEnters(ThisPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Artifact),` + form + `)])])])`
		nodes, err := Parse(strings.NewReader(src), "invalid.ron")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile(nodes[0]); err == nil {
			t.Fatalf("accepted %s", form)
		}
	}
}

func TestRONCopyTypeModificationLayers(t *testing.T) {
	g, a, b := ronGame()
	chooser := &ronEntryCopyPlayer{Player: a, accept: true}
	g.SetPlayerAt(0, chooser)
	g.PutOnBattlefield(repositoryCard(t, "LEB", "Sol Ring"), b.PlayerID())
	p := g.PutOnBattlefield(repositoryCard(t, "LEB", "Copy Artifact"), a.PlayerID())
	if err := mage.ApplyEffect(g, mage.Until(core.EndOfTurn, mage.AddCardType(mage.This, core.TypeCreature), mage.AddCardType(mage.This, core.TypeLand), mage.SetBasePowerToughness(mage.This, mage.Fixed(4), mage.Fixed(4)), mage.SetColors(mage.This, core.Red)), p.ID(), a.PlayerID(), nil); err != nil {
		t.Fatal(err)
	}
	if !p.HasType(core.TypeEnchantment) || !p.HasType(core.TypeCreature) || !p.HasType(core.TypeLand) || p.CurrentPower(g) != 4 {
		t.Fatal("later layers did not read modified copy")
	}
	chooser.pick = func([]*mage.Permanent, mage.GameReader) *mage.Permanent { return p }
	copied := g.PutOnBattlefield(repositoryCard(t, "LEB", "Clone"), a.PlayerID())
	if copied == nil || !copied.HasType(core.TypeArtifact) || !copied.HasType(core.TypeEnchantment) || copied.HasType(core.TypeCreature) || copied.HasType(core.TypeLand) || copied.Name() != "Sol Ring" || len(copied.Colors()) != 0 {
		t.Fatal("later layers entered copiable values")
	}
	g.GetEffects().RemoveEndOfTurn()
	g.ApplyContinuousEffects()
	p = g.FindPermanent(p.ID())
	if p.HasType(core.TypeCreature) || p.HasType(core.TypeLand) || !p.HasType(core.TypeEnchantment) {
		t.Fatal("cleanup removed copy exception")
	}
}

func TestRONCopyTypeModificationComposedTypes(t *testing.T) {
	g, a, b := ronGame()
	chooser := &ronEntryCopyPlayer{Player: a, accept: true}
	g.SetPlayerAt(0, chooser)
	source := g.PutOnBattlefield(repositoryCard(t, "LEB", "Sol Ring"), b.PlayerID())
	factory := permanentScopeFactory(t, "Enchantment", `AsPermanentEnters(ThisPermanent,[MayActions([EnterAsACopyOfAPermanent(IsCardtype(Artifact),CopyEffects([AddCardtypes([Enchantment,Artifact]),AddCardtypes([Land,Enchantment])]))])])`)
	p := g.PutOnBattlefield(factory(), a.PlayerID())
	if p == nil || len(p.Card.Types()) != 3 || !p.HasType(core.TypeArtifact) || !p.HasType(core.TypeEnchantment) || !p.HasType(core.TypeLand) || source.HasType(core.TypeLand) {
		t.Fatal("ordered additive exceptions or deduplication")
	}
}

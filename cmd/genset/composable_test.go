package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedCardsUseDSL(t *testing.T) {
	dir := t.TempDir()
	cards := []Card{{Name: "Example", ManaCost: "{G}", TypeLine: "Enchantment", OracleText: "Creatures you control get +1/+1.", Keywords: []string{"Flying"}}}
	writeFileWithCards(dir, "example", "enchantments.go", "registerEnchantments", cards, enchantmentStub)
	content, err := os.ReadFile(filepath.Join(dir, "enchantments.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"github.com/benprew/mage-go/pkg/mage/dsl"`) || strings.Contains(string(content), `"github.com/benprew/mage-go/pkg/mage"`) {
		t.Fatal("generated production definitions bypass the DSL")
	}
	if !strings.Contains(string(content), cards[0].OracleText) {
		t.Fatal("generator lost Oracle text")
	}
}

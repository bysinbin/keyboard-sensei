package engine

import (
	"strings"
	"testing"
)

func TestModifiersToMask(t *testing.T) {
	tests := []struct {
		mods     []string
		expected uint32
	}{
		{mods: []string{"cmd"}, expected: ModCmd},
		{mods: []string{"alt"}, expected: ModAlt},
		{mods: []string{"ctrl"}, expected: ModCtrl},
		{mods: []string{"shift"}, expected: ModShift},
		{mods: []string{"cmd", "alt"}, expected: ModCmd | ModAlt},
		{mods: []string{"command", "option", "control", "shift"}, expected: ModCmd | ModAlt | ModCtrl | ModShift},
		{mods: []string{}, expected: 0},
	}

	for _, tt := range tests {
		got := ModifiersToMask(tt.mods)
		if got != tt.expected {
			t.Errorf("ModifiersToMask(%v) = %d, expected %d", tt.mods, got, tt.expected)
		}
	}
}

func TestMaskToModifiers(t *testing.T) {
	mask := ModCmd | ModShift
	mods := MaskToModifiers(mask)
	if len(mods) != 2 {
		t.Fatalf("expected 2 modifiers, got %d", len(mods))
	}
	hasCmd := false
	hasShift := false
	for _, m := range mods {
		if m == "cmd" {
			hasCmd = true
		}
		if m == "shift" {
			hasShift = true
		}
	}
	if !hasCmd || !hasShift {
		t.Errorf("expected cmd and shift in modifiers, got %v", mods)
	}
}

func TestFormatShortcut(t *testing.T) {
	sc := FormatShortcut([]string{"cmd"}, 43)
	if !strings.Contains(sc, "⌘") || !strings.Contains(sc, "Ö") {
		t.Errorf("expected shortcut to contain ⌘ and Ö, got %s", sc)
	}

	scDev := FormatShortcut([]string{"cmd", "alt"}, 2)
	if !strings.Contains(scDev, "⌘") || !strings.Contains(scDev, "⌥") || !strings.Contains(scDev, "D") {
		t.Errorf("expected shortcut to contain ⌘, ⌥ and D, got %s", scDev)
	}
}

func TestVirtualKeysLookup(t *testing.T) {
	keysToVerify := []struct {
		code int
		name string
	}{
		{code: 43, name: "ö"},
		{code: 47, name: "ç"},
		{code: 44, name: "."},
		{code: 42, name: ","},
		{code: 27, name: "-"},
	}

	for _, k := range keysToVerify {
		info, ok := VirtualKeys[k.code]
		if !ok {
			t.Errorf("key code %d not found in VirtualKeys", k.code)
			continue
		}
		if info.Name != k.name {
			t.Errorf("key code %d name expected %s, got %s", k.code, k.name, info.Name)
		}
	}
}

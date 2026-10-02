package engine

import (
	"fmt"
	"strings"
)

// Modifier bit flags
const (
	ModCmd   uint32 = 1 << 0
	ModAlt   uint32 = 1 << 1
	ModCtrl  uint32 = 1 << 2
	ModShift uint32 = 1 << 3
)

type KeyInfo struct {
	Code        int    `json:"code"`
	Name        string `json:"name"`
	TurkishDesc string `json:"turkish_desc"`
}

// Known macOS virtual keycodes with Turkish Q & ANSI context
var VirtualKeys = map[int]KeyInfo{
	// Special ANSI / Turkish Q mapped keys
	43: {Code: 43, Name: "ö", TurkishDesc: "Türkçe 'ö' (ANSI Virgül tuşu ',')"},
	47: {Code: 47, Name: "ç", TurkishDesc: "Türkçe 'ç' (ANSI Nokta tuşu '.')"},
	44: {Code: 44, Name: ".", TurkishDesc: "Türkçe '.' (ANSI Bölü tuşu '/')"},
	42: {Code: 42, Name: ",", TurkishDesc: "Türkçe ',' (ANSI Ters Eğik Çizgi '\\')"},
	41: {Code: 41, Name: "ş", TurkishDesc: "Türkçe 'ş' (ANSI Noktalı Virgül ';')"},
	39: {Code: 39, Name: "i", TurkishDesc: "Türkçe 'i' (ANSI Tek Tırnak ''')"},
	33: {Code: 33, Name: "ğ", TurkishDesc: "Türkçe 'ğ' (ANSI Sol Köşeli Parantez '[')"},
	30: {Code: 30, Name: "ü", TurkishDesc: "Türkçe 'ü' (ANSI Sağ Köşeli Parantez ']')"},
	27: {Code: 27, Name: "-", TurkishDesc: "ANSI Eksi / Tire '-'"},
	24: {Code: 24, Name: "=", TurkishDesc: "ANSI Eşittir '='"},
	50: {Code: 50, Name: "\" (ANSI) / < (ISO)", TurkishDesc: "Türkçe '\"' (ANSI Grave / Tırnak) veya ISO '<' tuşu"},
	10: {Code: 10, Name: "\" (ISO) / §", TurkishDesc: "Türkçe '\"' (ISO Tırnak) / Bölüm '§' (1'in solundaki tuş)"},

	// Letters A-Z
	0:  {Code: 0, Name: "a", TurkishDesc: "A tuşu"},
	1:  {Code: 1, Name: "s", TurkishDesc: "S tuşu"},
	2:  {Code: 2, Name: "d", TurkishDesc: "D tuşu"},
	3:  {Code: 3, Name: "f", TurkishDesc: "F tuşu"},
	4:  {Code: 4, Name: "h", TurkishDesc: "H tuşu"},
	5:  {Code: 5, Name: "g", TurkishDesc: "G tuşu"},
	6:  {Code: 6, Name: "z", TurkishDesc: "Z tuşu"},
	7:  {Code: 7, Name: "x", TurkishDesc: "X tuşu"},
	8:  {Code: 8, Name: "c", TurkishDesc: "C tuşu"},
	9:  {Code: 9, Name: "v", TurkishDesc: "V tuşu"},
	11: {Code: 11, Name: "b", TurkishDesc: "B tuşu"},
	45: {Code: 45, Name: "n", TurkishDesc: "N tuşu"},
	46: {Code: 46, Name: "m", TurkishDesc: "M tuşu"},
	12: {Code: 12, Name: "q", TurkishDesc: "Q tuşu"},
	13: {Code: 13, Name: "w", TurkishDesc: "W tuşu"},
	14: {Code: 14, Name: "e", TurkishDesc: "E tuşu"},
	15: {Code: 15, Name: "r", TurkishDesc: "R tuşu"},
	17: {Code: 17, Name: "t", TurkishDesc: "T tuşu"},
	16: {Code: 16, Name: "y", TurkishDesc: "Y tuşu"},
	32: {Code: 32, Name: "u", TurkishDesc: "U tuşu"},
	34: {Code: 34, Name: "ı", TurkishDesc: "I tuşu (Türkçe I / İngilizce I)"},
	31: {Code: 31, Name: "o", TurkishDesc: "O tuşu"},
	35: {Code: 35, Name: "p", TurkishDesc: "P tuşu"},
	38: {Code: 38, Name: "j", TurkishDesc: "J tuşu"},
	40: {Code: 40, Name: "k", TurkishDesc: "K tuşu"},
	37: {Code: 37, Name: "l", TurkishDesc: "L tuşu"},

	// Numbers
	18: {Code: 18, Name: "1", TurkishDesc: "1 tuşu"},
	19: {Code: 19, Name: "2", TurkishDesc: "2 tuşu"},
	20: {Code: 20, Name: "3", TurkishDesc: "3 tuşu"},
	21: {Code: 21, Name: "4", TurkishDesc: "4 tuşu"},
	23: {Code: 23, Name: "5", TurkishDesc: "5 tuşu"},
	22: {Code: 22, Name: "6", TurkishDesc: "6 tuşu"},
	26: {Code: 26, Name: "7", TurkishDesc: "7 tuşu"},
	28: {Code: 28, Name: "8", TurkishDesc: "8 tuşu"},
	25: {Code: 25, Name: "9", TurkishDesc: "9 tuşu"},
	29: {Code: 29, Name: "0", TurkishDesc: "0 tuşu"},

	// Special keys
	49: {Code: 49, Name: "space", TurkishDesc: "Boşluk (Space)"},
	36: {Code: 36, Name: "return", TurkishDesc: "Enter / Return"},
	48: {Code: 48, Name: "tab", TurkishDesc: "Sekme (Tab)"},
	51: {Code: 51, Name: "backspace", TurkishDesc: "Geri Silme (Backspace)"},
	53: {Code: 53, Name: "escape", TurkishDesc: "ESC (Escape)"},
	117: {Code: 117, Name: "delete", TurkishDesc: "İleri Silme (Forward Delete)"},

	// Arrows
	123: {Code: 123, Name: "left", TurkishDesc: "Sol Ok Tuşu"},
	124: {Code: 124, Name: "right", TurkishDesc: "Sağ Ok Tuşu"},
	125: {Code: 125, Name: "down", TurkishDesc: "Aşağı Ok Tuşu"},
	126: {Code: 126, Name: "up", TurkishDesc: "Yukarı Ok Tuşu"},

	// Function Keys
	122: {Code: 122, Name: "F1", TurkishDesc: "F1 Tuşu"},
	120: {Code: 120, Name: "F2", TurkishDesc: "F2 Tuşu"},
	99:  {Code: 99, Name: "F3", TurkishDesc: "F3 Tuşu"},
	118: {Code: 118, Name: "F4", TurkishDesc: "F4 Tuşu"},
	96:  {Code: 96, Name: "F5", TurkishDesc: "F5 Tuşu"},
	97:  {Code: 97, Name: "F6", TurkishDesc: "F6 Tuşu"},
	98:  {Code: 98, Name: "F7", TurkishDesc: "F7 Tuşu"},
	100: {Code: 100, Name: "F8", TurkishDesc: "F8 Tuşu"},
	101: {Code: 101, Name: "F9", TurkishDesc: "F9 Tuşu"},
	109: {Code: 109, Name: "F10", TurkishDesc: "F10 Tuşu"},
	103: {Code: 103, Name: "F11", TurkishDesc: "F11 Tuşu"},
	111: {Code: 111, Name: "F12", TurkishDesc: "F12 Tuşu"},
}

func ModifiersToMask(mods []string) uint32 {
	var mask uint32
	for _, m := range mods {
		switch strings.ToLower(strings.TrimSpace(m)) {
		case "cmd", "command", "meta":
			mask |= ModCmd
		case "alt", "option", "opt":
			mask |= ModAlt
		case "ctrl", "control":
			mask |= ModCtrl
		case "shift":
			mask |= ModShift
		}
	}
	return mask
}

func MaskToModifiers(mask uint32) []string {
	var mods []string
	if mask&ModCmd != 0 {
		mods = append(mods, "cmd")
	}
	if mask&ModAlt != 0 {
		mods = append(mods, "alt")
	}
	if mask&ModCtrl != 0 {
		mods = append(mods, "ctrl")
	}
	if mask&ModShift != 0 {
		mods = append(mods, "shift")
	}
	return mods
}

func FormatShortcut(mods []string, keycode int) string {
	var symbols []string
	mask := ModifiersToMask(mods)
	if mask&ModCtrl != 0 {
		symbols = append(symbols, "⌃")
	}
	if mask&ModAlt != 0 {
		symbols = append(symbols, "⌥")
	}
	if mask&ModShift != 0 {
		symbols = append(symbols, "⇧")
	}
	if mask&ModCmd != 0 {
		symbols = append(symbols, "⌘")
	}

	keyStr := fmt.Sprintf("#%d", keycode)
	if info, ok := VirtualKeys[keycode]; ok {
		keyStr = strings.ToUpper(info.Name)
	}

	return strings.Join(append(symbols, keyStr), " + ")
}

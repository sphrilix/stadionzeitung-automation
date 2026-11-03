package bfv

import (
	"strconv"
	"strings"
)

// glyphText returns the text a glyph stands for, by its Adobe glyph name,
// e.g. "a", "adieresis", "one" or "uni0326". It covers what the
// obfuscation fonts of bfv.de contain: Latin-1, Latin Extended-A and some
// symbols.
func glyphText(name string) (string, bool) {
	if len(name) == 1 && ('A' <= name[0] && name[0] <= 'Z' || 'a' <= name[0] && name[0] <= 'z') {
		return name, true
	}
	if hex, ok := strings.CutPrefix(name, "uni"); ok && len(hex) == 4 {
		if r, err := strconv.ParseUint(hex, 16, 32); err == nil {
			return string(rune(r)), true
		}
	}
	s, ok := glyphNames[name]
	return s, ok
}

var glyphNames = map[string]string{
	"space": " ", "exclam": "!", "quotedbl": `"`, "numbersign": "#",
	"dollar": "$", "percent": "%", "ampersand": "&", "quotesingle": "'",
	"parenleft": "(", "parenright": ")", "asterisk": "*", "plus": "+",
	"comma": ",", "hyphen": "-", "period": ".", "slash": "/",
	"zero": "0", "one": "1", "two": "2", "three": "3",
	"four": "4", "five": "5", "six": "6", "seven": "7",
	"eight": "8", "nine": "9", "colon": ":", "semicolon": ";",
	"less": "<", "equal": "=", "greater": ">", "question": "?",
	"at": "@", "bracketleft": "[", "backslash": `\`, "bracketright": "]",
	"asciicircum": "^", "underscore": "_", "grave": "`", "braceleft": "{",
	"bar": "|", "braceright": "}", "asciitilde": "~", "exclamdown": "¡",
	"cent": "¢", "sterling": "£", "currency": "¤", "yen": "¥",
	"brokenbar": "¦", "section": "§", "dieresis": "¨", "copyright": "©",
	"ordfeminine": "ª", "guillemotleft": "«", "registered": "®", "macron": "¯",
	"degree": "°", "plusminus": "±", "twosuperior": "²", "threesuperior": "³",
	"acute": "´", "paragraph": "¶", "periodcentered": "·", "cedilla": "¸",
	"onesuperior": "¹", "ordmasculine": "º", "guillemotright": "»", "onequarter": "¼",
	"onehalf": "½", "threequarters": "¾", "questiondown": "¿", "Agrave": "À",
	"Aacute": "Á", "Acircumflex": "Â", "Atilde": "Ã", "Adieresis": "Ä",
	"Aring": "Å", "AE": "Æ", "Ccedilla": "Ç", "Egrave": "È",
	"Eacute": "É", "Ecircumflex": "Ê", "Edieresis": "Ë", "Igrave": "Ì",
	"Iacute": "Í", "Icircumflex": "Î", "Idieresis": "Ï", "Eth": "Ð",
	"Ntilde": "Ñ", "Ograve": "Ò", "Oacute": "Ó", "Ocircumflex": "Ô",
	"Otilde": "Õ", "Odieresis": "Ö", "multiply": "×", "Oslash": "Ø",
	"Ugrave": "Ù", "Uacute": "Ú", "Ucircumflex": "Û", "Udieresis": "Ü",
	"Yacute": "Ý", "Thorn": "Þ", "germandbls": "ß", "agrave": "à",
	"aacute": "á", "acircumflex": "â", "atilde": "ã", "adieresis": "ä",
	"aring": "å", "ae": "æ", "ccedilla": "ç", "egrave": "è",
	"eacute": "é", "ecircumflex": "ê", "edieresis": "ë", "igrave": "ì",
	"iacute": "í", "icircumflex": "î", "idieresis": "ï", "eth": "ð",
	"ntilde": "ñ", "ograve": "ò", "oacute": "ó", "ocircumflex": "ô",
	"otilde": "õ", "odieresis": "ö", "divide": "÷", "oslash": "ø",
	"ugrave": "ù", "uacute": "ú", "ucircumflex": "û", "udieresis": "ü",
	"yacute": "ý", "thorn": "þ", "ydieresis": "ÿ", "Amacron": "Ā",
	"amacron": "ā", "Abreve": "Ă", "abreve": "ă", "Aogonek": "Ą",
	"aogonek": "ą", "Cacute": "Ć", "cacute": "ć", "Cdotaccent": "Ċ",
	"cdotaccent": "ċ", "Ccaron": "Č", "ccaron": "č", "Dcaron": "Ď",
	"dcaron": "ď", "Dcroat": "Đ", "dcroat": "đ", "Emacron": "Ē",
	"emacron": "ē", "Edotaccent": "Ė", "edotaccent": "ė", "Eogonek": "Ę",
	"eogonek": "ę", "Ecaron": "Ě", "ecaron": "ě", "Gbreve": "Ğ",
	"gbreve": "ğ", "Gdotaccent": "Ġ", "gdotaccent": "ġ", "Gcommaaccent": "Ģ",
	"gcommaaccent": "ģ", "Hbar": "Ħ", "hbar": "ħ", "Imacron": "Ī",
	"imacron": "ī", "Iogonek": "Į", "iogonek": "į", "Idotaccent": "İ",
	"IJ": "Ĳ", "ij": "ĳ", "Kcommaaccent": "Ķ", "kcommaaccent": "ķ",
	"Lacute": "Ĺ", "lacute": "ĺ", "Lcommaaccent": "Ļ", "lcommaaccent": "ļ",
	"Lcaron": "Ľ", "lcaron": "ľ", "Ldot": "Ŀ", "ldot": "ŀ",
	"Lslash": "Ł", "lslash": "ł", "Nacute": "Ń", "nacute": "ń",
	"Ncommaaccent": "Ņ", "ncommaaccent": "ņ", "Ncaron": "Ň", "ncaron": "ň",
	"Eng": "Ŋ", "eng": "ŋ", "Omacron": "Ō", "omacron": "ō",
	"Ohungarumlaut": "Ő", "ohungarumlaut": "ő", "OE": "Œ", "oe": "œ",
	"Racute": "Ŕ", "racute": "ŕ", "Rcommaaccent": "Ŗ", "rcommaaccent": "ŗ",
	"Rcaron": "Ř", "rcaron": "ř", "Sacute": "Ś", "sacute": "ś",
	"Scedilla": "Ş", "scedilla": "ş", "Scaron": "Š", "scaron": "š",
	"Tcedilla": "Ţ", "tcedilla": "ţ", "Tcaron": "Ť", "tcaron": "ť",
	"Tbar": "Ŧ", "tbar": "ŧ", "Umacron": "Ū", "umacron": "ū",
	"Uring": "Ů", "uring": "ů", "Uhungarumlaut": "Ű", "uhungarumlaut": "ű",
	"Uogonek": "Ų", "uogonek": "ų", "Wcircumflex": "Ŵ", "wcircumflex": "ŵ",
	"Ycircumflex": "Ŷ", "ycircumflex": "ŷ", "Ydieresis": "Ÿ", "Zacute": "Ź",
	"zacute": "ź", "Zdotaccent": "Ż", "zdotaccent": "ż", "Zcaron": "Ž",
	"zcaron": "ž", "florin": "ƒ", "circumflex": "ˆ", "caron": "ˇ",
	"breve": "˘", "dotaccent": "˙", "ring": "˚", "ogonek": "˛",
	"tilde": "˜", "hungarumlaut": "˝",
}

package receiptcategory

import (
	"regexp"
	"strings"
)

var (
	bracketRe = regexp.MustCompile(`\([^)]*\)`)
	numberRe  = regexp.MustCompile(`[0-9]+([.,][0-9]+)?`)
	// оставляем только буквы, пробел и дефис (дефис — часть слов вроде
	// "аи-92", "то-го"), остальную пунктуацию выбрасываем
	punctRe = regexp.MustCompile(`[^\p{L}\s-]+`)
	spaceRe = regexp.MustCompile(`\s+`)
)

// единицы измерения/фасовки — не несут смысла для категоризации, мешают
// совпадению по ключевым словам ("молоко 1л" не должно давать токен "л")
var unitWords = map[string]bool{
	"кг": true, "г": true, "гр": true, "л": true, "мл": true,
	"шт": true, "уп": true, "упак": true, "м": true, "см": true, "мм": true,
	"пач": true, "бан": true, "бут": true, "кор": true, "compact": true,
}

// NormalizeItemName приводит название позиции чека к виду, удобному для
// сравнения: нижний регистр, без бренда в скобках, без чисел/пунктуации.
func NormalizeItemName(name string) string {
	s := strings.ToLower(name)
	s = bracketRe.ReplaceAllString(s, " ")
	s = numberRe.ReplaceAllString(s, " ")
	s = punctRe.ReplaceAllString(s, " ")
	s = spaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// Tokenize разбивает уже нормализованное название на отдельные слова,
// отбрасывая единицы измерения и совсем короткие "слова" (предлоги, мусор
// после чистки пунктуации).
func Tokenize(normalized string) []string {
	words := strings.Fields(normalized)
	out := make([]string, 0, len(words))
	for _, w := range words {
		if len([]rune(w)) < 2 {
			continue
		}
		if unitWords[w] {
			continue
		}
		out = append(out, w)
	}
	return out
}

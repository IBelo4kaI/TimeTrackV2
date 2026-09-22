package receiptcategory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	repo "timetrack/internal/adapter/mysql/sqlc"

	"github.com/kljensen/snowball/russian"
)

// ItemForClassify — минимум, нужный алгоритму из позиции чека.
type ItemForClassify struct {
	Name string
}

// ClassifyResult — итог автоклассификации. CategoryID == nil означает
// "Без категории" — ни продавец, ни позиции ни с чем не совпали.
type ClassifyResult struct {
	CategoryID *int32
	Source     string // "seed" | "keyword_match" | "" (без категории)
}

// stem — лёгкая нормализация слова к основе (см. спецификацию: "по
// возможности — привести слово к начальной форме через лёгкий стеммер").
// stemStopWords=true — стеммер snowball по умолчанию не трогает частые
// стоп-слова, нам нужно наоборот единообразие для всех слов.
func stem(word string) string {
	return russian.Stem(word, true)
}

// Classify реализует алгоритм из спецификации:
//  1. ИНН продавца -> merchant_category (уже знаем — готово).
//  2. Не найдено -> по всем позициям: нормализация, токенизация, стемминг,
//     каждое слово (или фраза, если ключ multi-word) ищем в keyword_category,
//     совпадение — +1 голос категории.
//  3. Голосов нет -> "Без категории".
//  4. Голоса есть -> категория с максимальным весом (при равенстве —
//     меньший id, для детерминированности).
//  5. Победившую по ключевым словам категорию запоминаем в
//     merchant_category с source='keyword_match' — следующий чек от этого
//     же продавца в следующий раз попадёт в шаг 1.
func (s *service) Classify(ctx context.Context, sellerInn string, items []ItemForClassify) (ClassifyResult, error) {
	return s.classify(ctx, sellerInn, items, true)
}

// Preview — то же самое, но без шага 5 (самообучение): для показа
// определившейся категории ДО сохранения чека (см. ReceiptScan.vue), когда
// неизвестно, будет ли чек вообще добавлен — писать в merchant_category
// спекулятивно не нужно.
func (s *service) Preview(ctx context.Context, sellerInn string, items []ItemForClassify) (ClassifyResult, error) {
	return s.classify(ctx, sellerInn, items, false)
}

func (s *service) classify(ctx context.Context, sellerInn string, items []ItemForClassify, learn bool) (ClassifyResult, error) {
	if sellerInn != "" {
		mc, err := s.repo.GetMerchantCategory(ctx, sellerInn)
		if err == nil {
			id := mc.CategoryID
			return ClassifyResult{CategoryID: &id, Source: mc.Source}, nil
		} else if !errors.Is(err, sql.ErrNoRows) {
			return ClassifyResult{}, fmt.Errorf("get merchant category: %w", err)
		}
	}

	keywords, err := s.repo.ListKeywordCategories(ctx)
	if err != nil {
		return ClassifyResult{}, fmt.Errorf("list keyword categories: %w", err)
	}

	votes := make(map[int32]int)
	for _, item := range items {
		normalized := NormalizeItemName(item.Name)
		if normalized == "" {
			continue
		}

		stemmed := make(map[string]bool)
		for _, tok := range Tokenize(normalized) {
			stemmed[stem(tok)] = true
		}

		for _, kw := range keywords {
			if strings.Contains(kw.Keyword, " ") {
				// фраза (несколько слов) — сравниваем целиком подстрокой в
				// нормализованном (не стеммированном — стемминг фраз
				// ненадёжен) названии позиции
				if strings.Contains(normalized, kw.Keyword) {
					votes[kw.CategoryID]++
				}
				continue
			}
			if stemmed[stem(kw.Keyword)] {
				votes[kw.CategoryID]++
			}
		}
	}

	if len(votes) == 0 {
		return ClassifyResult{}, nil
	}

	var bestID int32
	bestVotes := -1
	for id, v := range votes {
		if v > bestVotes || (v == bestVotes && id < bestID) {
			bestID, bestVotes = id, v
		}
	}

	if learn && sellerInn != "" {
		if err := s.repo.UpsertMerchantCategory(ctx, repo.UpsertMerchantCategoryParams{
			Inn:        sellerInn,
			CategoryID: bestID,
			Source:     "keyword_match",
		}); err != nil {
			// самообучение не должно ломать сохранение чека — просто лог
			fmt.Printf("learn merchant category: %v\n", err)
		}
	}

	return ClassifyResult{CategoryID: &bestID, Source: "keyword_match"}, nil
}

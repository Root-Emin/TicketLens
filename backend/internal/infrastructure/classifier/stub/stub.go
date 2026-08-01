// Package stub provides a deterministic, keyword-driven Classifier.
//
// It exists so the backend runs, seeds and tests without the Python model
// service. Determinism is the point: the same ticket always yields the same
// label and the same confidence, so seed data and end-to-end tests are
// reproducible. When the real model lands, an http adapter sits beside this
// package and port.Classifier does not change.
package stub

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/masterfabric-go/masterfabric/internal/domain/triage/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/port"
)

const (
	modelName    = "stub"
	modelVersion = "v0"

	// Confidence is derived from how many keywords matched, bounded to a band
	// that straddles a realistic review threshold so needs_human_review is
	// actually exercised.
	minConfidence  = 0.50
	maxConfidence  = 0.95
	confidenceStep = 0.15
)

// Classifier is a keyword-matching implementation of port.Classifier.
type Classifier struct{}

// New creates a new stub classifier.
func New() *Classifier { return &Classifier{} }

// Verify interface compliance at compile time.
var _ port.Classifier = (*Classifier)(nil)

// categoryKeywords maps each taxonomy label to the terms that signal it.
// Turkish and English terms sit side by side because real tickets mix both.
// Order matters only for tie-breaking, which is resolved deterministically by
// the fixed order of model.AllCategories.
var categoryKeywords = map[model.Category][]string{
	model.CategoryIntegration: {
		"entegrasyon", "integration", "trendyol", "hepsiburada", "n11", "pazaryeri",
		"marketplace", "kargo", "cargo", "erp", "logo", "mikro", "netsis", "muhasebe",
		"api", "sdk", "webhook", "senkron", "aktarm", "aktarmıyor", "sanal pos",
	},
	model.CategoryPaymentOps: {
		"hakediş", "hakedis", "settlement", "mutabakat", "iade", "refund",
		"chargeback", "ters ibraz", "para", "ödeme alamıyorum", "odeme alamiyorum",
		"işlem eksik", "islem eksik", "hesabıma geçmedi", "hesabima gecmedi", "payout",
	},
	model.CategoryBilling: {
		"fatura", "invoice", "abonelik", "subscription", "paket", "plan",
		"komisyon", "commission", "ücretlendirme", "ucretlendirme", "iptal talebi",
		"upgrade", "downgrade", "billing",
	},
	model.CategoryTechnicalIssue: {
		"hata", "error", "500", "çöktü", "coktu", "açılmıyor", "acilmiyor",
		"yavaş", "yavas", "slow", "down", "site kapalı", "site kapali",
		"panel açılmıyor", "yüklenmiyor", "yuklenmiyor", "timeout", "bug",
	},
	model.CategoryOnboarding: {
		"kurulum", "setup", "onboarding", "veri göçü", "veri gocu", "migration",
		"canlıya", "canliya", "go-live", "başvuru", "basvuru", "aktivasyon",
		"activation", "hesap açılışı", "hesap acilisi",
	},
	model.CategoryHowTo: {
		"nasıl", "nasil", "how to", "how do", "eğitim", "egitim", "doküman",
		"dokuman", "documentation", "kılavuz", "kilavuz", "öğrenmek", "ogrenmek",
		"ayarlamak", "konfigüre", "konfigure",
	},
	model.CategoryAccountAccess: {
		"şifre", "sifre", "password", "giriş yapamıyorum", "giris yapamiyorum",
		"login", "oturum", "session", "yetki", "rol", "role", "kullanıcı ekle",
		"kullanici ekle", "erişim", "erisim", "access", "2fa",
	},
	model.CategoryFeatureRequest: {
		"özellik talebi", "ozellik talebi", "feature request", "yol haritası",
		"yol haritasi", "roadmap", "öneri", "oneri", "eklenebilir mi",
		"olsa çok iyi", "olsa cok iyi", "destekliyor musunuz",
	},
	model.CategorySales: {
		"satın al", "satin al", "demo", "teklif", "fiyat", "pricing", "quote",
		"ek modül", "ek modul", "satış", "satis", "sales", "yeni paket",
	},
	model.CategoryCompliance: {
		"kvkk", "gdpr", "sözleşme", "sozlesme", "contract", "veri silme",
		"verilerimin silinmesi", "denetim", "audit", "compliance", "aydınlatma",
		"aydinlatma", "belge talebi", "imha",
	},
}

// urgentKeywords mark a stopped revenue stream, not an angry tone. Urgency is
// about the business being unable to operate.
var urgentKeywords = []string{
	"site kapalı", "site kapali", "site down", "çöktü", "coktu", "kapandı", "kapandi",
	"ödeme alamıyorum", "odeme alamiyorum", "ödeme alamıyoruz", "tahsilat yapamıyorum",
	"hakediş", "hakedis", "hesabıma geçmedi", "hesabima gecmedi", "para yatmadı",
	"para yatmadi", "kargo etiketi basmıyor", "kargo etiketi basmiyor",
	"satış yapamıyorum", "satis yapamiyorum", "sipariş alamıyorum", "siparis alamiyorum",
	"acil", "urgent", "production down", "müşterilerim etkileniyor",
}

// highKeywords are real breakage that has not stopped the whole business.
var highKeywords = []string{
	"hata", "error", "500", "aktarmıyor", "aktarmiyor", "senkronize olmuyor",
	"çalışmıyor", "calismiyor", "başarısız", "basarisiz", "failed", "bozuk",
	"eksik görünüyor", "eksik gorunuyor", "iade", "chargeback",
}

// lowKeywords are questions and requests: nothing is broken.
var lowKeywords = []string{
	"nasıl", "nasil", "how to", "how do", "bilgi", "öğrenmek", "ogrenmek",
	"doküman", "dokuman", "eğitim", "egitim", "öneri", "oneri", "özellik talebi",
	"ozellik talebi", "demo", "teklif", "fiyat", "merak",
}

// Classify scores the ticket text against the keyword tables.
func (c *Classifier) Classify(_ context.Context, in port.ClassifyInput) (port.ClassifyResult, error) {
	// The subject is weighted more heavily by being counted twice.
	text := normalize(in.Subject + " " + in.Subject + " " + in.Body)

	category, categoryHits := c.bestCategory(text)
	priority, priorityHits := c.priority(text)

	raw, _ := json.Marshal(map[string]any{
		"engine":         modelName,
		"category_hits":  categoryHits,
		"priority_hits":  priorityHits,
		"matched_length": len(text),
	})

	return port.ClassifyResult{
		Priority:           string(priority),
		PriorityConfidence: confidence(priorityHits),
		Category:           string(category),
		CategoryConfidence: confidence(categoryHits),
		ModelName:          modelName,
		ModelVersion:       modelVersion,
		Raw:                raw,
	}, nil
}

// bestCategory returns the highest scoring label. Ties break on the fixed order
// of model.AllCategories so the result never depends on map iteration order.
func (c *Classifier) bestCategory(text string) (model.Category, int) {
	type scored struct {
		category model.Category
		hits     int
		rank     int
	}

	results := make([]scored, 0, len(model.AllCategories))
	for rank, category := range model.AllCategories {
		results = append(results, scored{
			category: category,
			hits:     countMatches(text, categoryKeywords[category]),
			rank:     rank,
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].hits != results[j].hits {
			return results[i].hits > results[j].hits
		}
		return results[i].rank < results[j].rank
	})

	best := results[0]
	if best.hits == 0 {
		// Nothing matched. There is no "other" class by design, so fall back to
		// the broadest label at minimum confidence and let needs_human_review
		// flag it.
		return model.CategoryHowTo, 0
	}
	return best.category, best.hits
}

// priority follows revenue impact: a stopped business is urgent, visible
// breakage is high, questions are low.
func (c *Classifier) priority(text string) (model.TicketPriority, int) {
	if hits := countMatches(text, urgentKeywords); hits > 0 {
		return model.TicketPriorityUrgent, hits
	}
	if hits := countMatches(text, highKeywords); hits > 0 {
		return model.TicketPriorityHigh, hits
	}
	if hits := countMatches(text, lowKeywords); hits > 0 {
		return model.TicketPriorityLow, hits
	}
	return model.TicketPriorityNormal, 0
}

// confidence turns a hit count into a bounded score. Zero hits sit at the floor
// so an unmatched ticket falls under any sensible review threshold.
func confidence(hits int) float64 {
	score := minConfidence + float64(hits)*confidenceStep
	if score > maxConfidence {
		return maxConfidence
	}
	return score
}

func countMatches(text string, keywords []string) int {
	count := 0
	for _, keyword := range keywords {
		if strings.Contains(text, normalize(keyword)) {
			count++
		}
	}
	return count
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

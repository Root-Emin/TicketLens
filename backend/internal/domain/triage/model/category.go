package model

// Category is the classifier's label set. It is fixed and model-wide: the model
// learns these ten labels, while departments are per-organization. Mapping a
// category onto a department is the application's job, which is what lets us
// tell "the model was wrong" apart from "the organization has no department for
// this category".
//
// There is deliberately no "other" bucket. Ten classes cover the domain, and an
// escape hatch would collect every ambiguous example and destroy the low
// confidence signal. Uncertainty is expressed through needs_human_review.
type Category string

const (
	// CategoryTechnicalIssue: the platform itself is broken, erroring or slow.
	CategoryTechnicalIssue Category = "technical_issue"
	// CategoryIntegration: a third party link is failing — marketplace, cargo,
	// ERP/accounting, virtual POS, API/SDK/webhook. Kept apart from
	// technical_issue because it is routed to a different team.
	CategoryIntegration Category = "integration"
	// CategoryPaymentOps: money movement operations — settlement delays,
	// refunds, chargebacks, a transaction that looks missing.
	CategoryPaymentOps Category = "payment_ops"
	// CategoryBilling: what the customer pays the platform — invoices,
	// subscription, plan changes, commission rates, cancellation.
	CategoryBilling Category = "billing"
	// CategoryOnboarding: setup, data migration, go-live, application and
	// activation processes.
	CategoryOnboarding Category = "onboarding"
	// CategoryHowTo: nothing is broken; the customer asks how to do something.
	CategoryHowTo Category = "how_to"
	// CategoryAccountAccess: users, roles, sessions, passwords, panel access.
	CategoryAccountAccess Category = "account_access"
	// CategoryFeatureRequest: a capability that does not exist yet, roadmap.
	CategoryFeatureRequest Category = "feature_request"
	// CategorySales: pre-sales, add-on modules, demo requests.
	CategorySales Category = "sales"
	// CategoryCompliance: KVKK/GDPR, contracts, data deletion, audit documents.
	CategoryCompliance Category = "compliance"
)

// AllCategories is the taxonomy in a stable order, used for validation and for
// zero-filling the by-category statistics.
var AllCategories = []Category{
	CategoryTechnicalIssue,
	CategoryIntegration,
	CategoryPaymentOps,
	CategoryBilling,
	CategoryOnboarding,
	CategoryHowTo,
	CategoryAccountAccess,
	CategoryFeatureRequest,
	CategorySales,
	CategoryCompliance,
}

// ValidCategory reports whether c is part of the taxonomy.
func ValidCategory(c Category) bool {
	for _, known := range AllCategories {
		if known == c {
			return true
		}
	}
	return false
}

package main

import "github.com/Root-Emin/TicketLens/internal/domain/triage/model"

// customerSeed is one demo customer.
type customerSeed struct {
	Email    string
	FullName string
	Company  string
}

var demoCustomers = []customerSeed{
	{"alice.morgan@modaboutique.com", "Alice Morgan", "Moda Boutique"},
	{"michael.reed@acmetrade.com", "Michael Reed", "Acme Trade"},
	{"jessica.klein@technomarket.com", "Jessica Klein", "Techno Market"},
	{"daniel.foster@homedecor.com", "Daniel Foster", "Home Decor"},
	{"emily.carter@organicfoods.com", "Emily Carter", "Organic Foods"},
	{"brandon.hughes@sportcenter.com", "Brandon Hughes", "Sport Center"},
	{"olivia.bennett@bookworld.com", "Olivia Bennett", "Book World"},
	{"chris.walker@autoparts.com", "Chris Walker", "Auto Parts"},
	{"sophia.turner@cosmetichouse.com", "Sophia Turner", "Cosmetic House"},
	{"kevin.brooks@buildmarket.com", "Kevin Brooks", "Build Market"},
	{"hannah.price@babyshop.com", "Hannah Price", "Baby Shop"},
	{"ryan.cooper@petworld.com", "Ryan Cooper", "Pet World"},
	{"laura.evans@furniturehome.com", "Laura Evans", "Furniture Home"},
	{"nathan.ross@jewelrystudio.com", "Nathan Ross", "Jewelry Studio"},
}

// ticketSeed is one demo ticket. Category records what the text is written to
// trigger; the stub classifier still decides on its own, and the seed verifies
// the outcome rather than forcing it.
type ticketSeed struct {
	Subject  string
	Body     string
	Category model.Category
}

// ambiguousCategory marks tickets written WITHOUT classifier keywords. They
// exercise the low-confidence path: the stub finds nothing, confidence sits at
// the floor and needs_human_review turns true.
const ambiguousCategory model.Category = ""

// demoTickets is the ticket corpus. Volume is weighted the way a B2B SaaS help
// desk actually looks: integration and platform faults dominate, money
// operations and how-to questions follow.
var demoTickets = []ticketSeed{
	// ── integration (16) ─────────────────────────────────────────────────────
	{"Trendyol integration is not transferring products", "Our Trendyol marketplace integration has not been transferring products since last night, the sync keeps failing.", model.CategoryIntegration},
	{"Hepsiburada stock synchronization is not working", "Stock counts are not updating in the Hepsiburada integration, the sync job keeps halting halfway.", model.CategoryIntegration},
	{"N11 orders are not coming into the panel", "The N11 marketplace integration is not pulling orders, the API returns an error.", model.CategoryIntegration},
	{"Shipping label won't print", "The cargo integration is not creating labels, the shipping service returns an error and we cannot ship.", model.CategoryIntegration},
	{"Logo accounting integration is not transferring records", "The Logo ERP integration is not transferring records to accounting, there has been no sync since last night.", model.CategoryIntegration},
	{"Mikro ERP connection dropped", "The Mikro accounting integration lost its connection, data transfer stopped completely.", model.CategoryIntegration},
	{"Product codes don't match in the Netsis integration", "In the Netsis ERP integration the product codes do not match, so the transfer fails.", model.CategoryIntegration},
	{"Virtual POS integration returns an error", "Our virtual POS integration returns an error at the checkout step, the bank side looks fine.", model.CategoryIntegration},
	{"Webhook notifications are not arriving", "Order webhook notifications have not reached our server for two days, there is no record in the integration logs.", model.CategoryIntegration},
	{"Our API key is not working", "When we send requests with the newly created API key, the integration returns an authorization error.", model.CategoryIntegration},
	{"Sync broke after the SDK update", "After we updated the SDK version, the marketplace synchronization started failing.", model.CategoryIntegration},
	{"Trendyol commission data is not transferred", "In the Trendyol integration the commission field comes empty, the transfer is incomplete.", model.CategoryIntegration},
	{"Marketplace price updates are delayed", "In the marketplace integration price updates appear hours later, the sync is very slow.", model.CategoryIntegration},
	{"Aras cargo tracking number is not returned", "The Aras cargo integration creates the shipment but does not transfer the tracking number.", model.CategoryIntegration},
	{"Hepsiburada returns are not transferred", "In the Hepsiburada integration return orders are not transferred to the panel, the sync skips them.", model.CategoryIntegration},
	{"We are getting API rate limit errors", "Our integration service keeps hitting rate limit errors when making requests over the API.", model.CategoryIntegration},

	// ── technical_issue (16) ─────────────────────────────────────────────────
	{"The panel won't open, we get a 500 error", "Since this morning the admin panel won't open, a 500 error appears on the screen.", model.CategoryTechnicalIssue},
	{"Reports are not loading", "The sales reports page is not loading, it errors out after a long wait.", model.CategoryTechnicalIssue},
	{"The site is running very slowly", "The panel is extremely slow today, pages open late and occasionally throw an error.", model.CategoryTechnicalIssue},
	{"Order list is not loading", "The order list screen is not loading, the page stays blank and there is an error in the console.", model.CategoryTechnicalIssue},
	{"Product add page throws an error", "When adding a new product and pressing save we get an error, the record is not created.", model.CategoryTechnicalIssue},
	{"Dashboard charts come up empty", "The charts on the home screen are not loading, no data appears and an error message shows.", model.CategoryTechnicalIssue},
	{"Search results return very slowly", "Product search is extremely slow, sometimes it errors without returning any results.", model.CategoryTechnicalIssue},
	{"Excel export throws an error", "When trying to download a report as Excel the operation ends with an error.", model.CategoryTechnicalIssue},
	{"Image upload fails", "When uploading a product image the operation errors, the file just won't upload.", model.CategoryTechnicalIssue},
	{"The site is down, my customers can't get in", "Our site is completely down, my customers are affected and I cannot sell.", model.CategoryTechnicalIssue},
	{"Filtering is not working", "The filters on the order screen are not working, applying them throws an error.", model.CategoryTechnicalIssue},
	{"Bulk update screen won't open", "The bulk price update screen won't open, it keeps throwing an error.", model.CategoryTechnicalIssue},
	{"Notifications are not showing", "In-panel notifications don't show, the page is not loading and an error drops.", model.CategoryTechnicalIssue},
	{"White screen in the mobile app", "The mobile app stays on a white screen at launch and throws an error.", model.CategoryTechnicalIssue},
	{"Print preview is not loading", "The receipt print preview is not loading, the screen stays blank and an error appears.", model.CategoryTechnicalIssue},
	{"The panel is very slow, we get timeouts", "Operations in the panel are very slow, we frequently get timeout errors.", model.CategoryTechnicalIssue},

	// ── payment_ops (10) ─────────────────────────────────────────────────────
	{"Yesterday's payout did not reach my account", "The payout that closed yesterday was not credited to my account, I cannot receive payment.", model.CategoryPaymentOps},
	{"The refund was not reflected to the customer", "The refund amount we approved was not reflected on the customer's card, the refund process is stuck.", model.CategoryPaymentOps},
	{"We received a chargeback notice", "A chargeback notice arrived for a transaction, we would like information about the dispute process.", model.CategoryPaymentOps},
	{"The settlement report looks incomplete", "Some transactions appear to be missing in this week's settlement report.", model.CategoryPaymentOps},
	{"I cannot receive payment", "I cannot collect payments, incoming payments were not credited to my account and no payout shows.", model.CategoryPaymentOps},
	{"The payout amount is calculated short", "This month's payout is lower than expected, there is a reconciliation difference.", model.CategoryPaymentOps},
	{"The refund amount was calculated incorrectly", "In a partial refund the refund amount was miscalculated, the money came back short.", model.CategoryPaymentOps},
	{"There is a difference in the reconciliation report", "The amounts between the reconciliation file and the panel do not match, a transaction appears missing.", model.CategoryPaymentOps},
	{"The money transfer was delayed", "The payout normally arrived in two days, this time the deposit did not land.", model.CategoryPaymentOps},
	{"A transaction appears missing", "One of yesterday's transactions appears missing in the panel and is not in the settlement either.", model.CategoryPaymentOps},

	// ── how_to (9) ───────────────────────────────────────────────────────────
	{"How do I create a campaign?", "We want to learn how to create a campaign for the seasonal discount, step by step.", model.CategoryHowTo},
	{"How do I do a bulk product upload?", "How can we upload our list in bulk, is there documentation for it?", model.CategoryHowTo},
	{"Where can I find the user manual?", "Where is the panel's user manual or documentation located?", model.CategoryHowTo},
	{"How do I filter reports?", "How do I filter reports by date on the report screen, is there a guide?", model.CategoryHowTo},
	{"Are there any training videos?", "Is there training content or documentation for a new team member?", model.CategoryHowTo},
	{"How do I configure notification settings?", "How do I set my notification preferences, is there documentation to configure them?", model.CategoryHowTo},
	{"How is the low-stock alert level set?", "How should I configure the critical stock alert, we would like to learn.", model.CategoryHowTo},
	{"We want to learn the reporting module", "We request training or a guide for the reporting module.", model.CategoryHowTo},
	{"How do I change panel settings?", "How can I change the general panel settings, I could not find it in the documentation?", model.CategoryHowTo},

	// ── billing (3) — this org has NO matching department ─────────────────────
	{"I can't see my February invoice", "Our invoice for February does not appear in the panel, we request the invoice.", model.CategoryBilling},
	{"We want to upgrade our subscription package", "We would like to upgrade our current subscription package to a higher plan.", model.CategoryBilling},
	{"Our commission rate is wrong", "The commission rate in the contract differs from the rate reflected on the invoice.", model.CategoryBilling},

	// ── onboarding (3) ───────────────────────────────────────────────────────
	{"We're stuck during setup", "We cannot progress through the setup steps, we would like support during setup.", model.CategoryOnboarding},
	{"How long does data migration take?", "How long does the data migration process from the old system take, what is the migration plan?", model.CategoryOnboarding},
	{"What is needed to go live?", "What activation steps must we complete for go-live before going live?", model.CategoryOnboarding},

	// ── account_access (3) ───────────────────────────────────────────────────
	{"I can't reset my password", "The password reset email is not arriving, I cannot log in to the panel.", model.CategoryAccountAccess},
	{"We can't add a new user", "When we try to add a new user to the team we get a permission error, we cannot assign a role.", model.CategoryAccountAccess},
	{"We don't have access to the panel", "The accounting team has no access to the panel, permissions cannot be defined.", model.CategoryAccountAccess},

	// ── feature_request (3) ──────────────────────────────────────────────────
	{"Feature request for bulk discounts", "We would like to submit a feature request for applying bulk discounts to product groups.", model.CategoryFeatureRequest},
	{"Is multi-warehouse on the roadmap?", "Is multi-warehouse management on the roadmap, can you share it?", model.CategoryFeatureRequest},
	{"Do you support multiple currencies?", "Do you support selling in different currencies, we are submitting this as a suggestion.", model.CategoryFeatureRequest},

	// ── sales (3) ────────────────────────────────────────────────────────────
	{"We'd like a quote for an add-on module", "We want to purchase the reporting add-on module, we kindly request a price quote.", model.CategorySales},
	{"Demo request", "We have a demo request for the enterprise edition, could we meet at a convenient time?", model.CategorySales},
	{"Could you share a price list?", "We request a current price list and a quote for the new package options.", model.CategorySales},

	// ── compliance (3) ───────────────────────────────────────────────────────
	{"Data deletion request under KVKK", "Under KVKK we request the deletion of the data belonging to our company.", model.CategoryCompliance},
	{"Privacy notice and contract request", "We kindly request the privacy notice and a current copy of the contract.", model.CategoryCompliance},
	{"Document request for an audit", "We have a document request for our annual audit process.", model.CategoryCompliance},

	// ── ambiguous (7) — no keywords, low-confidence scenario ──────────────────
	{"I'd like to ask about something", "Hello, we are waiting to hear back from you regarding what we discussed yesterday.", ambiguousCategory},
	{"I have a quick question", "What is the current state of the request we submitted last week?", ambiguousCategory},
	{"We're waiting to hear back", "We have not yet heard back from your team on the matter, thank you.", ambiguousCategory},
	{"Continuation of our conversation", "We would like to move forward with what we talked about in Tuesday's meeting.", ambiguousCategory},
	{"About the attached matter", "Is there any progress on the matter we shared earlier?", ambiguousCategory},
	{"We have a small request", "Could you get in touch with us when you are available?", ambiguousCategory},
	{"A quick check-in", "What stage is the process at, could you let us know?", ambiguousCategory},
}

// agentReplies feed the multi-message threads.
var agentReplies = []string{
	"Hello, we have received your request. The relevant team is reviewing it and we will get back to you as soon as possible.",
	"We have forwarded the matter to the technical team. We found the related record in the logs and are working on it.",
	"We have identified the issue and a fix is being prepared. We will let you know once it is deployed.",
	"We have completed our checks and restarted the operation on our side. Could you please confirm?",
	"Thanks for the details, we have updated the process and are tracking it.",
}

var customerFollowUps = []string{
	"Thanks, could you let us know when this will be resolved?",
	"The issue persists, I am sharing a screenshot.",
	"We tried again today but ran into the same situation.",
	"Understood, we will wait. I would like to stress that this is urgent.",
}

var internalNotes = []string{
	"Internal note: critical customer, SLA must be tracked.",
	"Internal note: a similar record was opened last week, recurring issue.",
	"Internal note: escalated to the integration team.",
}

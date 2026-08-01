package events

// Kafka topic constants. Each bounded context has its own topic.
// Events within a context share the same topic and are distinguished by the
// Envelope.Type field (e.g. "user.registered", "user.invited").
const (
	// IAM events
	TopicIAM = "masterfabric.iam"

	// Tenant & App events
	TopicTenant = "masterfabric.tenant"

	// API Management events
	TopicAPIManagement = "masterfabric.api-management"

	// Audit events (consumers write to audit log)
	TopicAudit = "masterfabric.audit"

	// Ticketing / triage events (the classification consumer listens here)
	TopicTriage = "masterfabric.triage"
)

// Event type constants used in Envelope.Type for routing / filtering.
const (
	// IAM
	EventTypeUserRegistered = "user.registered"
	EventTypeUserInvited    = "user.invited"
	EventTypeRoleAssigned   = "role.assigned"
	EventTypeRoleRevoked    = "role.revoked"

	// Tenant
	EventTypeOrgCreated = "organization.created"
	EventTypeAppCreated = "app.created"
	EventTypeAppUpdated = "app.updated"

	// API Management
	EventTypeEndpointCreated = "endpoint.created"
	EventTypeEndpointUpdated = "endpoint.updated"
	EventTypeEndpointRetired = "endpoint.retired"

	// Triage
	EventTypeTicketCreated  = "ticket.created"
	EventTypeTicketUpdated  = "ticket.updated"
	EventTypeTicketAssigned = "ticket.assigned"
)

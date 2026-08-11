package gateway

import "context"

/*
  The values the pipeline hands to the interceptor chain, and the only supported
  way to read them.

  The key type is unexported, so an entry here cannot be reached — or overwritten
  — by any package that does not go through these helpers. That is the whole
  point: with a plain string key, anything in the process holding a "pii_masking"
  string could flip masking off on a request that was supposed to be scrubbed,
  and nothing would report it. staticcheck refuses built-in types as context keys
  (SA1029) for exactly this reason.

  Accessors rather than exported keys, matching internal/shared/middleware: the
  type assertion lives in one place per value instead of at every call site.
*/

type contextKey string

const (
	contextKeyEndpointSchema contextKey = "gateway_endpoint_schema"
	contextKeyPIIMasking     contextKey = "gateway_pii_masking"
	contextKeyEndpointID     contextKey = "gateway_endpoint_id"
	contextKeyAppID          contextKey = "gateway_app_id"
	contextKeyOrgID          contextKey = "gateway_org_id"
	contextKeyUserID         contextKey = "gateway_user_id"
)

// WithEndpointSchema carries the endpoint's JSON schema for the validator.
func WithEndpointSchema(ctx context.Context, schema []byte) context.Context {
	return context.WithValue(ctx, contextKeyEndpointSchema, schema)
}

// EndpointSchemaFromContext returns the endpoint's JSON schema, if one was set.
func EndpointSchemaFromContext(ctx context.Context) ([]byte, bool) {
	schema, ok := ctx.Value(contextKeyEndpointSchema).([]byte)
	return schema, ok
}

// WithPIIMasking carries the endpoint's PII masking flag.
func WithPIIMasking(ctx context.Context, masking bool) context.Context {
	return context.WithValue(ctx, contextKeyPIIMasking, masking)
}

// PIIMaskingFromContext reports whether this request's payloads must be masked.
// A missing value reads as false, so a request that never went through the
// pipeline is not silently treated as already masked.
func PIIMaskingFromContext(ctx context.Context) (bool, bool) {
	masking, ok := ctx.Value(contextKeyPIIMasking).(bool)
	return masking, ok
}

// WithEndpointID carries the managed endpoint's identifier.
func WithEndpointID(ctx context.Context, endpointID string) context.Context {
	return context.WithValue(ctx, contextKeyEndpointID, endpointID)
}

// EndpointIDFromContext returns the managed endpoint's identifier.
func EndpointIDFromContext(ctx context.Context) (string, bool) {
	endpointID, ok := ctx.Value(contextKeyEndpointID).(string)
	return endpointID, ok
}

// WithAppID carries the calling application's identifier.
func WithAppID(ctx context.Context, appID string) context.Context {
	return context.WithValue(ctx, contextKeyAppID, appID)
}

// AppIDFromContext returns the calling application's identifier.
func AppIDFromContext(ctx context.Context) (string, bool) {
	appID, ok := ctx.Value(contextKeyAppID).(string)
	return appID, ok
}

// WithOrgID carries the resolved organization identifier.
func WithOrgID(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, contextKeyOrgID, orgID)
}

// OrgIDFromContext returns the resolved organization identifier.
func OrgIDFromContext(ctx context.Context) (string, bool) {
	orgID, ok := ctx.Value(contextKeyOrgID).(string)
	return orgID, ok
}

// WithUserID carries the authenticated user's identifier.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, contextKeyUserID, userID)
}

// UserIDFromContext returns the authenticated user's identifier.
func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(contextKeyUserID).(string)
	return userID, ok
}

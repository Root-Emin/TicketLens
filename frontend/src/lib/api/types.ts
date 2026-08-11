/*
  TypeScript mirror of the Go DTOs under backend/internal/application (dto packages).
  Field names match the json tags exactly. Two pagination envelopes exist on the
  backend and both are represented: triage lists use ListResponse (meta.page_size
  + meta.total), IAM/tenant lists use PageResult (per_page + total_count).
*/

export type TicketStatus =
  | "open"
  | "in_progress"
  | "pending_customer"
  | "resolved"
  | "closed";

export type TicketPriority = "low" | "normal" | "high" | "urgent";

export type Category =
  | "technical_issue"
  | "integration"
  | "payment_ops"
  | "billing"
  | "onboarding"
  | "how_to"
  | "account_access"
  | "feature_request"
  | "sales"
  | "compliance";

export interface UserInfo {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  status: string;
  created_at: string;
}

export interface LoginResponse {
  token: string;
  user: UserInfo;
}

export interface DepartmentRef {
  id: string;
  name: string;
}

export interface CustomerRef {
  id: string;
  full_name: string;
  email: string;
}

export interface UserRef {
  id: string;
  full_name: string;
  email: string;
}

export interface LatestAnalysis {
  predicted_category: Category | null;
  predicted_priority: string;
  priority_confidence: number;
  department_confidence: number;
  needs_human_review: boolean;
  mapping_fallback: boolean;
  model_name: string;
}

export interface TicketListItem {
  id: string;
  subject: string;
  status: TicketStatus;
  priority: TicketPriority;
  priority_overridden: boolean;
  department_overridden: boolean;
  department: DepartmentRef;
  customer: CustomerRef;
  assignee: UserRef | null;
  message_count: number;
  latest_analysis: LatestAnalysis | null;
  created_at: string;
  updated_at: string;
  resolved_at?: string;
}

export interface MessageInfo {
  id: string;
  ticket_id: string;
  author_type: "customer" | "agent" | "system";
  author_id?: string;
  body: string;
  is_internal: boolean;
  created_at: string;
}

export interface AnalysisInfo {
  id: string;
  ticket_id: string;
  predicted_priority: string;
  priority_confidence: number;
  predicted_category: Category | null;
  predicted_department_id?: string;
  mapping_fallback: boolean;
  department_confidence: number;
  needs_human_review: boolean;
  model_name: string;
  model_version: string;
  raw_response?: unknown;
  created_at: string;
}

export interface TicketDetail extends TicketListItem {
  messages: MessageInfo[];
  analyses: AnalysisInfo[];
  /**
   * The confidence cutoff currently configured on the backend
   * (CLASSIFIER_REVIEW_THRESHOLD), used to draw the marker on a confidence bar.
   *
   * It reflects the setting as it stands now, not the one each analysis was
   * judged under. For "was this flagged", read `needs_human_review` on the
   * analysis itself — that is the verdict the backend actually recorded.
   */
  review_threshold: number;
}

export interface DepartmentInfo {
  id: string;
  name: string;
  description: string;
  category: Category | null;
  is_default: boolean;
  /** Every ticket ever routed here, not just the open ones. */
  ticket_count: number;
  /** People assigned to this department. Portal customers are not counted. */
  staff_count: number;
}

/**
 * One person on the support roster — dto.StaffInfo, from GET /staff.
 *
 * Deliberately not UserInfo. That one is IAM's and describes any account,
 * including a customer's; this is triage's view of a colleague, and it is the
 * only shape that carries a department.
 */
export interface StaffInfo {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  /** Precomputed server-side, falling back to the email. */
  full_name: string;
  status: string;
  /** null when the person is on the roster but on no team. */
  department: DepartmentRef | null;
  created_at: string;
}

export interface StatsOverview {
  tickets: {
    total: number;
    open: number;
    in_progress: number;
    pending_customer: number;
    resolved: number;
    closed: number;
  };
  by_priority: {
    low: number;
    normal: number;
    high: number;
    urgent: number;
  };
  by_category: Record<string, number>;
  by_department: { department_id: string; name: string; count: number }[];
  ai: {
    analyzed: number;
    priority_accept_rate: number;
    department_accept_rate: number;
    avg_priority_confidence: number;
    needs_review: number;
    unmapped_count: number;
  };
}

/** Triage list envelope: {data, meta:{page,page_size,total}}. */
export interface ListResponse<T> {
  data: T[];
  meta: { page: number; page_size: number; total: number };
}

/** Unpaginated collection envelope: {data}. */
export interface CollectionResponse<T> {
  data: T[];
}

/**
 * IAM/tenant list envelope: {data, page, per_page, total_count, total_pages}.
 *
 * Flat rather than nested under `meta`, unlike ListResponse above. The two
 * shapes are a backend fact (shared/pagination.Result vs triage's own dto), not
 * a modelling choice here, so both are spelled out rather than reconciled.
 */
export interface PageResult<T> {
  data: T[];
  page: number;
  per_page: number;
  total_count: number;
  total_pages: number;
}

/** Organization identity, from GET /organizations. */
export interface OrgInfo {
  id: string;
  name: string;
  slug: string;
  status: string;
  created_at: string;
}

export interface TicketFilters {
  status?: TicketStatus[];
  priority?: TicketPriority[];
  department_id?: string;
  assignee_id?: string;
  needs_review?: boolean;
  overridden?: boolean;
  q?: string;
  sort?: string;
  page?: number;
  page_size?: number;
}

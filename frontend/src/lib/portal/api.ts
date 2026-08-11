import { api, buildTicketQuery } from "@/lib/api/client";
import type {
  ListResponse,
  MessageInfo,
  TicketDetail,
  UserInfo,
} from "@/lib/api/types";

import { PAGE_SIZE, sortParamFor, statusesFor } from "./filters";
import type { PortalTicketListItem, PortalTicketQuery } from "./types";

/*
  Every backend call the customer portal makes, in one place.

  Nothing here talks to the network directly: it all goes through lib/api/client,
  which routes via /api/proxy/* so the JWT stays in an httpOnly cookie. Scoping
  is the backend's job — a customer token must resolve to that customer's own
  tickets. The portal never passes a customer id, because a client-supplied
  owner filter is not a filter, it is a hole.

  Endpoints marked ASSUMED do not exist in their portal form yet. They are
  listed in the handover report; each one degrades to a clear error rather than
  a broken screen.
*/

export interface CreateTicketInput {
  subject: string;
  body: string;
}

export const portalApi = {
  /** The signed-in account. */
  me: () => api.get<UserInfo>("/me"),

  /**
   * The caller's own tickets.
   *
   * ASSUMED: `GET /tickets` auto-filters to the caller when the token carries
   * only `ticket:read_own`. The filter exists on the repository side already —
   * see the TODO(customer-portal) in list_tickets.go — but the customer token
   * that would pin it does not.
   */
  listTickets: (query: PortalTicketQuery) =>
    api.get<ListResponse<PortalTicketListItem>>(
      `/tickets${buildTicketQuery({
        q: query.q,
        status: statusesFor(query.status),
        sort: sortParamFor(query.sort),
        page: query.page,
        page_size: PAGE_SIZE,
      })}`,
    ),

  /** A shallow page used only to derive the dashboard numbers. */
  listForStats: () =>
    api.get<ListResponse<PortalTicketListItem>>(
      `/tickets${buildTicketQuery({ page: 1, page_size: 100, sort: "-created_at" })}`,
    ),

  getTicket: (id: string) => api.get<TicketDetail>(`/tickets/${id}`),

  /**
   * Raise a ticket.
   *
   * ASSUMED: `POST /tickets` accepts no `customer_id` from a customer token and
   * takes the owner from the claims instead. The DTO marks customer_id required
   * today, so this is the one call that certainly needs a backend change.
   *
   * Priority, category and department are deliberately absent: the classifier
   * sets them, which is the whole point of the product.
   */
  createTicket: (input: CreateTicketInput) =>
    api.post<TicketDetail>("/tickets", {
      subject: input.subject.trim(),
      body: input.body.trim(),
    }),

  /** Reply on an open ticket. Customers can never write an internal note. */
  reply: (id: string, body: string) =>
    api.post<MessageInfo>(`/tickets/${id}/messages`, {
      body: body.trim(),
      is_internal: false,
    }),

  /**
   * Reopen a resolved ticket.
   *
   * ASSUMED: `PATCH /tickets/{id}` is reachable with a customer token for a
   * status change on their own ticket. The route requires `ticket:update`,
   * which per the contract only staff hold, so this returns 403 until the
   * backend grants something narrower (a `ticket:reopen_own`, or a dedicated
   * POST /tickets/{id}/reopen). The UI handles that answer.
   */
  reopen: (id: string) =>
    api.patch<TicketDetail>(`/tickets/${id}`, { status: "open" }),

  /**
   * ASSUMED: there is no password endpoint on the backend at all today
   * (`/auth` exposes register and login only). The form reports the failure
   * rather than pretending the change went through.
   */
  changePassword: (input: { current_password: string; new_password: string }) =>
    api.post<{ ok: boolean }>("/auth/change-password", input),
};

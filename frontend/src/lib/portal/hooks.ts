import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import type { TicketDetail } from "@/lib/api/types";

import { portalApi, type CreateTicketInput } from "./api";
import { computeStats } from "./stats";
import type { PortalStats, PortalTicketQuery } from "./types";

/*
  React Query bindings for the portal.

  Keys are namespaced under "portal" so they never collide with the staff-facing
  keys in lib/api/hooks.ts — the two panels ask the same endpoints different
  questions and must not share a cache entry.
*/

export const portalKeys = {
  all: ["portal"] as const,
  me: ["portal", "me"] as const,
  tickets: (query: PortalTicketQuery) => ["portal", "tickets", query] as const,
  ticket: (id: string) => ["portal", "ticket", id] as const,
  stats: ["portal", "stats"] as const,
};

export function usePortalMe() {
  return useQuery({
    queryKey: portalKeys.me,
    queryFn: () => portalApi.me(),
    staleTime: 5 * 60 * 1000,
  });
}

export function usePortalTickets(query: PortalTicketQuery) {
  return useQuery({
    queryKey: portalKeys.tickets(query),
    queryFn: () => portalApi.listTickets(query),
    // Keeping the previous page on screen while the next one loads is what
    // stops the list from collapsing to a skeleton on every filter click.
    placeholderData: (previous) => previous,
  });
}

export function usePortalTicket(id: string) {
  return useQuery({
    queryKey: portalKeys.ticket(id),
    queryFn: () => portalApi.getTicket(id),
    enabled: Boolean(id),
  });
}

/** The dashboard's numbers plus the five most recent tickets, in one request. */
export function usePortalOverview() {
  return useQuery({
    queryKey: portalKeys.stats,
    queryFn: async () => {
      const page = await portalApi.listForStats();
      const stats: PortalStats = computeStats(page.data);
      return { stats, recent: page.data.slice(0, 5), total: page.meta.total };
    },
  });
}

export function useCreateTicket() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateTicketInput) => portalApi.createTicket(input),
    onSuccess: (ticket: TicketDetail) => {
      // Seed the detail cache so the redirect lands on a rendered ticket
      // instead of a spinner over data we already hold.
      qc.setQueryData(portalKeys.ticket(ticket.id), ticket);
      qc.invalidateQueries({ queryKey: portalKeys.all });
    },
  });
}

export function useReplyToTicket(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: string) => portalApi.reply(id, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: portalKeys.ticket(id) });
      qc.invalidateQueries({ queryKey: portalKeys.stats });
    },
  });
}

export function useReopenTicket(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => portalApi.reopen(id),
    onSuccess: (ticket) => {
      qc.setQueryData(portalKeys.ticket(id), ticket);
      qc.invalidateQueries({ queryKey: portalKeys.all });
    },
  });
}

export function useChangePassword() {
  return useMutation({
    mutationFn: (input: { current_password: string; new_password: string }) =>
      portalApi.changePassword(input),
  });
}

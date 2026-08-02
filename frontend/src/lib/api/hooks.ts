import {
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";

import { api, buildTicketQuery } from "./client";
import type {
  CollectionResponse,
  DepartmentInfo,
  ListResponse,
  MessageInfo,
  StatsOverview,
  TicketDetail,
  TicketFilters,
  TicketListItem,
  UserInfo,
} from "./types";

export const keys = {
  me: ["me"] as const,
  tickets: (f: TicketFilters) => ["tickets", f] as const,
  ticket: (id: string) => ["ticket", id] as const,
  departments: ["departments"] as const,
  stats: (range?: { from?: string; to?: string }) => ["stats", range] as const,
};

export function useMe() {
  return useQuery({
    queryKey: keys.me,
    queryFn: () => api.get<UserInfo>("/me"),
    staleTime: 5 * 60 * 1000,
  });
}

export function useTickets(filters: TicketFilters) {
  return useQuery({
    queryKey: keys.tickets(filters),
    queryFn: () =>
      api.get<ListResponse<TicketListItem>>(
        `/tickets${buildTicketQuery(filters as Record<string, unknown>)}`,
      ),
    placeholderData: (prev) => prev,
  });
}

export function useTicket(id: string) {
  return useQuery({
    queryKey: keys.ticket(id),
    queryFn: () => api.get<TicketDetail>(`/tickets/${id}`),
    enabled: Boolean(id),
  });
}

export function useDepartments() {
  return useQuery({
    queryKey: keys.departments,
    queryFn: () => api.get<CollectionResponse<DepartmentInfo>>("/departments"),
    staleTime: 5 * 60 * 1000,
  });
}

export function useStats(range?: { from?: string; to?: string }) {
  return useQuery({
    queryKey: keys.stats(range),
    queryFn: () =>
      api.get<StatsOverview>(
        `/stats/overview${buildTicketQuery({ ...range } as Record<string, unknown>)}`,
      ),
  });
}

interface UpdatePayload {
  priority?: string;
  department_id?: string;
  status?: string;
}

export function useUpdateTicket(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: UpdatePayload) =>
      api.patch<TicketDetail>(`/tickets/${id}`, payload),
    onSuccess: (data) => {
      qc.setQueryData(keys.ticket(id), data);
      qc.invalidateQueries({ queryKey: ["tickets"] });
      qc.invalidateQueries({ queryKey: ["stats"] });
    },
  });
}

export function useAssignTicket(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (assigneeId: string | null) =>
      api.post<TicketDetail>(`/tickets/${id}/assign`, { assignee_id: assigneeId }),
    onSuccess: (data) => {
      qc.setQueryData(keys.ticket(id), data);
      qc.invalidateQueries({ queryKey: ["tickets"] });
    },
  });
}

export function useAddMessage(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload: { body: string; is_internal: boolean }) =>
      api.post<MessageInfo>(`/tickets/${id}/messages`, payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: keys.ticket(id) });
    },
  });
}

export function useReanalyze(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.post<TicketDetail>(`/tickets/${id}/analyses`, {}),
    onSuccess: (data) => {
      qc.setQueryData(keys.ticket(id), data);
      qc.invalidateQueries({ queryKey: ["tickets"] });
      qc.invalidateQueries({ queryKey: ["stats"] });
    },
  });
}

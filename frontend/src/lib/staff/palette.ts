import type { Priority, Ticket, TicketStatus } from "./types";

/*
  The narrow ticket shape the command palette needs.

  Kept out of the palette component because that module is "use client": a
  server layout cannot call a function exported from a client module, only
  render it. This file carries no directive, so both sides can use it.
*/

export interface PaletteTicket {
  id: string;
  subject: string;
  customer: string;
  priority: Priority;
  status: TicketStatus;
}

export function paletteTicketsFrom(list: Ticket[]): PaletteTicket[] {
  return list.map((ticket) => ({
    id: ticket.id,
    subject: ticket.subject,
    customer: ticket.customer.name,
    priority: ticket.priority,
    status: ticket.status,
  }));
}

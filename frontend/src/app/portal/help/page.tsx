"use client";

import { BookOpen, Mail, MessageSquareText, Sparkles } from "lucide-react";

import { FaqList } from "@/components/portal/help/faq-list";
import { ActionLink, PageHeader } from "@/components/portal/primitives";
import { NewTicketDialog } from "@/components/portal/tickets/new-ticket-dialog";
import { Panel, PanelHeader, PanelSection } from "@/components/staff/primitives";
import {
  CONTACT_CHANNELS,
  FAQ,
  GETTING_STARTED,
} from "@/lib/portal/help-content";

/*
  The Help Center.

  Three sections, in the order someone actually needs them: how this works, the
  questions they are likely to have, and how to reach a person when neither
  helped. The content is data (lib/portal/help-content.ts) rather than markup,
  which is what leaves room for the obvious next step — an endpoint that returns
  the articles most relevant to a customer's open tickets, rendered by the same
  FaqList this page already uses.
*/

export default function PortalHelpPage() {
  return (
    <>
      <PageHeader
        title="Help Center"
        description="How TicketLens works, and how to get an answer fast."
        action={<NewTicketDialog />}
      />

      {/* Getting started */}
      <Panel>
        <PanelHeader
          title={
            <span className="inline-flex items-center gap-2">
              <BookOpen className="size-[18px] text-tl-blue" strokeWidth={1.9} aria-hidden />
              Getting started
            </span>
          }
        />
        <PanelSection>
          <ol className="grid gap-5 sm:grid-cols-3">
            {GETTING_STARTED.map((step, index) => (
              <li key={step.id}>
                <span
                  className="inline-flex size-7 items-center justify-center rounded-full bg-tl-blue-soft text-ui-sm font-bold text-tl-blue tabular-nums"
                  aria-hidden
                >
                  {index + 1}
                </span>
                <h3 className="mt-2.5 text-ui-md font-semibold text-tl-ink">
                  {step.title}
                </h3>
                <p className="mt-1 text-ui-sm leading-relaxed text-tl-muted">
                  {step.description}
                </p>
              </li>
            ))}
          </ol>
        </PanelSection>
      </Panel>

      {/* FAQ */}
      <Panel>
        <PanelHeader
          title={
            <span className="inline-flex items-center gap-2">
              <MessageSquareText
                className="size-[18px] text-tl-blue"
                strokeWidth={1.9}
                aria-hidden
              />
              Frequently asked
            </span>
          }
        />
        <PanelSection>
          <FaqList articles={FAQ} />
        </PanelSection>
      </Panel>

      {/* Contact */}
      <Panel>
        <PanelHeader
          title={
            <span className="inline-flex items-center gap-2">
              <Mail className="size-[18px] text-tl-blue" strokeWidth={1.9} aria-hidden />
              Contact support
            </span>
          }
        />
        <PanelSection className="space-y-4">
          <ul className="grid gap-4 sm:grid-cols-2">
            {CONTACT_CHANNELS.map((channel) => (
              <li
                key={channel.id}
                className="rounded-card border border-tl-line bg-tl-canvas/60 px-4 py-3.5"
              >
                <h3 className="text-ui-md font-semibold text-tl-ink">
                  {channel.title}
                </h3>
                <p className="mt-1 text-ui-sm leading-relaxed text-tl-muted">
                  {channel.description}
                </p>
              </li>
            ))}
          </ul>

          <div className="flex flex-wrap items-center gap-3 rounded-card bg-tl-blue-soft/60 px-4 py-3.5">
            <Sparkles
              className="size-[18px] shrink-0 text-tl-blue"
              strokeWidth={1.9}
              aria-hidden
            />
            <p className="min-w-0 flex-1 text-ui-sm leading-relaxed text-tl-ink-soft">
              Not sure who to ask? Just describe the problem — we work out the
              right team for you.
            </p>
            <ActionLink href="/portal/tickets" variant="secondary" size="sm">
              See my tickets
            </ActionLink>
          </div>
        </PanelSection>
      </Panel>
    </>
  );
}

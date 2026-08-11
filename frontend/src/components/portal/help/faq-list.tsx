"use client";

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/shadcn/accordion";
import type { HelpArticle } from "@/lib/portal/help-content";

/**
 * A list of question-and-answer articles.
 *
 * Takes its articles as a prop rather than importing them, so the same
 * component renders the static FAQ today and an AI-suggested set tomorrow —
 * see lib/portal/help-content.ts.
 *
 * `type="single"` with `collapsible`: opening a second answer closes the first,
 * which keeps the list scannable instead of turning into a wall.
 */
export function FaqList({ articles }: { articles: HelpArticle[] }) {
  return (
    <Accordion type="single" collapsible className="w-full">
      {articles.map((article) => (
        <AccordionItem key={article.id} value={article.id}>
          <AccordionTrigger className="text-left text-ui-md font-semibold text-tl-ink">
            {article.question}
          </AccordionTrigger>
          <AccordionContent className="text-ui-md leading-relaxed text-tl-muted">
            {article.answer}
          </AccordionContent>
        </AccordionItem>
      ))}
    </Accordion>
  );
}

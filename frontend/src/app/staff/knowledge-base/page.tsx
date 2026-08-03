import { BookOpen } from "lucide-react";

import { PlaceholderPage } from "@/components/staff/placeholder-page";

export default function KnowledgeBasePage() {
  return (
    <PlaceholderPage
      icon={BookOpen}
      title="Knowledge Base"
      description="Canned responses and internal runbooks, searchable from the composer so an answer is never rewritten twice."
      planned={[
        "Searchable articles with categories and ownership",
        "Insert an article into a reply without leaving the ticket",
        "Suggested articles based on the ticket's predicted category",
        "Flag articles that go stale when a linked ticket reopens",
      ]}
    />
  );
}

/*
  Help Center content.

  Kept as data rather than JSX so the same shapes can come from an API later
  without touching a component. That is the whole reason this file exists: when
  the classifier starts suggesting articles for a customer's draft, the endpoint
  returns `HelpArticle[]`, `<ArticleList>` renders them, and nothing else moves.

  `id` is the stable handle a future suggestion endpoint would refer to, so it
  stays fixed even when the wording changes.
*/

export interface HelpArticle {
  id: string;
  question: string;
  answer: string;
}

export interface HelpStep {
  id: string;
  title: string;
  description: string;
}

export const FAQ: HelpArticle[] = [
  {
    id: "how-routing-works",
    question: "How does my ticket reach the right team?",
    answer:
      "When you submit a request, our classifier reads the subject and description and predicts both the topic and how urgent it is. That prediction picks the department and the starting priority, which is why you are never asked to choose either. A support agent reviews the routing and can move a ticket if the model got it wrong.",
  },
  {
    id: "why-no-priority",
    question: "Why can't I set the priority myself?",
    answer:
      "Because a priority everyone can set stops meaning anything. Urgency is judged from what you describe — an outage reads as urgent whether or not the word appears — so the queue reflects real impact rather than who marked their ticket loudest.",
  },
  {
    id: "response-times",
    question: "How quickly will someone reply?",
    answer:
      "Most requests get a first reply the same working day, and urgent ones are picked up sooner. Your dashboard shows the average turnaround across your own tickets, so you can see what to expect rather than a promise made in general.",
  },
  {
    id: "ticket-statuses",
    question: "What do the ticket statuses mean?",
    answer:
      "Open means we have it and no one has started yet. In Progress means an agent is working on it. Waiting Customer means the ball is with you — we have asked something and are holding until you answer. Resolved means we believe it is done; you can reopen it if it is not.",
  },
  {
    id: "add-information",
    question: "Can I add information after submitting?",
    answer:
      "Yes. Open the ticket and reply in the conversation. Everything you add goes to the agent handling it, and the ticket returns to the top of their queue.",
  },
  {
    id: "reopen",
    question: "What if a resolved ticket comes back?",
    answer:
      "Open the ticket and choose Reopen. The conversation, the history and the routing all carry over, so you never have to explain the problem from the beginning again.",
  },
];

export const GETTING_STARTED: HelpStep[] = [
  {
    id: "describe",
    title: "Describe what happened",
    description:
      "Create a ticket with a short subject and as much detail as you can: what you expected, what happened instead, and anything you have already tried.",
  },
  {
    id: "routed",
    title: "We route it automatically",
    description:
      "Our AI reads the request, works out the topic and urgency, and sends it to the team that handles that kind of problem — usually within seconds.",
  },
  {
    id: "follow",
    title: "Follow it in one thread",
    description:
      "Every reply lands in the same conversation. You will see the status change as the ticket moves, and you can add information at any point.",
  },
];

/** Where a customer can reach a person, when the portal is not enough. */
export const CONTACT_CHANNELS = [
  {
    id: "ticket",
    title: "Open a ticket",
    description: "The fastest route. Tracked, routed and answered in writing.",
  },
  {
    id: "email",
    title: "support@ticketlens.dev",
    description: "Prefer email? Write to us and we will open the ticket for you.",
  },
] as const;

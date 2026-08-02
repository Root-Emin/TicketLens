import { redirect } from "next/navigation";

// The queue is the agent's home. Auth gating happens in src/proxy.ts; an
// unauthenticated visitor is bounced to /login before this renders.
export default function Home() {
  redirect("/tickets");
}

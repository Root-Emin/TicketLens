import { redirect } from "next/navigation";

// The staff panel is the app's front door while the other roles are being
// built. Auth gating happens in src/proxy.ts.
export default function Home() {
  redirect("/staff");
}

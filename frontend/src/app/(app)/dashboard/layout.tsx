/** Same palette wrapper as the queue — see app/(app)/tickets/layout.tsx. */
export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="legacy-theme min-h-full">{children}</div>;
}

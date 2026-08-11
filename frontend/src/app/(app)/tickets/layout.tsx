/*
  The queue's palette wrapper.

  /tickets and /tickets/[id] were written against token names — `bg-surface`,
  `text-accent`, `border-border` — that shadcn later claimed globally. The
  .legacy-theme class re-declares them so those screens keep resolving, and as of
  globals.css §2 it resolves them to the management panel's own tl-* palette.
  The result is that the queue is drawn in the same colours as /team and
  /departments without a line of its markup being rewritten, and without the
  wrapper leaking onto the new screens, whose shadcn components (skeletons,
  menus, dialogs) would otherwise inherit an accent that is no longer neutral.

  A layout rather than a class on each page: both screens have several early
  returns for their loading and error branches, and one of them would have been
  missed.
*/
export default function TicketsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="legacy-theme min-h-full">{children}</div>;
}

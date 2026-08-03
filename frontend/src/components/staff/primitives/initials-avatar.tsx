import { cn } from "@/lib/utils";

/*
  Initials avatar.

  The reference design uses photographs; there are no image assets in the repo
  and pulling from an external host would be a privacy and CSP problem. Initials
  on a flat tint is the honest fallback, and `src` is accepted so real images can
  be dropped in later without touching call sites.

  The tint is chosen from a hash of the name, so a given person keeps the same
  colour everywhere on the screen instead of changing between the list, the
  thread and the details panel.
*/

const TINTS = [
  "bg-blue-100 text-blue-700",
  "bg-emerald-100 text-emerald-700",
  "bg-amber-100 text-amber-700",
  "bg-violet-100 text-violet-700",
  "bg-rose-100 text-rose-700",
  "bg-cyan-100 text-cyan-700",
];

function tintFor(seed: string): string {
  let hash = 0;
  for (let i = 0; i < seed.length; i += 1) {
    hash = (hash * 31 + seed.charCodeAt(i)) >>> 0;
  }
  return TINTS[hash % TINTS.length];
}

export function InitialsAvatar({
  name,
  initials,
  size = 32,
  className,
}: {
  name: string;
  initials: string;
  size?: number;
  className?: string;
}) {
  return (
    <span
      style={{
        width: size,
        height: size,
        fontSize: Math.max(9, Math.round(size * 0.36)),
      }}
      className={cn(
        "inline-flex shrink-0 select-none items-center justify-center rounded-full font-semibold",
        tintFor(name),
        className,
      )}
      aria-hidden
    >
      {initials}
    </span>
  );
}

/** Avatar with a presence dot cut into its lower-right corner. */
export function PresenceAvatar({
  name,
  initials,
  size = 36,
  ringClass = "border-tl-canvas",
  className,
}: {
  name: string;
  initials: string;
  size?: number;
  /** Border colour of the dot's cut-out, matched to whatever sits behind it. */
  ringClass?: string;
  className?: string;
}) {
  return (
    <span className={cn("relative inline-flex shrink-0", className)}>
      <InitialsAvatar name={name} initials={initials} size={size} />
      <span
        className={cn(
          "absolute bottom-0 right-0 size-[11px] rounded-full border-2 bg-tl-green",
          ringClass,
        )}
        aria-hidden
      />
    </span>
  );
}

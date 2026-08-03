import { cn } from "@/lib/utils";

/**
 * Sparkline. Samples are 0..1 oldest-first; the path is built in a fixed
 * viewBox and stretched, so it stays crisp at any card width.
 *
 * Purely decorative — the number beside it carries the meaning — so it is
 * hidden from assistive technology rather than given a label nobody wants read
 * out six times on the dashboard.
 */
export function Sparkline({
  points,
  stroke,
  className,
}: {
  points: number[];
  stroke: string;
  className?: string;
}) {
  const w = 100;
  const h = 28;
  const pad = 3;

  if (points.length < 2) return null;

  const step = w / (points.length - 1);
  const d = points
    .map((p, i) => {
      const x = i * step;
      const y = h - pad - p * (h - pad * 2);
      return `${i === 0 ? "M" : "L"}${x.toFixed(1)} ${y.toFixed(1)}`;
    })
    .join(" ");

  return (
    <svg
      viewBox={`0 0 ${w} ${h}`}
      preserveAspectRatio="none"
      aria-hidden
      focusable="false"
      className={cn("h-7 w-16", className)}
    >
      <path
        d={d}
        fill="none"
        stroke={stroke}
        strokeWidth={2}
        strokeLinecap="round"
        strokeLinejoin="round"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
}

/*
  The lens.

  Concentric rings drawn with borders — no image, no SVG asset, no gradient
  stack. The name of the product is the metaphor: a lens brought onto a support
  queue, so the brand panel is an aperture with one blade lit in the brand blue.

  Every inset stays under 50%. At exactly 50% a ring has zero width, and past it
  the box collapses and the element silently stops rendering — which is the one
  way this can look broken without throwing anything.

  Purely decorative, so the whole block is aria-hidden and never announced.
*/

interface RingSpec {
  /** Distance from each edge of the square, as a percentage. Must stay < 50. */
  inset: number;
  className: string;
}

const RINGS: RingSpec[] = [
  { inset: 0, className: "border-white/[0.05]" },
  { inset: 9, className: "border-white/[0.07]" },
  { inset: 18, className: "border-white/10" },
  { inset: 27, className: "border-white/[0.13]" },
  { inset: 36, className: "border-tl-blue/35" },
];

export function LensRings({ className }: { className?: string }) {
  return (
    <div className={className} aria-hidden>
      <div className="relative aspect-square w-full">
        {RINGS.map((ring) => (
          <span
            key={ring.inset}
            className={`absolute rounded-full border ${ring.className}`}
            style={{ inset: `${ring.inset}%` }}
          />
        ))}

        {/* The lit blade: a quarter arc on the innermost ring, rotated, so the
            glass reads as catching light without anything animating. */}
        <span
          className="absolute rounded-full border-2 border-transparent border-t-tl-blue"
          style={{ inset: "36%", rotate: "40deg" }}
        />

        {/* Aperture: a soft halo around a solid core. */}
        <span className="absolute rounded-full bg-tl-blue/15" style={{ inset: "41%" }} />
        <span className="absolute rounded-full bg-tl-blue" style={{ inset: "45.5%" }} />
      </div>
    </div>
  );
}

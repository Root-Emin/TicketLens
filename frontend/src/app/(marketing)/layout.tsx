import { Archivo } from "next/font/google";

/*
  Archivo is loaded here rather than in the root layout so that /staff, /portal
  and the legacy panels never pay for a face they do not set a single word in.
  The variable is scoped to the wrapper below, which is all `.type-display`
  needs to resolve.

  `wdth` is requested explicitly: next/font ships only the weight axis by
  default, and without the width axis `.type-display` would silently render at
  normal width — the one thing the face was picked for.
*/
const archivo = Archivo({
  variable: "--font-archivo",
  subsets: ["latin", "latin-ext"],
  axes: ["wdth"],
  display: "swap",
});

export default function MarketingLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className={archivo.variable}>{children}</div>;
}

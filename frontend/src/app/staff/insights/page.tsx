import { Sparkles } from "lucide-react";

import { PlaceholderPage } from "@/components/staff/placeholder-page";

export default function InsightsPage() {
  return (
    <PlaceholderPage
      icon={Sparkles}
      title="AI Insights"
      description="How well the triage model is doing its job, and where a human keeps having to correct it."
      planned={[
        "Prediction accuracy tracked against staff corrections",
        "Confidence distribution, and which categories sit lowest",
        "Tickets the model was unsure about, queued for review",
        "Drift alerts when accuracy falls for a given category",
      ]}
    />
  );
}

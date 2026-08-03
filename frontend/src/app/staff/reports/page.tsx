import { FileText } from "lucide-react";

import { PlaceholderPage } from "@/components/staff/placeholder-page";

export default function ReportsPage() {
  return (
    <PlaceholderPage
      icon={FileText}
      title="Reports"
      description="Queue health over time, rather than the single snapshot the dashboard shows today."
      planned={[
        "First response and resolution times by team and period",
        "SLA attainment, with the breaches broken out",
        "Volume by category, subcategory and customer",
        "Scheduled exports to CSV on the workspace timezone",
      ]}
    />
  );
}

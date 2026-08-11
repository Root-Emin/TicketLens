"use client";

import { Panel, PanelHeader, PanelSection } from "@/components/staff/primitives";
import { usePersistentState } from "@/hooks/use-persistent-state";
import { cn } from "@/lib/utils";

/*
  Notification preferences.

  There is no preferences endpoint, so these are stored in localStorage through
  the hook the staff rail already uses for its collapsed state. That is a real
  preference — it survives a reload on this device — and the note under the list
  says exactly what it is rather than implying the choice reached a server.

  When the API arrives, `usePersistentState` becomes a mutation and the note
  goes; the markup does not change.
*/

interface Preference {
  id: string;
  label: string;
  description: string;
  defaultValue: boolean;
}

const PREFERENCES: Preference[] = [
  {
    id: "reply",
    label: "Replies from support",
    description: "Email me when someone answers one of my tickets.",
    defaultValue: true,
  },
  {
    id: "status",
    label: "Status changes",
    description: "Email me when a ticket is resolved or reopened.",
    defaultValue: true,
  },
  {
    id: "digest",
    label: "Weekly summary",
    description: "A short recap of everything still open on my account.",
    defaultValue: false,
  },
];

const STORAGE_KEY = "tl.portal.notifications";

/** The stored shape: preference id → enabled. */
type PreferenceState = Record<string, boolean>;

const DEFAULTS: PreferenceState = Object.fromEntries(
  PREFERENCES.map((preference) => [preference.id, preference.defaultValue]),
);

export function NotificationPreferences() {
  const [state, setState] = usePersistentState<PreferenceState>(
    STORAGE_KEY,
    DEFAULTS,
  );

  const toggle = (id: string, enabled: boolean) =>
    setState({ ...DEFAULTS, ...state, [id]: enabled });

  return (
    <Panel>
      <PanelHeader title="Notification preferences" />
      <PanelSection className="space-y-4">
        <ul className="space-y-1">
          {PREFERENCES.map((preference) => {
            const enabled = state[preference.id] ?? preference.defaultValue;
            return (
              <li key={preference.id}>
                <label
                  className={cn(
                    "flex cursor-pointer items-start gap-3 rounded-btn px-2 py-2.5 transition-colors duration-150",
                    "hover:bg-tl-line-soft/60",
                  )}
                >
                  <input
                    type="checkbox"
                    checked={enabled}
                    onChange={(event) =>
                      toggle(preference.id, event.target.checked)
                    }
                    className="mt-0.5 size-4 shrink-0 rounded border-tl-line accent-tl-blue focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
                  />
                  <span className="min-w-0">
                    <span className="block text-ui-md font-medium text-tl-ink">
                      {preference.label}
                    </span>
                    <span className="mt-0.5 block text-ui-sm text-tl-muted">
                      {preference.description}
                    </span>
                  </span>
                </label>
              </li>
            );
          })}
        </ul>

        <p className="text-ui-xs text-tl-faint">
          Saved on this device for now — these preferences aren&apos;t synced to
          your account yet.
        </p>
      </PanelSection>
    </Panel>
  );
}

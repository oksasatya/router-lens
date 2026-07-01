import { BarChart3, FolderKanban, LayoutDashboard, ScrollText, Settings, Tag } from "lucide-react";
import type { ComponentType } from "react";

export interface NavItem {
  readonly to: string;
  readonly key: string; // i18n key
  readonly icon: ComponentType<{ className?: string }>;
}

// Single source of truth for primary navigation — shared by the desktop sidebar
// and the mobile drawer (no duplicated link lists).
export const NAV: readonly NavItem[] = [
  { to: "/", key: "nav.dashboard", icon: LayoutDashboard },
  { to: "/logs", key: "nav.logs", icon: ScrollText },
  { to: "/analytics", key: "nav.analytics", icon: BarChart3 },
  { to: "/projects", key: "nav.projects", icon: FolderKanban },
  { to: "/pricing", key: "nav.pricing", icon: Tag },
  { to: "/settings", key: "nav.settings", icon: Settings },
];

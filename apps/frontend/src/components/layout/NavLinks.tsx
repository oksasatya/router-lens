import { Link } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { NAV } from "./nav";

export function NavLinks({ onNavigate }: { readonly onNavigate?: () => void }) {
  const { t } = useTranslation();
  return (
    <nav className="flex flex-col gap-1">
      {NAV.map(({ to, key, icon: Icon }) => (
        <Link
          key={to}
          to={to}
          onClick={onNavigate}
          activeOptions={{ exact: to === "/" }}
          className="flex min-h-11 items-center gap-3 rounded-lg px-3 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          activeProps={{ className: "bg-accent text-foreground" }}
        >
          <Icon className="size-4 shrink-0" />
          <span>{t(key)}</span>
        </Link>
      ))}
    </nav>
  );
}

import { useTranslation } from "react-i18next";
import { NavLinks } from "./NavLinks";

export function Sidebar() {
  const { t } = useTranslation();
  return (
    <aside className="hidden w-60 shrink-0 flex-col border-r border-border bg-sidebar md:flex">
      <div className="flex h-16 items-center gap-2.5 px-5">
        <img src="/logo.webp" alt="" className="size-7 rounded-md" />
        <span className="font-heading text-lg font-semibold tracking-tight">{t("app.name")}</span>
      </div>
      <div className="flex-1 px-3 py-2">
        <NavLinks />
      </div>
    </aside>
  );
}

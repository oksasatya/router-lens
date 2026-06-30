import { Menu } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { LanguageToggle } from "@/components/LanguageToggle";
import { ThemeToggle } from "@/components/ThemeToggle";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { NavLinks } from "./NavLinks";

export function Topbar() {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  return (
    <header className="flex h-16 items-center gap-2 border-b border-border px-4 md:px-6">
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetTrigger
          render={<Button variant="ghost" size="icon" className="md:hidden" aria-label={t("common.menu")} />}
        >
          <Menu className="size-5" />
        </SheetTrigger>
        <SheetContent side="left" className="w-64 p-0">
          <SheetTitle className="flex h-16 items-center gap-2.5 px-5 font-heading text-lg font-semibold">
            <img src="/logo.webp" alt="" className="size-7 rounded-md" />
            {t("app.name")}
          </SheetTitle>
          <div className="px-3">
            <NavLinks onNavigate={() => setOpen(false)} />
          </div>
        </SheetContent>
      </Sheet>
      <div className="flex-1" />
      <ThemeToggle />
      <LanguageToggle />
      <Avatar className="size-8">
        <AvatarFallback className="bg-primary/15 text-xs text-primary">RL</AvatarFallback>
      </Avatar>
    </header>
  );
}

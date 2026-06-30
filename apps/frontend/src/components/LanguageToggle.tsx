import GB from "country-flag-icons/react/3x2/GB";
import ID from "country-flag-icons/react/3x2/ID";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

const FLAGS = { en: GB, id: ID } as const;
type Lang = keyof typeof FLAGS;

export function LanguageToggle() {
  const { i18n, t } = useTranslation();
  const lang: Lang = i18n.language === "id" ? "id" : "en";
  const Current = FLAGS[lang];

  const change = (l: Lang) => {
    void i18n.changeLanguage(l);
    globalThis.localStorage?.setItem("lang", l);
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="ghost" size="icon" aria-label={t("common.language")} />}>
        <Current className="h-4 w-6 rounded-[2px]" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem onClick={() => change("en")}>
          <GB className="h-3.5 w-5 rounded-[2px]" /> English
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => change("id")}>
          <ID className="h-3.5 w-5 rounded-[2px]" /> Indonesia
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

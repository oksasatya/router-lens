import { Eye, EyeOff } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

// React 19 passes `ref` as a normal prop, and Base UI's Input forwards it — so
// react-hook-form's `register(...)` spread (incl. its ref) flows through to the DOM.
type PasswordInputProps = Readonly<React.ComponentProps<typeof Input>>;

/** Password field with a show/hide toggle. Shared by login + setup (anti-duplication). */
export function PasswordInput({ className, ...props }: PasswordInputProps) {
  const { t } = useTranslation();
  const [show, setShow] = useState(false);
  return (
    <div className="relative">
      <Input type={show ? "text" : "password"} className={cn("pr-9", className)} {...props} />
      <button
        type="button"
        onClick={() => setShow((s) => !s)}
        className="absolute inset-y-0 right-0 grid w-9 place-items-center rounded-r-lg text-muted-foreground outline-none hover:text-foreground focus-visible:text-foreground"
        aria-label={t(show ? "auth.hidePassword" : "auth.showPassword")}
      >
        {show ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
      </button>
    </div>
  );
}

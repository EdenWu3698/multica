"use client";

import { Keyboard, Languages, LogOut, Monitor, Moon, Palette, Sun, User } from "lucide-react";
import { useAuthStore } from "@multica/core/auth";
import { useWorkspacePaths } from "@multica/core/paths";
import { resolvePublicFileUrl } from "@multica/core/workspace/avatar-url";
import { ActorAvatar } from "@multica/ui/components/common/actor-avatar";
import { useTheme } from "@multica/ui/components/common/theme-provider";
import { TooltipHint } from "@multica/ui/components/common/tooltip-hint";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@multica/ui/components/ui/dropdown-menu";
import { cn } from "@multica/ui/lib/utils";
import { useLogout } from "../auth";
import { useT } from "../i18n";
import { AppLink } from "../navigation";
import { RAIL_ICON_BUTTON_CLASS } from "./rail-styles";

const THEMES = [
  { value: "light", icon: Sun },
  { value: "dark", icon: Moon },
  { value: "system", icon: Monitor },
] as const;

/** The signed-in user's menu at the foot of the rail. */
export function AccountMenu() {
  const { t } = useT("layout");
  const user = useAuthStore((s) => s.user);
  const p = useWorkspacePaths();
  const logout = useLogout();
  const { theme, setTheme } = useTheme();
  const name = user?.name ?? "";
  const avatar = (size: "md" | "lg") => (
    <ActorAvatar
      name={name}
      initials={(name || "U").charAt(0).toUpperCase()}
      avatarUrl={resolvePublicFileUrl(user?.avatar_url)}
      size={size}
    />
  );

  return (
    <DropdownMenu>
      <TooltipHint label={t(($) => $.account.trigger)} side="right">
        <DropdownMenuTrigger
          aria-label={t(($) => $.account.trigger)}
          className={cn(RAIL_ICON_BUTTON_CLASS, "rounded-full")}
        >
          {avatar("md")}
        </DropdownMenuTrigger>
      </TooltipHint>
      <DropdownMenuContent side="right" align="end" sideOffset={8} className="w-60">
        <div className="flex items-center gap-2.5 px-2 py-1.5">
          {avatar("lg")}
          <div className="min-w-0 flex-1">
            <p className="truncate text-body font-medium leading-tight">{user?.name}</p>
            <p className="truncate text-caption leading-tight text-muted-foreground">
              {user?.email}
            </p>
          </div>
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuItem render={<AppLink href={`${p.settings()}?tab=profile`} />}>
            <User />
            {t(($) => $.account.profile)}
          </DropdownMenuItem>
          <DropdownMenuSub>
            <DropdownMenuSubTrigger>
              <Palette />
              {t(($) => $.account.theme)}
            </DropdownMenuSubTrigger>
            <DropdownMenuSubContent>
              <DropdownMenuRadioGroup value={theme ?? "system"} onValueChange={setTheme}>
                {THEMES.map(({ value, icon: Icon }) => (
                  <DropdownMenuRadioItem key={value} value={value}>
                    <Icon />
                    {t(($) => $.account.themes[value])}
                  </DropdownMenuRadioItem>
                ))}
              </DropdownMenuRadioGroup>
            </DropdownMenuSubContent>
          </DropdownMenuSub>
          <DropdownMenuItem render={<AppLink href={`${p.settings()}?tab=preferences`} />}>
            <Languages />
            {t(($) => $.account.language)}
          </DropdownMenuItem>
          <DropdownMenuItem render={<AppLink href={`${p.settings()}?tab=shortcuts`} />}>
            <Keyboard />
            {t(($) => $.account.shortcuts)}
          </DropdownMenuItem>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuItem variant="destructive" onClick={logout}>
            <LogOut />
            {t(($) => $.sidebar.log_out)}
          </DropdownMenuItem>
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

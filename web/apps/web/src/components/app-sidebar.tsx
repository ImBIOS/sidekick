import { Link, useNavigate, useRouterState } from "@tanstack/react-router";
import {
  KeyRoundIcon,
  MonitorSmartphoneIcon,
  PackageIcon,
  RocketIcon,
} from "lucide-react";

import { authClient } from "@/lib/auth-client";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@sidekick/ui/components/sidebar";

import { SidekickMark } from "./brand";

const SECTIONS: {
  title: string;
  items: { to: string; label: string; exact: boolean; icon: typeof RocketIcon }[];
}[] = [
  {
    title: "Platform",
    items: [
      { to: "/console", label: "Get started", exact: true, icon: RocketIcon },
      { to: "/simulators", label: "Simulators", exact: false, icon: MonitorSmartphoneIcon },
      { to: "/builds", label: "Builds", exact: false, icon: PackageIcon },
    ],
  },
  {
    title: "Account",
    items: [{ to: "/settings", label: "API keys", exact: false, icon: KeyRoundIcon }],
  },
];

/** Console sidebar, built on the shadcn dashboard-01 Sidebar primitives. */
export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const navigate = useNavigate();
  const { data: session } = authClient.useSession();
  const initial = (session?.user.name ?? "?").slice(0, 1).toUpperCase();

  return (
    <Sidebar collapsible="offcanvas" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              size="lg"
              render={<Link to="/" />}
            >
              <SidekickMark size={24} />
              <span className="font-bold tracking-tight">Sidekick</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        {SECTIONS.map((section) => (
          <SidebarGroup key={section.title}>
            <SidebarGroupLabel>{section.title}</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {section.items.map(({ to, label, exact, icon: Icon }) => {
                  const active = exact ? pathname === to : pathname.startsWith(to);
                  return (
                    <SidebarMenuItem key={to}>
                      <SidebarMenuButton isActive={active} tooltip={label} render={<Link to={to} />}>
                        <Icon />
                        <span>{label}</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  );
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              size="lg"
              tooltip="Sign out"
              onClick={() =>
                authClient.signOut({ fetchOptions: { onSuccess: () => navigate({ to: "/" }) } })
              }
            >
              <span className="flex size-8 items-center justify-center rounded-lg bg-white/10 text-sm font-bold">
                {initial}
              </span>
              <span className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-medium">{session?.user.name ?? ""}</span>
                <span className="truncate text-xs text-muted-foreground">
                  {session?.user.email ?? ""} · Sign out
                </span>
              </span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  );
}

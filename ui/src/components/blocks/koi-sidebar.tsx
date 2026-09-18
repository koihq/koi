import { HugeiconsIcon } from "@hugeicons/react";
import type { ComponentProps, ReactElement, ReactNode } from "react";
import { KoiMark } from "@/components/blocks/koi-mark";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";

export type KoiSidebarItem = {
  id: string;
  label: string;
  icon: ComponentProps<typeof HugeiconsIcon>["icon"];
  active: boolean;
  render: ReactElement;
};

function Item({ item }: { item: KoiSidebarItem }) {
  return (
    <SidebarMenuItem>
      <SidebarMenuButton render={item.render} isActive={item.active} tooltip={item.label}>
        <HugeiconsIcon icon={item.icon} strokeWidth={2} />
        <span>{item.label}</span>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}

export function KoiSidebar({
  brand,
  home,
  workspace,
  workspaceLabel,
  action,
  actionLabel,
  navigationLabel,
  footer,
}: {
  brand: ReactNode;
  home: KoiSidebarItem;
  workspace: KoiSidebarItem[];
  workspaceLabel: string;
  action: KoiSidebarItem[];
  actionLabel: string;
  navigationLabel: string;
  footer: KoiSidebarItem;
}) {
  return (
    <Sidebar variant="inset" collapsible="icon">
      <SidebarHeader>
        <div className="flex h-10 min-w-0 items-center gap-2 px-2 group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-0">
          <KoiMark className="size-5 shrink-0" />
          <div className="min-w-0 truncate text-[13px] font-medium text-sidebar-foreground group-data-[collapsible=icon]:hidden">
            {brand}
          </div>
        </div>
      </SidebarHeader>
      <SidebarContent>
        <nav aria-label={navigationLabel}>
          <SidebarGroup>
            <SidebarMenu>
              <Item item={home} />
            </SidebarMenu>
          </SidebarGroup>
          <SidebarGroup className="mt-4">
            <SidebarGroupLabel>{workspaceLabel}</SidebarGroupLabel>
            <SidebarMenu>
              {workspace.map((item) => (
                <Item key={item.id} item={item} />
              ))}
            </SidebarMenu>
          </SidebarGroup>
          <SidebarGroup className="mt-4">
            <SidebarGroupLabel>{actionLabel}</SidebarGroupLabel>
            <SidebarMenu>
              {action.map((item) => (
                <Item key={item.id} item={item} />
              ))}
            </SidebarMenu>
          </SidebarGroup>
        </nav>
      </SidebarContent>
      <SidebarFooter>
        <SidebarMenu>
          <Item item={footer} />
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  );
}

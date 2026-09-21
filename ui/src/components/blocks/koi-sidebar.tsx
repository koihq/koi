import { HugeiconsIcon } from "@hugeicons/react";
import type { ComponentProps, ReactElement, ReactNode } from "react";
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
  header,
  workspace,
  workspaceLabel,
  action,
  actionLabel,
  navigationLabel,
  footer,
}: {
  header: ReactNode;
  workspace: KoiSidebarItem[];
  workspaceLabel: string;
  action: KoiSidebarItem[];
  actionLabel: string;
  navigationLabel: string;
  footer?: KoiSidebarItem;
}) {
  return (
    <Sidebar variant="inset" collapsible="icon">
      <SidebarHeader>{header}</SidebarHeader>
      <SidebarContent>
        <nav aria-label={navigationLabel}>
          <SidebarGroup>
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
      {footer ? (
        <SidebarFooter>
          <SidebarMenu>
            <Item item={footer} />
          </SidebarMenu>
        </SidebarFooter>
      ) : null}
    </Sidebar>
  );
}

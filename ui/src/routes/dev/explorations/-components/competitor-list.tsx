import type { Competitor } from "@/api/competitors.api";
import { BrandPill } from "@/components/blocks/brand-pill";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import C from "../-locales/mvp.en.json";
import { Icon } from "./common";
import { copy } from "./list-controls";

export function CompetitorList({
  items,
  onEdit,
  onConfirm,
  onDelete,
}: {
  items: Competitor[];
  onEdit: (competitor: Competitor) => void;
  onConfirm: (id: string) => void;
  onDelete: (competitor: Competitor) => void;
}) {
  return (
    <ul className="hc-management-list">
      {items.map((item) => (
        <li key={item.id} className="hc-management-row">
          <button type="button" className="hc-row-content" onClick={() => onEdit(item)}>
            <BrandPill name={item.name} domain={item.domain || undefined} />
            <small>{item.domain || item.githubRepo}</small>
            {!item.confirmed && <small>{item.source === "observed" ? C.observed : C.draft}</small>}
          </button>
          <div className="hc-actions">
            {!item.confirmed && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => (item.domain || item.githubRepo ? onConfirm(item.id) : onEdit(item))}
              >
                {item.domain || item.githubRepo ? C.confirm : C.identity}
              </Button>
            )}
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    aria-label={copy(C.actions, { name: item.name })}
                  />
                }
              >
                <Icon name="more" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="hc-brand-menu">
                <DropdownMenuItem onClick={() => onEdit(item)}>{C.edit}</DropdownMenuItem>
                <DropdownMenuItem onClick={() => onDelete(item)}>
                  {item.source === "observed" && !item.confirmed ? C.dismiss : C.delete}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </li>
      ))}
    </ul>
  );
}

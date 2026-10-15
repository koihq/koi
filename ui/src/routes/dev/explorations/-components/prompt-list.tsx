import type { Prompt } from "@/api/prompts.api";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import C from "../-locales/mvp.en.json";
import { Icon } from "./common";
import { HomeBrandText } from "./home-brand-text";
import { copy } from "./list-controls";

export function PromptList({
  items,
  onEdit,
  onConfirm,
  onDelete,
}: {
  items: Prompt[];
  onEdit: (prompt: Prompt) => void;
  onConfirm: (id: string) => void;
  onDelete: (prompt: Prompt) => void;
}) {
  return (
    <ul className="hc-management-list">
      {items.map((item) => (
        <li key={item.id} className="hc-management-row">
          <button type="button" className="hc-row-content" onClick={() => onEdit(item)}>
            <HomeBrandText text={item.text} />
            <small>{item.confirmed ? C.confirmed : C.draft}</small>
          </button>
          <div className="hc-actions">
            {!item.confirmed && (
              <Button variant="ghost" size="sm" onClick={() => onConfirm(item.id)}>
                {C.confirm}
              </Button>
            )}
            <DropdownMenu>
              <DropdownMenuTrigger
                render={
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    aria-label={copy(C.actions, { name: item.text })}
                  />
                }
              >
                <Icon name="more" />
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="hc-brand-menu">
                <DropdownMenuItem onClick={() => onEdit(item)}>{C.edit}</DropdownMenuItem>
                <DropdownMenuItem onClick={() => onDelete(item)}>{C.delete}</DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </li>
      ))}
    </ul>
  );
}

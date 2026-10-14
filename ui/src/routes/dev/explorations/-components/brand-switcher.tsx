import { type CSSProperties, useState } from "react";
import { BrandPill } from "@/components/blocks/brand-pill";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import C from "../-locales/home-context.en.json";

export function BrandSwitcher<T extends string>({
  brands,
  value,
  onChange,
  maxVisible = 3,
}: {
  brands: { id: T; name: string; domain?: string; src?: string }[];
  value: T;
  onChange: (brand: T) => void;
  maxVisible?: number;
}) {
  const [input, setInput] = useState("pointer");
  const [hovered, setHovered] = useState(false);
  const [keyboardFocus, setKeyboardFocus] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const expanded = hovered || keyboardFocus || menuOpen;
  const limit = Math.max(1, Math.floor(maxVisible));
  const visible = brands.slice(0, limit);
  const active = brands.find((brand) => brand.id === value);

  // Keep the active brand visible without reordering the other pinned slots.
  if (active && !visible.some((brand) => brand.id === value)) {
    visible.splice(visible.length - 1, 1, active);
  }

  const overflow = brands.filter((brand) => !visible.some((item) => item.id === brand.id));

  return (
    <fieldset
      className="hc-brand-switcher"
      aria-label={C.brand}
      data-input={input}
      onPointerEnter={(event) => {
        if (event.pointerType !== "touch") setHovered(true);
      }}
      onPointerLeave={() => setHovered(false)}
      onPointerDownCapture={() => setInput("pointer")}
      onPointerMove={() => {
        if (input !== "pointer") setInput("pointer");
      }}
      onKeyDownCapture={() => setInput("keyboard")}
      onFocusCapture={(event) => {
        if (event.target.matches(":focus-visible")) {
          setInput("keyboard");
          setKeyboardFocus(true);
        }
      }}
      onBlurCapture={(event) => {
        if (!event.currentTarget.contains(event.relatedTarget)) setKeyboardFocus(false);
      }}
    >
      <TooltipProvider delay={350}>
        {visible.map((brand, index) => (
          <Tooltip key={brand.id}>
            <TooltipTrigger
              render={<button type="button" className="hc-brand-avatar" />}
              aria-label={brand.name}
              aria-pressed={value === brand.id}
              style={
                {
                  "--brand-offset": `${expanded ? 0 : -12 * index}px`,
                  "--brand-layer": visible.length - index + 1,
                } as CSSProperties
              }
              onClick={() => onChange(brand.id)}
            >
              <span className="hc-brand-disc" aria-hidden="true">
                <BrandPill
                  name={brand.name}
                  domain={brand.domain}
                  src={brand.src}
                  className="hc-brand-logo"
                  iconOnly
                />
              </span>
            </TooltipTrigger>
            <TooltipContent
              side="bottom"
              sideOffset={8}
              className="hc-brand-tooltip"
              data-input={input}
            >
              <BrandPill name={brand.name} domain={brand.domain} src={brand.src} />
            </TooltipContent>
          </Tooltip>
        ))}
      </TooltipProvider>
      {overflow.length > 0 && (
        <DropdownMenu open={menuOpen} onOpenChange={setMenuOpen}>
          <DropdownMenuTrigger
            className="hc-brand-overflow"
            aria-label={C.moreBrands.replace("{count}", String(overflow.length))}
            style={
              {
                "--brand-offset": `${expanded ? 0 : -12 * visible.length}px`,
                "--brand-layer": 1,
              } as CSSProperties
            }
          >
            {`+${overflow.length}`}
          </DropdownMenuTrigger>
          <DropdownMenuContent
            align="start"
            sideOffset={8}
            className="hc-brand-menu hc-brand-overflow-menu"
            data-input={input}
          >
            <DropdownMenuRadioGroup
              value={value}
              onValueChange={(id) => {
                const brand = brands.find((item) => item.id === id);
                if (brand) onChange(brand.id);
              }}
            >
              {overflow.map((brand) => (
                <DropdownMenuRadioItem key={brand.id} value={brand.id}>
                  <BrandPill name={brand.name} domain={brand.domain} src={brand.src} />
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      )}
    </fieldset>
  );
}

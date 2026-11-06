import { Checkbox } from "@/components/ui/checkbox";

export function PromptSelectCheckbox({
  label,
  checked,
  indeterminate = false,
  onChange,
}: {
  label: string;
  checked: boolean;
  indeterminate?: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <span className="prompt-check-wrap relative inline-grid size-7 place-items-center align-middle">
      <Checkbox
        aria-label={label}
        checked={checked}
        indeterminate={indeterminate}
        onCheckedChange={onChange}
      />
    </span>
  );
}

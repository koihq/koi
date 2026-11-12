import { describe, expect, test } from "vitest";
import { page, userEvent } from "vitest/browser";
import { render } from "vitest-browser-react";

import { DataTable } from "@/components/blocks/data-table";

describe("DataTable", () => {
  test("renders server-paginated rows with headers, sort state and selection", async () => {
    await render(
      <DataTable
        ariaLabel="Brands"
        data={[
          { id: "one", name: "Koi", domain: "koi.dev" },
          { id: "two", name: "Acme", domain: "acme.com" },
        ]}
        columns={[
          { id: "name", header: "Brand", cell: (row) => row.name, sort: "ascending" },
          { id: "domain", header: "Domain", cell: (row) => row.domain },
        ]}
        getRowId={(row) => row.id}
        isRowSelected={(row) => row.id === "one"}
      />,
    );

    const table = page.getByRole("table", { name: "Brands" });

    await expect.element(table).toBeVisible();
    await expect
      .element(table.getByRole("columnheader", { name: "Brand" }))
      .toHaveAttribute("aria-sort", "ascending");
    await expect
      .element(table.getByRole("row", { name: "Koi koi.dev" }))
      .toHaveAttribute("data-selected", "true");
    await expect
      .element(table.getByRole("row", { name: "Acme acme.com" }))
      .not.toHaveAttribute("data-selected");
  });

  test("an overflowing table becomes a focusable region that scrolls with the keyboard", async () => {
    await page.viewport(320, 600);

    await render(
      <DataTable
        ariaLabel="Answers"
        data={[{ id: "one", text: "A long answer ".repeat(20) }]}
        columns={[
          { id: "text", header: "Answer", cell: (row) => row.text },
          { id: "engine", header: "Engine", cell: () => "ChatGPT" },
        ]}
        getRowId={(row) => row.id}
      />,
    );

    const region = page.getByRole("region", { name: "Answers" });
    await expect.element(region).toBeVisible();

    region.element().focus();
    await userEvent.keyboard("{ArrowRight}");

    await expect.poll(() => region.element().scrollLeft).toBeGreaterThan(0);
  });
});

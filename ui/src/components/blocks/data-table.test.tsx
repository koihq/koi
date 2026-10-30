import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { DataTable } from "@/components/blocks/data-table";

describe("DataTable", () => {
  it("renders server-paginated rows with their headers and selection state", () => {
    const markup = renderToStaticMarkup(
      <DataTable
        ariaLabel="Brands"
        data={[{ id: "one", name: "Koi" }]}
        columns={[{ id: "name", header: "Brand", cell: (row) => row.name }]}
        getRowId={(row) => row.id}
        isRowSelected={(row) => row.id === "one"}
      />,
    );

    expect(markup).toContain('aria-label="Brands"');
    expect(markup).toContain("<th");
    expect(markup).toContain("Brand</th>");
    expect(markup).toContain('data-selected="true"');
    expect(markup).toContain("Koi</td>");
  });
});

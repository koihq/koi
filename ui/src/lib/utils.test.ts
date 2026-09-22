import { describe, expect, it } from "vitest";

import { cn, fieldErrorsFromZod } from "@/lib/utils";

describe("cn", () => {
  it("joins class names", () => {
    expect(cn("foo", "bar")).toBe("foo bar");
  });

  it("merges conflicting Tailwind classes", () => {
    expect(cn("p-2", "p-4")).toBe("p-4");
  });
});

describe("fieldErrorsFromZod", () => {
  it("maps the first issue per field", () => {
    expect(
      fieldErrorsFromZod(
        [
          { path: ["githubRepo"] },
          { path: ["githubRepo"] },
          { path: ["name"] },
          { path: ["unknown"] },
        ],
        { name: "name", githubRepo: "github_repo" },
        (field) => field,
      ),
    ).toEqual({ github_repo: "github_repo", name: "name" });
  });
});

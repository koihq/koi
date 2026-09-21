import { z } from "zod";

import type { BrandRequest } from "@/api/generated";

function validHostname(host: string): boolean {
  if (host.length === 0 || host.length > 253) return false;

  const ipv4 = host.split(".");
  if (
    ipv4.length === 4 &&
    ipv4.every((part) => /^(0|[1-9]\d{0,2})$/.test(part) && Number(part) <= 255)
  ) {
    return false;
  }

  return host
    .split(".")
    .every(
      (label) =>
        label.length > 0 &&
        label.length <= 63 &&
        !label.startsWith("-") &&
        !label.endsWith("-") &&
        /^[a-z0-9-]+$/i.test(label),
    );
}

function validGitHubRepo(value: string): boolean {
  const match = /^https:\/\/github\.com(\/[^?#]*)$/i.exec(value);
  if (!match || match[1].includes("%")) return false;

  const parts = match[1].replace(/\/$/, "").split("/");
  if (parts.length !== 3) return false;

  const [, owner, repo] = parts;
  return Boolean(
    owner &&
      repo &&
      owner.length <= 39 &&
      !owner.startsWith("-") &&
      !owner.endsWith("-") &&
      /^[a-z0-9-]+$/i.test(owner) &&
      repo !== "." &&
      repo !== ".." &&
      /^[a-z0-9._-]+$/i.test(repo),
  );
}

export const brandFormSchema = z
  .object({
    name: z.string().trim().min(1),
    domain: z.string().trim(),
    githubRepo: z.string().trim(),
    country: z
      .string()
      .trim()
      .transform((value) => value.toUpperCase())
      .pipe(z.string().regex(/^[A-Z]{2}$/)),
    description: z.string().trim(),
  })
  .superRefine((value, context) => {
    if (!value.domain && !value.githubRepo) {
      context.addIssue({ code: "custom", path: ["domain"] });
      context.addIssue({ code: "custom", path: ["githubRepo"] });
    }
    if (value.domain && !validHostname(value.domain)) {
      context.addIssue({ code: "custom", path: ["domain"] });
    }
    if (value.githubRepo && !validGitHubRepo(value.githubRepo)) {
      context.addIssue({ code: "custom", path: ["githubRepo"] });
    }
  }) satisfies z.ZodType<BrandRequest>;

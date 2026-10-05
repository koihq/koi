import errors from "@/locales/en/errors.json";

export const errorCodes = [
  "validation",
  "email_taken",
  "brand_limit_reached",
  "invalid_credentials",
  "not_found",
  "unauthorized",
  "invalid_json",
  "internal",
  "feature_unavailable",
  "generation_conflict",
  "idempotency_conflict",
  "initial_already_completed",
  "run_not_ready",
] as const;

export type ErrorCode = (typeof errorCodes)[number];

export const errorCodeCopy: Record<ErrorCode, string> = errors.codes;

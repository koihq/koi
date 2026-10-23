import errors from "@/locales/en/errors.json";

export const errorCodes = [
  "validation",
  "resource_conflict",
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
  "prompt_limit_reached",
  "billing_required",
  "billing_conflict",
  "billing_checkout_pending",
  "billing_subscription_exists",
  "billing_price_mismatch",
] as const;

export type ErrorCode = (typeof errorCodes)[number];

export const errorCodeCopy: Record<ErrorCode, string> = errors.codes;

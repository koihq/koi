import errors from "@/locales/en/errors.json";

export const errorCodes = [
  "validation",
  "email_taken",
  "invalid_credentials",
  "not_found",
  "unauthorized",
  "invalid_json",
  "internal",
] as const;

export type ErrorCode = (typeof errorCodes)[number];

export const errorCodeCopy: Record<ErrorCode, string> = errors.codes;

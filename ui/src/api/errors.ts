import { type ErrorCode, errorCodes } from "@/api/errorCodes";
import type { Envelope, FieldError } from "@/api/generated";
import i18n from "@/config/i18n";

export class ApiError extends Error {
  readonly code: string;
  readonly fields?: FieldError[];
  readonly requestId?: string;

  constructor(code: string, fields?: FieldError[], requestId?: string) {
    super(code);
    this.name = "ApiError";
    this.code = code;
    this.fields = fields;
    this.requestId = requestId;
  }
}

function isErrorCode(code: string): code is ErrorCode {
  return (errorCodes as readonly string[]).includes(code);
}

export function displayMessage(code: string, fields?: FieldError[]): string {
  const field = fields?.[0];
  if (field?.field === "password" && field.code === "min_length") {
    return i18n.t("fields.password.min_length", { ns: "errors" });
  }

  if (field?.field === "email" && field.code === "invalid") {
    return i18n.t("fields.email.invalid", { ns: "errors" });
  }

  if (field?.field === "email" && field.code === "taken") {
    return i18n.t("fields.email.taken", { ns: "errors" });
  }

  if (isErrorCode(code)) {
    return i18n.t(`codes.${code}`, { ns: "errors" });
  }

  return i18n.t("codes.internal", { ns: "errors" });
}

export function messageFromUnknown(err: unknown): string {
  if (err instanceof ApiError) {
    return displayMessage(err.code, err.fields);
  }

  return displayMessage("internal");
}

export async function apiErrorFromResponse(response: Response): Promise<ApiError> {
  try {
    const body = (await response.json()) as Envelope;
    const code = body.error?.code ?? "internal";

    return new ApiError(code, body.error?.fields, body.error?.requestId);
  } catch {
    return new ApiError("internal");
  }
}

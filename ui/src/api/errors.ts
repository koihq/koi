import type { Envelope, FieldError, Middleware } from "@/api/generated";

export class ApiError extends Error {
  readonly code: string;
  readonly fields?: FieldError[];

  constructor(code: string, message: string, fields?: FieldError[]) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.fields = fields;
  }
}

export function displayMessage(code: string, fields?: FieldError[]): string {
  const field = fields?.[0];
  if (field?.field === "password" && field.code === "min_length") {
    return "Password must be at least 8 characters";
  }
  if (field?.field === "email" && field.code === "invalid") {
    return "Enter a valid email";
  }
  if (field?.field === "email" && field.code === "taken") {
    return "That email is already registered";
  }
  switch (code) {
    case "email_taken":
      return "That email is already registered";
    case "invalid_credentials":
      return "Invalid email or password";
    case "unauthorized":
      return "Sign in to continue";
    case "validation":
      return "Check the form and try again";
    default:
      return "Something went wrong";
  }
}

export function messageFromUnknown(err: unknown): string {
  if (err instanceof ApiError) {
    return err.message;
  }
  return displayMessage("internal");
}

async function fromResponse(response: Response): Promise<ApiError> {
  try {
    const body = (await response.json()) as Envelope;
    const code = body.error?.code ?? "internal";
    return new ApiError(code, displayMessage(code, body.error?.fields), body.error?.fields);
  } catch {
    return new ApiError("internal", displayMessage("internal"));
  }
}

export const envelopeMiddleware: Middleware = {
  async post({ response }) {
    if (response.ok) {
      return;
    }
    throw await fromResponse(response);
  },
};

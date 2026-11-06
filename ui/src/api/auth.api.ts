import type { AuthApi, AuthRes, CredentialsReq, MeRes, ResetPasswordReq } from "@/api/generated";

export type Credentials = CredentialsReq;

export type Account = MeRes;

export function createAuthApi(api: AuthApi) {
  return {
    login: (request: Credentials): Promise<AuthRes> => api.login({ credentialsReq: request }),
    logout: (): Promise<void> => api.logout(),
    forgotPassword: (email: string): Promise<void> =>
      api.forgotPassword({ forgotPasswordReq: { email } }),
    resetPassword: (request: ResetPasswordReq): Promise<void> =>
      api.resetPassword({ resetPasswordReq: request }),
    me: (init?: RequestInit): Promise<Account> => api.me(init),
    signup: (request: Credentials): Promise<AuthRes> => api.signup({ credentialsReq: request }),
  };
}

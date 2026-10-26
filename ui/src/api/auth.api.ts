import type { AuthApi, AuthRes, CredentialsReq, MeRes } from "@/api/generated";

export type Credentials = CredentialsReq;

export type Account = MeRes;

export function createAuthApi(api: AuthApi) {
  return {
    login: (request: Credentials): Promise<AuthRes> => api.login({ credentialsReq: request }),
    logout: (): Promise<void> => api.logout(),
    me: (init?: RequestInit): Promise<Account> => api.me(init),
    signup: (request: Credentials): Promise<AuthRes> => api.signup({ credentialsReq: request }),
  };
}

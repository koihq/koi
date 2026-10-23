import type { AuthApi, CredentialsReq, MeRes } from "@/api/generated";

export type Credentials = CredentialsReq;

export type Account = MeRes;

export function createAuthApi(api: AuthApi) {
  return {
    login: (request: Credentials): Promise<Account> => api.login({ credentialsReq: request }),
    logout: (): Promise<void> => api.logout(),
    me: (init?: RequestInit): Promise<Account> => api.me(init),
    signup: (request: Credentials): Promise<Account> => api.signup({ credentialsReq: request }),
  };
}

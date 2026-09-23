import type { AuthApi, CredentialsReq, MeRes } from "@/api/generated";
import { callApi } from "@/api/shared/apiClient";

export type Credentials = CredentialsReq;
export type Account = MeRes;

export function createAuthApi(api: AuthApi) {
  return {
    login: (request: Credentials): Promise<Account> => callApi(() => api.login(request)),
    logout: (): Promise<void> => callApi(() => api.logout()),
    me: (): Promise<Account> => callApi(() => api.me()),
    signup: (request: Credentials): Promise<Account> => callApi(() => api.signup(request)),
  };
}

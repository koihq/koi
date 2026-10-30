import type { CredentialsApi } from "@/api/generated";

export function createCredentialsApi(api: CredentialsApi) {
  return {
    list: (page = 1, pageSize = 20) => api.listCredentials({ page, pageSize }),
    revoke: (id: string) => api.revokeCredential({ id }),
  };
}

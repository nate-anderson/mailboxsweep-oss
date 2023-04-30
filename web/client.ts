import { NextRouter, useRouter } from "next/router";
import { Mailboxsweep, DefaultService, ApiError } from "./client-gen";

export const API = new Mailboxsweep({
  BASE: `https://${process.env.MAILBOXSWEEP_API_HOST}:${process.env.MAILBOXSWEEP_API_PORT}`,
  WITH_CREDENTIALS: true,
}).default;

API.commitChangeset

interface APIClient extends DefaultService {};

function wrap<RequestT, ResponseT>(router: NextRouter, func: (req: RequestT) => Promise<ResponseT>): (req: RequestT) => Promise<ResponseT> {
  const wrapped = async (req: RequestT): Promise<ResponseT> => {
    try {
      const result = await func(req);
      return result;
    } catch (error) {
      if (error instanceof ApiError) {
        const e = error as ApiError;
        if (e.status === 401) {
          router.push("/login?prompt=1");
        }
      }
      return null as ResponseT;
    }
  };
  return wrapped;
}

/**
 * Hook to get a wrapped API client that will redirect to login if an API request
 * fails with a 401
 * @returns APIClient
 */
export function useAPIClient(): APIClient {
  const router = useRouter();
  const patchedClient: Record<string, any> = {};
  Object.entries(API).forEach(([key, method]) => {
    if (typeof method === 'function') {
      patchedClient[key] = wrap(router, method);
    }
  })
  return API;
}

import { Configuration, DefaultApi, FetchParams, Middleware, RequestContext, ResponseContext } from "./client-gen";

const csrfTokenKey = "csrf_token";
const csrfHeader = "X-Csrf-Token";

const getCsrfToken = () => localStorage.getItem(csrfTokenKey) ?? undefined;
const setCsrfToken = (token: string) => localStorage.setItem(csrfTokenKey, token);

function maybeNavigate(dest: string): void {
  if (!window.location.href.endsWith(dest)) {
    window.location.href = dest;
  }
}

const csrfAuthRedirectMiddleware = (redirectOnAuthError: boolean): Middleware => ({
  pre: async (context: RequestContext): Promise<FetchParams | void> => {
    return {
      url: context.url,
      init: {
        ...context.init,
        headers: {
          ...context.init.headers,
          [csrfHeader]: getCsrfToken(),
        } as Record<string, string>
      }
    }
  },
  post: async (context: ResponseContext): Promise<Response | void> => {
    if (csrfHeader in context.response.headers) {
      const newToken = context.response.headers[csrfHeader];
      setCsrfToken(newToken as string);
    }

    if (context.response.status >= 400) {
      const errorBody = await context.response.json();

      // if error has a well-defined error code, that will be more useful than
      // the status code. try that first.
      if (redirectOnAuthError) {
        switch (errorBody.code) {
          case "google_reauthentication_required":
            maybeNavigate("/login/gmail/reauth/");
            return;
          case "email_login_required":
            if (errorBody.provider === "gmail") {
              maybeNavigate("/login/gmail/reauth/");
            } else {
              maybeNavigate("/email-account/login/");
            }
            return;
          case "terms_and_conditions_not_accepted":
            maybeNavigate("/agreement/TERMS_AND_CONDITIONS/");
            return;
          default:
            if (process.env.ENV === "dev") {
              // console.log("unknown or undefined body code in error: ", e.body);
            }
        }

        if (context.response.status === 401) {
          maybeNavigate("/login?prompt=1");
          return;
        }
      }
    }

    return context.response;
  }
})

export const API = new DefaultApi(new Configuration({
  basePath: `${process.env.MAILBOXSWEEP_API_PROTO ?? "https"}://${process.env.MAILBOXSWEEP_API_HOST}:${process.env.MAILBOXSWEEP_API_PORT}`,
  middleware: [csrfAuthRedirectMiddleware(true)],
  credentials: "include",
}));

export const APINoRedirect = new DefaultApi(new Configuration({
  basePath: `${process.env.MAILBOXSWEEP_API_PROTO ?? "https"}://${process.env.MAILBOXSWEEP_API_HOST}:${process.env.MAILBOXSWEEP_API_PORT}`,
  middleware: [csrfAuthRedirectMiddleware(false)],
  credentials: "include",
}));

/**
 * Hook to get a wrapped API client that will redirect to login if an API request
 * fails with a 401
 * @returns APIClient
 */
export function useAPIClient(): DefaultApi {
  return API;
}

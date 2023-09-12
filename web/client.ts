import { Mailboxsweep, DefaultService, ApiError, BaseHttpRequest, OpenAPIConfig, CancelablePromise } from "./client-gen";
import { ApiRequestOptions } from "./client-gen/core/ApiRequestOptions";
import { request as __request} from "./client-gen/core/request"

class HTTPRequestWithErrorHandling extends BaseHttpRequest {
  constructor(config: OpenAPIConfig) {
    super(config);
  }

  private maybeNavigate(dest: string): void {
    console.log({ href: window.location.href });
    if (!window.location.href.endsWith(dest)) {
      window.location.href = dest;
    }
  }

  public override request<T>(options: ApiRequestOptions): CancelablePromise<T> {
    return new CancelablePromise(async (resolve, reject, _) => {
      try {
        const result = await __request<T>(this.config, options);
        resolve(result);
      } catch(error) {
        if (error instanceof ApiError) {
          const e = error as ApiError;
          console.log({ apiErr: e });
          // if error has a well-defined error code, that will be more useful than
          // the status code. try that first.
          switch (e.body.code) {
            case "google_reauthentication_required":
              this.maybeNavigate("/login/gmail/reauth/");
              return null as T;
            case "email_login_required":
              if (e.body.provider === "gmail") {
                this.maybeNavigate("/login/gmail/reauth/");
              } else {
                this.maybeNavigate("/email-account/login/");
              }
              return null as T;
            case "terms_and_conditions_not_accepted":
              this.maybeNavigate("/agreement/TERMS_AND_CONDITIONS/");
              return null as T;
            default:
              if (process.env.ENV === "dev") {
                console.log("unknown or undefined body code in error: ", e.body);
              }
          }

          if (e.status === 401) {
            this.maybeNavigate("/login?prompt=1");
            return null as T;
          }
        }

        reject(error);
      }
    });
  }
}

export const API = new Mailboxsweep({
  BASE: `${process.env.MAILBOXSWEEP_API_PROTO ?? "https"}://${process.env.MAILBOXSWEEP_API_HOST}:${process.env.MAILBOXSWEEP_API_PORT}`,
  WITH_CREDENTIALS: true,
}, HTTPRequestWithErrorHandling).default;

// Does not redirect on auth errors
export const APINoRedirect = new Mailboxsweep({
  BASE: `${process.env.MAILBOXSWEEP_API_PROTO ?? "https"}://${process.env.MAILBOXSWEEP_API_HOST}:${process.env.MAILBOXSWEEP_API_PORT}`,
  WITH_CREDENTIALS: true,
}).default;

/**
 * Hook to get a wrapped API client that will redirect to login if an API request
 * fails with a 401
 * @returns APIClient
 */
export function useAPIClient(): DefaultService {
  return API;
}

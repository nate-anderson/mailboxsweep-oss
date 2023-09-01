import { Mailboxsweep, DefaultService, ApiError, BaseHttpRequest, OpenAPIConfig, CancelablePromise } from "./client-gen";
import { ApiRequestOptions } from "./client-gen/core/ApiRequestOptions";
import { request as __request} from "./client-gen/core/request"

class HTTPRequestWithErrorHandling extends BaseHttpRequest {
  constructor(config: OpenAPIConfig) {
    super(config);
  }

  public override request<T>(options: ApiRequestOptions): CancelablePromise<T> {
    return new CancelablePromise(async (resolve, reject, _) => {
      try {
        const result = await __request<T>(this.config, options);
        resolve(result);
      } catch(error) {
        if (error instanceof ApiError) {
          const e = error as ApiError;

          // if error has a well-defined error code, that will be more useful than
          // the status code. try that first.
          switch (e.body.code) {
            case "google_reauthentication_required":
            case "email_login_required":
              window.location.href = "/login/gmail/reauth";
              return null as T;
            case "terms_and_conditions_not_accepted":
              window.location.href = "/agreement/TERMS_AND_CONDITIONS";
              return null as T;
            default:
              console.log("unknown or undefined body code in error: ", e.body);
          }

          if (e.status === 401) {
            window.location.href = "/login?prompt=1";
            return null as T;
          }
        }

        reject(error);
      }
    });
  }
}

export const API = new Mailboxsweep({
  BASE: `https://${process.env.MAILBOXSWEEP_API_HOST}:${process.env.MAILBOXSWEEP_API_PORT}`,
  WITH_CREDENTIALS: true,
}, HTTPRequestWithErrorHandling).default;


/**
 * Hook to get a wrapped API client that will redirect to login if an API request
 * fails with a 401
 * @returns APIClient
 */
export function useAPIClient(): DefaultService {
  return API;
}

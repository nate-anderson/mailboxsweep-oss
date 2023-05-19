import { Mailboxsweep, DefaultService, ApiError, BaseHttpRequest, OpenAPIConfig, CancelablePromise } from "./client-gen";
import { ApiRequestOptions } from "./client-gen/core/ApiRequestOptions";
import { request as __request} from "./client-gen/core/request"

class HTTPRequestWithErrorHandling extends BaseHttpRequest {
  constructor(config: OpenAPIConfig) {
    super(config);
  }

  public override async request<T>(options: ApiRequestOptions): CancelablePromise<T> {
    try {
      const result = await __request<T>(this.config, options);
      return result;
    } catch(error) {
      if (error instanceof ApiError) {
        const e = error as ApiError;
        console.log({ debugError: e });
        // if error has a well-defined error code, that will be more useful than
        // the status code. try that first.
        const body = JSON.parse(e.body);
        console.log({debugBody: body});
        switch (body.code) {
          case "google_reauthentication_required":
            window.location.href = "/login/gmail/reauth";
          default:
            console.log("unknown body code ", body.code);
            
        }

        if (e.status === 401) {
          window.location.href = "/login?prompt=1";
        }
      }

      throw error;
    }
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

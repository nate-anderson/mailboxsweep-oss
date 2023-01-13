import { NextResponse } from "next/server";
import { NextRequest } from "next/server";

const isLoggedInCookie = "X-MailboxSweep-LoggedIn";

const PUBLIC_FILE = /\.(.*)$/;

const loginRoutes = [
    "/login",
    "/login-token"
];

export default function middleware(request: NextRequest) {
    const { pathname } = request.nextUrl;
    
    // https://github.com/vercel/next.js/discussions/38587
    if (
        pathname.startsWith("/_next") || // exclude Next.js internals
        pathname.startsWith("/api") || //  exclude all API routes
        pathname.startsWith("/static") || // exclude static files
        PUBLIC_FILE.test(pathname) // exclude all files in the public folder
      ) {
        return NextResponse.next();
      }
    

    if (!loginRoutes.includes(pathname)) {
        const cookie = request.cookies.get(isLoggedInCookie)
        if (cookie?.value !== "true") {
            console.log("redirecting");
            return NextResponse.redirect(new URL("/login?prompt=1", request.url));
        }
    }
    return NextResponse.next();
    
}

import "../styles/globals.css";
import type { AppProps } from "next/app";
import Head from "next/head";
import { AdapterDateFns } from "@mui/x-date-pickers/AdapterDateFns";
import { LocalizationProvider } from "@mui/x-date-pickers";
import { GoogleTagManager } from '@next/third-parties/google';

export default function App({ Component, pageProps }: AppProps) {
  return (
    <>
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <Head>
          <meta name="viewport" content="initial-scale=1, width=device-width" />
          <title>MailboxSweep</title>
        </Head>
        <Component {...pageProps} />
        <GoogleTagManager gtmId={process.env.GOOGLE_TAG_ID ?? ""} />
      </LocalizationProvider>
    </>
  );
}

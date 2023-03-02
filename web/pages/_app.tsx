import "../styles/globals.css";
import type { AppProps } from "next/app";
import Head from "next/head";
import { AdapterDateFns } from "@mui/x-date-pickers/AdapterDateFns";
import { LocalizationProvider } from "@mui/x-date-pickers";

export default function App({ Component, pageProps }: AppProps) {
  return (
    <>
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <Head>
          <meta name="viewport" content="initial-scale=1, width=device-width" />
          <title>Mailboxsweep</title>
        </Head>
        <Component {...pageProps} />
      </LocalizationProvider>
    </>
  );
}

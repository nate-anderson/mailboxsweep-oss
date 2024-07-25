import "../styles/globals.css";
import type { AppProps } from "next/app";
import Head from "next/head";
import { AdapterDateFns } from "@mui/x-date-pickers/AdapterDateFns";
import { LocalizationProvider } from "@mui/x-date-pickers";
import { GoogleTagManager } from '@next/third-parties/google';
import { GTagManagerID } from "../gtag";
import { useEffect } from "react";
import Script from "next/script";

export default function App({ Component, pageProps }: AppProps) {

  return (
    <>
      <LocalizationProvider dateAdapter={AdapterDateFns}>
        <Head>
          <meta name="viewport" content="initial-scale=1, width=device-width" />
          <title>MailboxSweep</title>
        </Head>
        <Component {...pageProps} />
        <Script id="sprig-loader">
            {`
            (function(l,e,a,p) {
              if (window.Sprig) return;
              window.Sprig = function(){S._queue.push(arguments)}
              var S = window.Sprig;S.appId = a;S._queue = [];window.UserLeap=S;
              a=l.createElement('script');
              a.async=1;a.src=e+'?id='+S.appId;
              p=l.getElementsByTagName('script')[0];
              p.parentNode.insertBefore(a, p);
              console.log("Bingo, it ran!");
            })(document, 'https://cdn.sprig.com/shim.js', '${process.env.NEXT_PUBLIC_SPRIG_ENV_ID}');
            `}
          </Script>
        <GoogleTagManager gtmId={GTagManagerID[process.env.ENV ?? "dev"] ?? ""} />
      </LocalizationProvider>
    </>
  );
}

import { useRouter } from "next/router";
import { useEffect, useState } from "react";
import Container from "@mui/material/Container";
import CircularProgress from "@mui/material/CircularProgress";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";

const isLoggedInCookie = "X-MailboxSweep-LoggedIn";

// https://stackoverflow.com/questions/5968196/how-do-i-check-if-a-cookie-exists
function getCookie(name: string): string | undefined {
  const value = `; ${document.cookie}`;
  const parts = value.split(`; ${name}=`);
  if (parts.length === 2) return parts.pop()?.split(';').shift();
}

// hook for checking if the logged-in cookie is set and redirecting to login if not
export default function PrivatePage(props: { children: JSX.Element }) {
  const router = useRouter();
  const [ canRender, setCanRender ] = useState(false);
  useEffect(() => {
    const cookie = getCookie(isLoggedInCookie);
    console.log({ cookie });
    if (cookie === 'true') {
      setCanRender(true);
      return;
    }

    router.push("/login?prompt=1");
  });

  return canRender
    ? <>{props.children}</>
    : <Container component="main">
    <Box
      sx={{
        marginTop: "25vh",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
      }}
    >
      <Typography
        sx={{
          marginY: 3,
          fontFamily: "Lobster",
          fontSize: "3em",
          color: "var(--color-primary)",
        }}
      >
        MailboxSweep
      </Typography>
      <Typography
        sx={{
          marginY: 2,
        }}
      >
        Loading...
      </Typography>
      <CircularProgress />
    </Box>
  </Container>;
}
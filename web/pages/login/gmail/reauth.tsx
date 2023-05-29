import { useRouter } from "next/router";
import { useEffect } from "react";
import { useAPIClient } from "../../../client";
import Container from "@mui/material/Container";
import CircularProgress from "@mui/material/CircularProgress";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";

export default function Reauth() {
  const router = useRouter();

  const API = useAPIClient();

  useEffect(() => {
    if (!router.isReady) return;
    const query = router.query;

    API.reauthenticateGmail()
      .then((res) => {
        if (res.redirect && res.redirectUrl) {
          window.location.href = res.redirectUrl;
        } else {
          router.push("/home");
        }
      })
      .catch(console.error)
  }, [router.isReady, router.query, router, API]);

  return (
    <Container component="main">
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
          Signing back into your Google account
        </Typography>
        <CircularProgress />
      </Box>
    </Container>
  );
}

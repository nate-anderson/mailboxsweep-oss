import { useRouter } from "next/router";
import { useEffect } from "react";
import { useAPIClient } from "../../../client";
import Container from "@mui/material/Container";
import CircularProgress from "@mui/material/CircularProgress";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { getProviderByAddress, setProvider } from "../../../lib/steps";
import { getInitialStepPath } from "../../../components/signup-stepper";

export default function Login() {
  const router = useRouter();

  const API = useAPIClient();

  useEffect(() => {
    if (!router.isReady) return;
    const query = router.query;

    const stateToken = query["state"] as string;
    const authCode = query["code"] as string;

    API.submitGmailStateToken(stateToken, authCode)
      .then((res) => {
        const provider = getProviderByAddress(res.address);
        setProvider(provider);
        Promise.all([
          API.getPaymentStatus(),
          API.getUserAgreement("TERMS_AND_CONDITIONS"),
        ])
        .then(([paymentStatus, agreement]) => {
          const stepPath = getInitialStepPath(Boolean(agreement.agreedAt), Boolean(paymentStatus.subscribedAt), res.profileComplete);
          if (stepPath) {
            router.push(stepPath);
          } else {
            router.push("/home");
          }
        })
        .catch(() => {
          router.push("/home");
        });
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
          Signing you in...
        </Typography>
        <CircularProgress />
      </Box>
    </Container>
  );
}

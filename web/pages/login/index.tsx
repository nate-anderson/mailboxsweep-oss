import { API } from "../../client";
import Box from "@mui/material/Box";
import Container from "@mui/material/Container";
import TextField from "@mui/material/TextField";
import { useEffect, useState } from "react";
import { useRouter } from "next/router";
import CssBaseline from "@mui/material/CssBaseline";
import Typography from "@mui/material/Typography";
import Dialog from "@mui/material/Dialog";
import DialogTitle from "@mui/material/DialogTitle";
import DialogContent from "@mui/material/DialogContent";
import { useConstantSnackbar, useSnackbar } from "../../components/snackbar";
import ImageLogo from "../../components/image-logo";
import Alert from "@mui/material/Alert";
import LoadingButton from "@mui/lab/LoadingButton";

function StagingWarning(props: { environment: string | undefined }) {
  if (props.environment === "prod") {
    return <></>;
  }

  return <Alert
    severity="warning"
    sx={{
      marginBottom: 4
    }}
  >
    This is the MailboxSweep <b>{props.environment}</b> test site, only available to approved test users. Please visit <a href="https://mailboxsweep.com">MailboxSweep.com</a> to sign up for our mailing list.
  </Alert>
}

export default function Login() {
  const router = useRouter();
  const [email, changeEmail] = useState("");
  const [modalOpen, setModalOpen] = useState(false);
  const [redirectPromptSnackbar, showRedirectSnackbar] = useConstantSnackbar(
    "Please log in to continue.",
    "success"
  );
  const [isLoading, setIsLoading] = useState(false);
  const [errorSnackbar, setErrorMessage] = useSnackbar("error", 5000);

  const submitLogin = async (
    address: string,
    setMagicLinkModalOpen: (open: boolean) => void
  ) => {
    try {
      setIsLoading(true);
      const res = await API.initiateLogin({
        address,
      });
      setIsLoading(false);
  
      if (res.ok && !res.redirect) {
        setMagicLinkModalOpen(true);
      }
  
      if (res.ok && res.redirect) {
        if (!res.redirectUrl) {
          console.error("Redirect was true, but no URL was provided");
          return;
        }
        window.location.href = res.redirectUrl;
      }
    } catch (error) {
      console.log({ initiateLoginError: error!.toString() });
      setErrorMessage("Sorry, something went wrong. Please try again in a moment.");
    }
  };

  useEffect(() => {
    if (!router.isReady) return;
    const query = router.query;

    if (Boolean(query["prompt"])) {
      showRedirectSnackbar(true);
    }
  }, [router.isReady, router.query, router, showRedirectSnackbar]);

  return (
    <Container component="main">
      {redirectPromptSnackbar}
      {errorSnackbar}
      <Dialog
        open={modalOpen}
        aria-labelledby="login-email-modal-title"
        aria-describedby="login-email-modal-text"
      >
        <DialogTitle>Check your email!</DialogTitle>
        <DialogContent>
          We've sent you a link to log into Mailboxsweep. This is for your
          convenience and security. Please check your inbox to continue!
        </DialogContent>
      </Dialog>
      <CssBaseline />
      <form onSubmit={(e) => {
        e.preventDefault();
        submitLogin(email, () => setModalOpen(true))
      }}>
      <Box
        sx={{
          marginTop: "20vh",
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
        }}
      >
        <ImageLogo />
        <Typography
          align="center"
          sx={{
            marginY: 2,
          }}
        >
          Log in or sign up
        </Typography>
        <StagingWarning environment={process.env.ENV}/>
          <TextField
            label="Email"
            name="email"
            type="email"
            value={email}
            onChange={(e) => changeEmail(e.target.value)}
            sx={{
              marginX: 20,
              maxWidth: 300,
              width: "100%",
            }}
          ></TextField>
          <LoadingButton
            variant="contained"
            size="large"
            sx={{
              marginY: 3,
              backgroundColor: "var(--color-accent)",
              color: "white",
            }}
            type="submit"
            loading={isLoading}
          >
            Sign in
          </LoadingButton>
        </Box>
      </form>
    </Container>
  );
}

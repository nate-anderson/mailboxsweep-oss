import { API } from "../../client";
import Navbar from "../../components/navbar";
import Container from "@mui/material/Container";
import Box from "@mui/material/Box";
import Button from "../../components/button";
import CssBaseline from "@mui/material/CssBaseline";
import Typography from "@mui/material/Typography";
import Avatar from "@mui/material/Avatar";
import TextField from "@mui/material/TextField";
import Grid from "@mui/material/Grid";
import IconEnvelope from "@mui/icons-material/Email";
import { useEffect, useState } from "react";
import { ApiError } from "../../client-gen";
import { parseISO, format } from "date-fns";
import { displayFormat } from "../../dates";
import Dialog from "@mui/material/Dialog";
import DialogTitle from "@mui/material/DialogTitle";
import DialogContent from "@mui/material/DialogContent";
import DialogActions from "@mui/material/DialogActions";
import MUIButton from "@mui/material/Button";
import IconHelp from "@mui/icons-material/HelpOutline";
import Link from "next/link";
import { useConstantSnackbar, useSnackbar } from "../../components/snackbar";

const renderPasswordOnly = (
  password: string,
  setPassword: (value: string) => void
) => {
  return (
    <Grid item xs={12} sm={12}>
      <TextField
        label="Password"
        type="password"
        autoComplete="none"
        required
        fullWidth
        value={password}
        onChange={(e) => {
          setPassword(e.target.value);
        }}
      ></TextField>
    </Grid>
  );
};

const renderPasswordAndIMAP = (
  password: string,
  setPassword: (value: string) => void,
  host: string,
  setHost: (value: string) => void,
  port: number | undefined,
  setPort: (port: number) => void
) => {
  return (
    <>
      <Grid item xs={12} sm={12}>
        <TextField
          label="Password"
          type="password"
          autoComplete="none"
          required
          fullWidth
          value={password}
          onChange={(e) => {
            setPassword(e.target.value);
          }}
        ></TextField>
      </Grid>
      <Grid item xs={12} sm={12}>
        <TextField
          autoFocus
          label="IMAP Host"
          type="text"
          autoComplete="none"
          required
          fullWidth
          value={host}
          onChange={(e) => {
            setHost(e.target.value);
          }}
        ></TextField>
      </Grid>
      <Grid item xs={12} sm={12}>
        <TextField
          label="IMAP Port"
          type="text"
          autoComplete="none"
          required
          fullWidth
          value={port}
          onChange={(e) => {
            setPort(Number(e.target.value));
          }}
        ></TextField>
      </Grid>
    </>
  );
};

export default function Login() {
  const [address, changeAddress] = useState("");
  const [password, changePassword] = useState("");
  const [submitButtonEnabled, changeSubmitButtonEnabled] = useState(false);
  const [created, changeCreated] = useState<Date>();
  const [expires, changeExpires] = useState<Date>();
  const [credentialSet, changeCredentialSet] = useState(false);
  const [errorSnackbar, changeError] = useSnackbar("error");
  const [successSnackbar, showSuccess] = useConstantSnackbar(
    "Inbox credential updated!",
    "success"
  );

  const [showUnknownServerDialog, setShowUnknownServerDialog] = useState(false);
  const [showImapServerForm, setShowImapServerForm] = useState(false);

  const [imapHost, setImapHost] = useState("");
  const [imapPort, setImapPort] = useState<number>(993);

  const resetFilledInput = () => changePassword("********");

  useEffect(() => {
    API.getInboxCredential()
      .then((res) => {
        changeAddress(res.address);
        // 2022-12-29T23:15:17-08:00
        const createdDate = parseISO(res.created);
        if (isNaN(createdDate.valueOf())) {
          return changeError("Invalid created date.");
        }
        changeCreated(createdDate);

        const expiresDate = parseISO(res.expires);
        if (isNaN(expiresDate.valueOf())) {
          return changeError("Invalid expires date.");
        }
        changeExpires(expiresDate);
        changeCredentialSet(true);
        resetFilledInput();
      })
      .catch((error) => {
        if (error instanceof ApiError) {
          const e = error as ApiError;
          if (e.status == 404) {
            API.getUserData()
              .then((res) => {
                changeAddress(res.address);
              })
              .catch((err) => {
                console.error({ getUserDataError: err });
              });
            return;
          }
        }
        console.log({ getInboxCredentialError: error });
      });
      // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const submitCredential = () => {
    changeSubmitButtonEnabled(false);
    if (showImapServerForm) {
      API.storeImapServerAndCredentials({
        password,
        imapHost,
        imapPort,
      })
        .then((res) => {
          showSuccess(true);
        })
        .catch((err) => {
          if (err instanceof ApiError) {
            const e = err as ApiError;
            changeError(e.message);
          }
        });
      return;
    }
    API.storeInboxCredential({
      password,
    })
      .then(() => {
        showSuccess(true);
        resetFilledInput();
      })
      .catch((err) => {
        if (err instanceof ApiError) {
          const e = err as ApiError;
          switch (e.status) {
            case 502:
              changeError(
                "We couldn't log into your email provider with this password. Please try again."
              );
              break;
            case 404:
              setShowUnknownServerDialog(true);
              setShowImapServerForm(true);
              break;
            default:
              changeError("Something went wrong: " + e.body);
          }
        }
      });
  };

  const updatePassword = (password: string) => {
    changePassword(password);
    changeSubmitButtonEnabled(password !== "");
  };

  const updateHost = (host: string) => {
    changeSubmitButtonEnabled(host !== "");
    setImapHost(host);
  };

  const updatePort = (port: number) => {
    changeSubmitButtonEnabled(port !== 0);
    setImapPort(port);
  };

  return (
    <>
      <Navbar page="Email Account" />
      {successSnackbar}
      {errorSnackbar}
      <Container component="main">
        <Dialog
          open={showUnknownServerDialog}
          aria-labelledby="login-email-modal-title"
          aria-describedby="login-email-modal-text"
          onClose={() => setShowUnknownServerDialog(false)}
        >
          <DialogTitle>We need a bit more info.</DialogTitle>
          <DialogContent>
            We don't recognize your email provider - but that's okay! As long as
            your email provider supports IMAP (and most do!) we can continue. If
            you need help locating your email provider's IMAP settings,{" "}
            <a>click here. @TODO</a>
          </DialogContent>
          <DialogActions>
            <MUIButton onClick={() => setShowUnknownServerDialog(false)}>
              Continue
            </MUIButton>
          </DialogActions>
        </Dialog>
        <CssBaseline />
        <Box
          sx={{
            marginTop: 6,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
          }}
        >
          <Avatar sx={{ m: 1, bgcolor: "var(--color-accent)" }}>
            <IconEnvelope />
          </Avatar>
          <Typography
            align="center"
            sx={{
              color: "var(--color-black)",
              fontSize: "2em",
              fontWeight: 100,
            }}
          >
            Inbox Credentials
          </Typography>
          <Typography
            align="center"
            sx={{
              marginY: 2,
              color: "var(--color-gray)",
              maxWidth: 600,
            }}
          >
            Your credentials are stored with enterprise-grade AES-256 encryption
            and deleted after 30 days.
          </Typography>
        </Box>

        {credentialSet ? (
          <Box
            sx={{
              maxWidth: 600,
              width: "auto",
              marginX: "auto",
            }}
          >
            <Typography
              sx={{
                color: "var(--color-accent)",
                marginY: 1,
                fontSize: 14,
              }}
              align="left"
            >
              Last updated: {format(created!, displayFormat)}
            </Typography>
            <Typography
              sx={{
                color: "var(--color-accent)",
                marginY: 1,
                fontSize: 14,
              }}
              align="left"
            >
              Will be deleted at {format(expires!, displayFormat)}
            </Typography>
          </Box>
        ) : (
          <></>
        )}

        <Box
          component="form"
          sx={{
            marginY: 3,
            maxWidth: 600,
            width: "auto",
            marginX: "auto",
          }}
        >
          <Grid container spacing={3}>
            <Grid item xs={12} sm={12}>
              {/* Read-only */}
              <TextField
                label="Email"
                fullWidth
                disabled
                value={address}
              ></TextField>
            </Grid>
            {showImapServerForm
              ? renderPasswordAndIMAP(
                  password,
                  updatePassword,
                  imapHost,
                  updateHost,
                  imapPort,
                  updatePort
                )
              : renderPasswordOnly(password, updatePassword)}

            <Grid
              item
              xs={12}
              sm={12}
              sx={{
                display: "flex",
                justifyContent: showImapServerForm ? "space-between" : "right",
              }}
            >
              {showImapServerForm ? (
                <Link href="#">
                  <MUIButton endIcon={<IconHelp />}>Help</MUIButton>
                </Link>
              ) : (
                <></>
              )}

              <Button
                disabled={!submitButtonEnabled}
                onClick={submitCredential}
              >
                Save
              </Button>
            </Grid>
          </Grid>
        </Box>
      </Container>
    </>
  );
}

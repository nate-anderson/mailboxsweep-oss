import CssBaseline from "@mui/material/CssBaseline";
import Container from "@mui/material/Container";
import Navbar from "../components/navbar";
import Alert from "@mui/material/Alert";
import AlertTitle from "@mui/material/AlertTitle";
import Box from "@mui/material/Box";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import Typography from "@mui/material/Typography";
import { ApiError, ClassificationStatus, Plan, UpdateOperation } from "../client-gen";
import { useAPIClient } from "../client";
import { useEffect, useState } from "react";
import { useSnackbar } from "../components/snackbar";
import PrivatePage from "../auth";
import { format, parseISO } from "date-fns";
import ActionLink from "../components/action-link";
import BackupPrompt1 from "../components/prompt/backup-1";
import BackupPrompt2 from "../components/prompt/backup-2";
import LinearProgress from "@mui/material/LinearProgress";

const renderClassificationStatus = (
  status: ClassificationStatus.status | undefined,
  operation: UpdateOperation | undefined,
) => {
  const boxify = (children: React.ReactNode) => (
    <Box
      sx={{
        marginTop: 3,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
      }}
    >
      {children}
    </Box>
  );

  switch (status) {
    case ClassificationStatus.status.IN_PROGRESS:
      return boxify(
        <Alert
          severity="info"
          variant="filled"
          sx={{
            maxWidth: 600,
            backgroundColor: "var(--color-accent)",
          }}
        >
          <AlertTitle>Inbox Scan In Progress</AlertTitle>
          We're categorizing your inbox messages now. This requires going
          through a lot of emails and can take some time. We'll send you a
          message when we're ready to start cleaning up!
        </Alert>
      );

    case ClassificationStatus.status.COMPLETE:
      return boxify(
        <Alert
          severity="success"
          variant="filled"
          sx={{
            maxWidth: 600,
            width: "100%",
            "& > .MuiAlert-message": {
              width: "100%",
            },
          }}
        >
          <AlertTitle>Inbox Scan Complete!</AlertTitle>
          <p style={{ display: "block" }}>
            Your inbox scan is complete! Let's start cleaning up.
          </p>
          <ActionLink
            href="/changeset/select"
            color="#2e7d32"
            backgroundColor="#EDF7ED"
            text="Let's go"
          />
        </Alert>
      );

    case ClassificationStatus.status.ERRORED:
      return boxify(
        <Alert
          severity="error"
          sx={{
            maxWidth: 600,
          }}
        >
          <AlertTitle>Something went wrong</AlertTitle>
          <p style={{ display: "block" }}>
            Something went wrong while classifying your inbox messages. You can
            try again, or contact support for help.
          </p>
          <ActionLink
            href="/classify"
            text="Try again"
            backgroundColor="white"
            color="var(--color-danger)"
          />
        </Alert>
      );
      
    case ClassificationStatus.status.EXECUTED:
      return boxify(
        <Alert
          severity="info"
          sx={{
            maxWidth: 600,
          }}
        >
          <AlertTitle>Cleanup complete! &#127881;</AlertTitle>
          <p style={{ display: "block" }}>
            We scanned and cleaned up your inbox! If you want to make more changes, you can scan your inbox again.
          </p>
          <ActionLink
            href="/classify"
            text="Clean up more"
            backgroundColor="var(--color-accent)"
            color="white"
          />
        </Alert>
      );

    case ClassificationStatus.status.EXECUTING:
      return boxify(
        <Alert
          severity="info"
          sx={{
            maxWidth: 600,
          }}
        >
          <AlertTitle>
            Your messages are being {operation === UpdateOperation.DELETE ? "deleted" : "marked read"}.
          </AlertTitle>
          <p style={{ display: "block" }}>
            This can take a few minutes, especially if you've selected a lot of messages. We'll let you know when cleanup is complete!
          </p>
          <LinearProgress />
        </Alert>
      );
    case undefined:
      // display nothing, the stepper should still be visible at this point
      return <></>;
  }
};

const renderActionPrompt = (status: ClassificationStatus.status | undefined) => {
  switch (status) {
    case ClassificationStatus.status.IN_PROGRESS:
      return <BackupPrompt1 />;
    case ClassificationStatus.status.COMPLETE:
      return <BackupPrompt2 />;
  }
}

export default function Home() {
  const [address, setAddress] = useState("");
  const [errorSnackbar, setErrorMessage] = useSnackbar("error");
  const [allMessagesCount, setAllMessagesCount] = useState<number>(0);
  const [unreadMessagesCount, setUnreadMessagesCount] = useState<number>(0);
  const [classificationStatus, setClassificationStatus] = useState<
    ClassificationStatus.status | undefined
  >();
  const [subscribedPlan, setSubscribedPlan] = useState<Plan | undefined>();
  const [planExpires, setPlanExpires] = useState<string | undefined>();
  const [operation, setOperation] = useState<UpdateOperation | undefined>();

  const API = useAPIClient();

  useEffect(() => {
    API.getInboxMetadata()
      .then((res) => {
        setAllMessagesCount(res.messageCount);
        setUnreadMessagesCount(res.unreadCount);
      })
      .catch((err) => {
        if (err instanceof ApiError) {
          const e = err as ApiError;
          switch (e.status) {
            case 404:
              return;
          }
        }
        console.log(err.toString());
        setErrorMessage("Something went wrong");
      });

    API.getClassificationStatus()
      .then((res) => {
        setAddress(res.address);
        if (res.status) {
          setClassificationStatus(res.status.status);
        }
        if (res.status?.operation) {
          setOperation(res.status.operation);
        }
      })
      .catch((err) => {
        if (err instanceof ApiError) {
          const e = err as ApiError;
          setErrorMessage(e.message);
        } else {
          console.log(err);
        }
      });

    API.getPaymentStatus()
      .then((res) => {
        if (res.plan) {
          setSubscribedPlan(res.plan);
        }
        if (res.validUntil) {
          // 2023-06-28T00:43:25-07:00
          const expires = parseISO(res.validUntil);
          setPlanExpires(format(expires, `MM/dd/yyyy`));
        }
      })
      .catch((err) => {
        if (err instanceof ApiError) {
          const e = err as ApiError;
          setErrorMessage(e.message);
        } else {
          console.log(err);
        }
      })
  }, []);

  return (
    <PrivatePage>
      <>
        <Navbar page="Home" />
        <Container component="main">
          {errorSnackbar}
          <CssBaseline />

          {renderClassificationStatus(classificationStatus, operation)}
          {renderActionPrompt(classificationStatus)}

          <Card
            sx={{
              maxWidth: 600,
              width: "auto",
              marginTop: 3,
              marginX: "auto",
            }}
          >
            <CardContent>
              <Typography
                sx={{
                  color: "var(--color-gray)",
                }}
              >
                Your inbox
              </Typography>
              <Typography
                sx={{
                  fontSize: "1.4em",
                }}
              >
                {address}
              </Typography>
              <Typography
                display={allMessagesCount > 0 ? "block" : "none"}
                sx={{
                  marginTop: 1,
                  color: "var(--color-gray)",
                }}
              >
                {allMessagesCount.toLocaleString()} Messages
              </Typography>
              <Typography
                display={unreadMessagesCount > 0 ? "block" : "none"}
                sx={{
                  color: "var(--color-accent)",
                }}
              >
                {unreadMessagesCount.toLocaleString()} Unread
              </Typography>
            </CardContent>

            {/* If scan is complete, link to scan inside the inbox
                      card with a big "Continue ->" button or similar */}
          </Card>

          <Card
            sx={{
              marginTop: 3,
              maxWidth: 600,
              marginX: "auto",
            }}
          >
            {subscribedPlan
              ? <CardContent>
                  <Typography
                    sx={{
                      color: "var(--color-gray)",
                    }}
                  >
                    Your plan
                  </Typography>
                  <Typography
                    sx={{
                      fontSize: "1.6em",
                      color: "var(--color-primary)",
                    }}
                  >
                    {subscribedPlan?.displayName}
                  </Typography>
                  <Typography
                    sx={{
                      marginTop: 1,
                      color: "var(--color-gray)",
                    }}
                  >
                    Expires {planExpires}
                  </Typography>
                  <Typography
                    sx={{
                      color: "var(--color-accent)",
                    }}
                  >
                    120k messages deleted
                  </Typography>
                </CardContent>
              : <></>
            }

            {/* If scan is complete, link to scan inside the inbox
                      card with a big "Continue ->" button or similar */}
          </Card>
        </Container>
      </>
    </PrivatePage>
  );
}

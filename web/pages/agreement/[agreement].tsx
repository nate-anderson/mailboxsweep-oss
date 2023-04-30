import { CssBaseline } from "@mui/material";
import { useRouter } from "next/router";
import { useEffect, useState } from "react";
import { API } from "../../client";
import { UserAgreement } from "../../client-gen";
import Navbar from "../../components/navbar";
import Container from "@mui/material/Container";
import Box from "@mui/material/Box";
import Avatar from "@mui/material/Avatar";
import Typography from "@mui/material/Typography";
import IconGrading from "@mui/icons-material/Grading";
import Paper from "@mui/material/Paper";
import Button from "../../components/button";
import { parseISO, format, formatISO } from "date-fns";
import { displayFormat } from "../../dates";
import { useConstantSnackbar, useSnackbar } from "../../components/snackbar";
import PrivatePage from "../../auth";

const renderAgreedAt = (agr?: UserAgreement) => {
  if (!agr?.agreedAt) {
    console.log("error: not agreed");
    return;
  }

  const agreedAt = parseISO(agr!.agreedAt!);
  const displayDate = format(agreedAt, displayFormat);
  return (
    <Box>
      <Typography
        sx={{
          marginY: 2,
        }}
      >
        You consented to this agreement on {displayDate}
      </Typography>
    </Box>
  );
};

const renderAgreementForm = (
  submit: (agreementCode: string) => Promise<void>,
  agr?: UserAgreement
) => {
  return (
    <Box
      component="form"
      sx={{
        marginY: 2,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
      }}
    >
      <Typography>I agree to the terms above</Typography>
      <Button
        sx={{
          marginY: 2,
        }}
        onClick={() => {
          if (agr?.key) {
            submit(agr.key);
          }
        }}
      >
        Agree
      </Button>
    </Box>
  );
};

export default function Agreement() {
  const router = useRouter();

  const [agreement, setUserAgreement] = useState<UserAgreement | undefined>();

  useEffect(() => {
    if (!router.isReady) return;

    const query = router.query;
    const { agreement: agreementCode } = router.query;

    API.getUserAgreements(agreementCode as string)
      .then((res) => {
        setUserAgreement(res);
      })
      .catch((error) => {
        console.log({ getUserAgreementsError: error });
      });
  }, [router.isReady, router.query]);

  const [successSnackbar, setShowSuccess] = useConstantSnackbar(
    "Agreement submitted!",
    "success"
  );
  const [errorSnackbar, setError] = useSnackbar("error");

  const submitAgreement = async (key: string): Promise<void> => {
    try {
      const res = await API.setConsentedAgreements({
        agreements: [key],
      });
      const agreedDate = formatISO(new Date());
      setShowSuccess(true);
      setUserAgreement((agr) => {
        if (!agr) {
          console.log("failed to set user agreement: existing undefined");
          return undefined;
        }
        return {
          ...agr,
          agreedAt: agreedDate,
        };
      });
    } catch (err) {
      console.log(err);
      setError("Failure submitting agreement. Please try again later.");
    }
  };

  return (
    <PrivatePage>
      <>
        <Navbar />
        {successSnackbar}
        {errorSnackbar}
        <Container component="main">
          <CssBaseline />
          <Box
            sx={{
              marginTop: 6,
              marginX: 1,
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
            }}
          >
            <Avatar sx={{ m: 1, bgcolor: "var(--color-accent)" }}>
              <IconGrading />
            </Avatar>
            <Typography
              align="center"
              sx={{
                color: "var(--color-black)",
                fontSize: "2em",
                fontWeight: 100,
              }}
            >
              {agreement?.name}
            </Typography>
            <Typography
              align="center"
              sx={{
                color: "var(--color-black)",
                marginY: 2,
              }}
            >
              {agreement?.description}
            </Typography>
            <Typography
              align="center"
              sx={{
                color: "var(--color-accent)",
                marginY: 1,
              }}
            >
              Updated {agreement?.agreementUpdatedAt}
            </Typography>
            <Paper variant="outlined">
              <Typography
                sx={{
                  margin: 1,
                  padding: 1,
                  fontFamily: "monospace",
                  fontSize: "12px",
                  lineHeight: "1.75em",
                }}
              >
                {agreement?.body}
              </Typography>
            </Paper>
            {agreement?.agreedAt
              ? renderAgreedAt(agreement)
              : renderAgreementForm(submitAgreement, agreement)}
          </Box>
        </Container>
      </>
    </PrivatePage>
  );
}

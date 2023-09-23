import { CssBaseline } from "@mui/material";
import { useRouter } from "next/router";
import { useEffect, useState } from "react";
import { API } from "../../client";
import { UserAgreement } from "../../client-gen";
import Navbar from "../../components/navbar";
import Container from "@mui/material/Container";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import IconGrading from "@mui/icons-material/Grading";
import Paper from "@mui/material/Paper";
import Button from "../../components/button";
import { parseISO, format, formatISO } from "date-fns";
import { displayFormat } from "../../dates";
import { useConstantSnackbar, useSnackbar } from "../../components/snackbar";
import PrivatePage from "../../auth";
import ReactMarkdown from 'react-markdown'
import PageHeader from "../../components/page-header";
import { nextStep } from "../../lib/steps";
import remarkGfm from "remark-gfm";

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
          textAlign: "center",
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
        marginTop: 2,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
      }}
    >
      <Typography>I agree to the terms above</Typography>
      <Button
        data-testid="button-agree-submit"
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
    let slug = "";
    if (typeof agreementCode === "string") {
      slug = agreementCode.replaceAll("-", "_").toUpperCase();
    } else {
      console.log("agreement code is not string");
    }

    API.getUserAgreement(slug)
      .then((res) => {
        setUserAgreement(res);
      })
      .catch((error) => {
        console.log({ getUserAgreementError: error });
      });
  }, [router.isReady]);

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
      const nextPath = nextStep();
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
      if (nextPath) {
        router.push(nextPath);
      }
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
          <PageHeader
            title={agreement?.name ?? ""}
            showStepper={true}
            icon={<IconGrading />}
            stepperSlug="TERMS"
          />
          <Box
            sx={{
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
            }}
          >
            <Paper
              variant="outlined"
              sx={{
                margin: 1,
                paddingX: 3,
                paddingY: 1,
                fontFamily: "monospace",
                fontSize: "12px",
                lineHeight: "1.75em",
                overflow: "auto",
                maxHeight: "400px",
              }}>
              <ReactMarkdown remarkPlugins={[remarkGfm]}>
                {agreement?.body ?? ""}
              </ReactMarkdown>
            </Paper>
            {
              agreement?.agreedAt
                ? renderAgreedAt(agreement)
                : renderAgreementForm(submitAgreement, agreement)
            }
            <Typography
              align="center"
              sx={{
                color: "var(--color-gray)",
                marginTop: 1,
                marginBottom: 3,
              }}
            >
              Terms Updated {agreement?.agreementUpdatedAt}
            </Typography>
          </Box>
        </Container>
      </>
    </PrivatePage>
  );
}

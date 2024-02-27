import { API } from "../../client";
import { ResponseError, Plan } from "../../client-gen";
import Navbar from "../../components/navbar";
import Container from "@mui/material/Container";
import Box from "@mui/material/Box";
import CssBaseline from "@mui/material/CssBaseline";
import Typography from "@mui/material/Typography";
import IconCardMembership from "@mui/icons-material/CardMembership";
import { useEffect, useState } from "react";
import PlanCard from "../../components/plan";
import PrivatePage from "../../auth";
import PageHeader from "../../components/page-header";
import { useSnackbar } from "../../components/snackbar";

export default function SelectPlan() {
  const [plans, setPlans] = useState<Plan[] | undefined>();
  const [showSuccess, setShowSuccess] = useState(false);
  const [errorSnackbar, setErrorMessage] = useSnackbar("error", 5000);

  useEffect(() => {
    API.listPlans()
      .then((res) => {
        setPlans(res.plans.sort((a, b) => a.displayIndex - b.displayIndex));
      })
      .catch((err) => {
        setErrorMessage("Failed to retrieve plans, please try again momentarily.");
      });
  }, []);

  const submit = async (plan: string) => {
    API.initiatePayment({
      initiatePaymentRequest: {
        plan,
      }
    })
    .then((res) => {
      setErrorMessage("");
      setShowSuccess(true);
      setTimeout(() => document.location.href = res.checkoutUrl!, 250);
    })
    .catch((error) => {
      if (error instanceof ResponseError) {
        const e = error as ResponseError;
        if (e.response.status === 422) {
          setErrorMessage("Your account already has an active plan.");
        } else {
          setErrorMessage(error.message);
        }
        return;
      }
      setErrorMessage(JSON.stringify(error));
    });
  };

  return (
    <PrivatePage>
      <>
        <Navbar />
        <Container
          component="main"
          sx={{
            marginBottom: 7,
          }}
        >
          <CssBaseline />
          {errorSnackbar}
          <PageHeader
            showStepper={true}
            title="Choose a Plan"
            icon={<IconCardMembership />}
            stepperSlug="PLAN"
          />
          <Box
            sx={{
              marginX: 1,
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
            }}
          >
            <Typography
              align="center"
              sx={{
                marginY: 2,
                color: "var(--color-gray)",
              }}
            >
              Choose the best option to get your inbox back to clean.
            </Typography>
          </Box>
          {plans
            ?.sort((p) => p.priceInCents)
            .map((p) => <PlanCard key={p.key} plan={p} submit={() => submit(p.key)} />) 
            }
        </Container>
      </>
    </PrivatePage>
  );
}

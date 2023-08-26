import { API } from "../../client";
import { Plan } from "../../client-gen";
import Navbar from "../../components/navbar";
import Container from "@mui/material/Container";
import Box from "@mui/material/Box";
import CssBaseline from "@mui/material/CssBaseline";
import Typography from "@mui/material/Typography";
import Avatar from "@mui/material/Avatar";
import IconCardMembership from "@mui/icons-material/CardMembership";
import { useEffect, useState } from "react";
import PlanCard from "../../components/plan";
import PrivatePage from "../../auth";
import PageHeader from "../../components/page-header";

export default function SelectPlan() {
  const [plans, setPlans] = useState<Plan[] | undefined>();
  const [showSuccess, setShowSuccess] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | undefined>();

  useEffect(() => {
    API.listPlans()
      .then((res) => {
        setPlans(res.plans);
      })
      .catch((err) => {
        console.log(err);
      });
  }, []);

  const submit = async (plan: string) => {
    API.initiatePayment({
      plan,
    })
    .then((res) => {
      setErrorMessage(undefined);
      setShowSuccess(true);
      window.location.href = res.checkoutUrl!;
    })
    .catch((error) => {
      setErrorMessage(error as string);
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

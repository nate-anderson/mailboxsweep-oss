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
    try {
      const res = await API.initiatePayment({
        plan,
      });
      setErrorMessage(undefined);
      setShowSuccess(true);
      window.open(res.checkoutUrl!);
    } catch (error) {
      setErrorMessage(error as string);
    }
  };

  return (
    <>
      <Navbar />
      <Container
        component="main"
        sx={{
          marginBottom: 7,
        }}
      >
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
            <IconCardMembership />
          </Avatar>
          <Typography
            align="center"
            sx={{
              color: "var(--color-black)",
              fontSize: "2em",
              fontWeight: 100,
            }}
          >
            Choose a Plan
          </Typography>
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
  );
}

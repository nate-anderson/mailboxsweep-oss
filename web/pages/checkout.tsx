import { API } from "../client";
import { Plan } from "../client-gen";
import { useRouter } from "next/router";
import { useEffect, useState } from "react";
import { useSnackbar } from "../components/snackbar";
import { loadStripe, Stripe, StripeCustomCheckoutOptions } from '@stripe/stripe-js';
import PrivatePage from "../auth";
import Navbar from "../components/navbar";
import CssBaseline from "@mui/material/CssBaseline";
import Container from "@mui/material/Container";
import { PaymentElement, CustomCheckoutProvider, useCustomCheckout } from '@stripe/react-stripe-js';
import Card from "@mui/material/Card";
import Button from "@mui/material/Button";
import Box from "@mui/material/Box";
import IconShoppingCart from "@mui/icons-material/ShoppingCart";
import PlanCard, { Price, PriceWithDiscount } from "../components/plan";
import { Typography } from "@mui/material";
import { LoadingSpinner } from "../components/loading-spinner";
import Grid from "@mui/material/Grid";
import Stack from "@mui/material/Stack";
import Radio from "@mui/material/Radio";
import FormLabel from '@mui/material/FormLabel';
import StripeBadge from "../public/stripe_blurple.svg";
import Image from "next/image";
import LoadingButton from "@mui/lab/LoadingButton";

export default function Agreement() {
    const router = useRouter();

    const [plan, setPlan] = useState<string | undefined>();
    const [showSuccess, setShowSuccess] = useState(false);
    const [plansLoading, setPlansLoading] = useState(true);
    const [errorSnackbar, setError] = useSnackbar("error");
    const [stripePromise, setStripePromise] = useState<Promise<Stripe | null>>();
    const [stripeOptions, setStripeOptions] = useState<StripeCustomCheckoutOptions | undefined>();
    const [stripeSessionId, setStripeSessionId] = useState<string | undefined>();
    const [allPlans, setAllPlans] = useState<Plan[]>([]);

    useEffect(() => {
        // on page load, get client secret to initiate Stripe
        const key: string | undefined = process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY;
        
        if (!key) {
            throw new Error("stripe publishable key not set in ENV (STRIPE_PUBLISHABLE_KEY)");
        }
        const sp = loadStripe(key, { betas: ['custom_checkout_beta_2']});
        setStripePromise(sp);

        API.listPlans().then((res) => {
            setAllPlans(res.plans);
            setPlansLoading(false);
        }).catch((error) => {
            setError("Failed to fetch available MailboxSweep plans, please try again.");
        });

    }, [router.isReady, plan]);

    const submitPlanSelection = (planKey: string) => {
        setPlan(planKey);
        API.initiatePayment({
            initiatePaymentRequest: {
                plan: planKey
            }
        }).then((res) => {
            if (res.clientSecret) {
                setStripeOptions({ clientSecret: res.clientSecret });
            } else {
                setError("Unable to create checkout session.");
            }

            if (res.sessionId) {
                setStripeSessionId(res.sessionId);
            } else {
                setError("Could not find Stripe session.");
            }
        }).catch((error) => {
            setError(JSON.stringify(error));
        })
    }

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
                    {
                        
                        <Typography
                            align="center"
                            sx={{
                                color: "var(--color-black)",
                                fontSize: "2em",
                                fontWeight: 100,
                                marginY: 3,
                                paddingX: 3,
                            }}
                        >
                            {Boolean(plan) ? "Checkout" : "Choose a Plan"}
                        </Typography>
                            
                    }
                    <Grid
                        container
                        item
                        direction={{
                            xs: "column",
                            sm: "column",
                            md: "row",
                            lg: "row"
                        }}
                        paddingX={{
                            md: 5,
                            sm: 0,
                        }}
                        justifyContent={"center"}
                        alignItems={"center"}
                        gap={4}
                    >
                    {
                        !Boolean(plan) 
                            ? plansLoading ? <LoadingSpinner /> : allPlans
                                .sort((p) => p.priceInCents)
                                .map((p) => <Box key={p.key}><PlanCard plan={p} submit={() => submitPlanSelection(p.key)} confirmText="Choose Plan"/></Box>) 
                            : <></>
                            
                    }
                    </Grid>
                    {
                        Boolean(plan)
                            ? <SelectedPlan plan={allPlans.find((p) => p.key === plan)!}></SelectedPlan>
                            : <></>
                    }
                    {
                        stripePromise && stripeOptions && stripeSessionId && Boolean(plan)
                        ?   <CheckoutForm
                                setError={setError}
                                stripePromise={stripePromise}
                                stripeOptions={stripeOptions}
                                stripeSessionId={stripeSessionId}
                            />
                        : <></>
                    }
                    
                </Container>
            </>
        </PrivatePage>
    );
}

function SelectedPlan(props: { plan: Plan }) {
    return <Card
        sx={{
            maxWidth: 600,
            width: "auto",
            marginTop: 1,
            marginX: "auto",
            padding: 2,
        }}
    >
      <Stack direction="row">
        <Radio checked={true} disableFocusRipple={true} disableRipple={true} disableTouchRipple={true} size="small"/>
        <FormLabel
            id="demo-controlled-radio-buttons-group"
            sx={{
                display: "flex",
                flexDirection: "row",
                alignItems: "center"
            }}
        >
            <Typography
                data-testid={`plan-name-${props.plan.key.toLowerCase()}`}
                variant="h6"
                sx={{
                    color: "var(--color-accent)",
                }}
            >
            {props.plan.displayName}
            </Typography>
        </FormLabel>
      </Stack>
      <Box marginLeft={2}>
        <PriceWithDiscount plan={props.plan} marginTop={0}/>
        <Typography
            sx={{
            color: "var(--color-gray)",
            marginY: 0,
            }}
        >
            Pay <b>once</b> and use for {props.plan.durationDays} days
        </Typography>
      </Box>
    </Card>
}

interface CheckoutProps {
    setError: (str: string) => void;
    stripePromise: Promise<Stripe | null>;
    stripeOptions: StripeCustomCheckoutOptions;
    stripeSessionId: string;
};

function CheckoutForm(props: CheckoutProps) {
    return <CustomCheckoutProvider stripe={props.stripePromise} options={props.stripeOptions}>
        <Checkout setError={props.setError} stripeSessionId={props.stripeSessionId} />
    </CustomCheckoutProvider>
}

function Checkout(props: { setError: (str: string) => void; stripeSessionId: string }) {
    const checkout = useCustomCheckout();
    return (
        <Box
            display="flex"
            alignItems="center"
            flexDirection="column"
            marginY={3}
        >
            <Image src={StripeBadge} alt="Secure checkout powered by Stripe" />
            <Card
                sx={{
                    maxWidth: "600px",
                    marginTop: 3,
                    marginX: "auto",
                    padding: 3,
                    width: "100%",
                }}
            >
                <PaymentElement />
                <Box
                    display="flex"
                    flexDirection="column"
                    alignContent="flex-end"
                >
                    <PayButton setError={props.setError} stripeSessionId={props.stripeSessionId} />
                </Box>
                
            </Card>
        </Box>
    );
}

function PayButton (props: { setError: (str: string) => void, stripeSessionId: string }) {
    const {confirm, canConfirm, confirmationRequirements} = useCustomCheckout();
    const [loading, setLoading] = useState(false);
    const router = useRouter();
    const successUrl = `http://localhost:3000/plan/payment-success?checkoutSessionId=${props.stripeSessionId}`;
  
    const handleClick = () => {
      setLoading(true);
      confirm({
        return_url: successUrl,
      }).then((result) => {
        setLoading(false);
        if (result.error) {
          // Confirmation failed. Display the error message.
          props.setError(result.error.message ?? "Something went wrong, please refresh and try again.");
        } else {
            router.push(successUrl);
        }
        
      }).catch((error) => {
        props.setError(error.message ?? "Something went wrong, please refresh and try again.")
      });
    };
  
    return (
      <LoadingButton
        loading={loading}
        disabled={!canConfirm || loading}
        onClick={handleClick}
        variant="contained"
        sx={{
            marginY: 2,
            marginX: "auto"
        }}
        endIcon={<IconShoppingCart />}
      >
        Securely Pay
      </LoadingButton>
    )
  };
  
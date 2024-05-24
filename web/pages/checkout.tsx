import { API } from "../client";
import { Plan, ResponseError } from "../client-gen";
import { useRouter } from "next/router";
import { useEffect, useState } from "react";
import { useSnackbar } from "../components/snackbar";
import {
    loadStripe,
    Stripe,
    StripeCustomCheckoutOptions,
    StripeCustomCheckoutResult,
    StripeCustomCheckoutConfirmationRequirement,
} from '@stripe/stripe-js';
import PrivatePage from "../auth";
import Navbar from "../components/navbar";
import CssBaseline from "@mui/material/CssBaseline";
import Container from "@mui/material/Container";
import { PaymentElement, CustomCheckoutProvider, useCustomCheckout } from '@stripe/react-stripe-js';
import Card from "@mui/material/Card";
import Checkbox from "@mui/material/Checkbox";
import Box from "@mui/material/Box";
import IconShoppingCart from "@mui/icons-material/ShoppingCart";
import PlanCard, { PriceWithDiscount } from "../components/plan";
import { Typography } from "@mui/material";
import { LoadingSpinner } from "../components/loading-spinner";
import Grid from "@mui/material/Grid";
import Stack from "@mui/material/Stack";
import Radio from "@mui/material/Radio";
import FormLabel from '@mui/material/FormLabel';
import StripeBadge from "../public/stripe_blurple.svg";
import Image from "next/image";
import LoadingButton from "@mui/lab/LoadingButton";
import FormGroup from "@mui/material/FormGroup";
import FormControlLabel from "@mui/material/FormControlLabel";
import Link from "next/link";
import PageHeader from "../components/page-header";
import IconCardMembership from "@mui/icons-material/CardMembership";
import Divider from '@mui/material/Divider';
import { nextStep, getStepIndex } from "../lib/steps";

export default function Agreement() {
    const router = useRouter();

    const [plan, setPlan] = useState<string | undefined>();
    const [plansLoading, setPlansLoading] = useState(true);
    const [errorSnackbar, setError] = useSnackbar("error");
    const [stripePromise, setStripePromise] = useState<Promise<Stripe | null>>();
    const [stripeOptions, setStripeOptions] = useState<StripeCustomCheckoutOptions | undefined>();
    const [stripeSessionId, setStripeSessionId] = useState<string | undefined>();
    const [allPlans, setAllPlans] = useState<Plan[]>([]);
    const [paymentLoading, setPaymentLoading] = useState(false);

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

        API.getPaymentStatus().then((res) => {
            if (res.validUntil) {
                if (typeof getStepIndex() !== 'undefined') {
                    const nextPath = nextStep();
                    if (nextPath) {
                        router.push(nextPath);
                        return;
                    }
                    router.push("/plan");
                }
            }
    
        }).catch((err => console.log(err)))

    }, [router.isReady, plan]);

    const submitPlanSelection = (planKey: string) => {
        setPlan(planKey);
        setPaymentLoading(true);
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
            const resErr = error as ResponseError;
            if (resErr.response.status == 422) {
                setError("You have already paid for a plan.");
                setTimeout(() => {
                    router.push("/home");
                }, 5000);
            }
        }).finally(() => {
            setPaymentLoading(false);
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
                    <PageHeader
                        showStepper={true}
                        title="Checkout"
                        icon={<IconCardMembership />}
                        stepperSlug="CHECKOUT"
                    />
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
                        marginTop={3}
                    >
                    {
                        !Boolean(plan) 
                            ? plansLoading ? <CheckoutLoadingSpinner /> : allPlans
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
                        <Grid
                            container
                            display="flex"
                            justifyContent="center"
                            alignItems="center"
                            marginTop={3}
                        >
                            <Image src={StripeBadge} alt="Secure checkout powered by Stripe" />
                        </Grid>
                    {
                        paymentLoading ? <LoadingSpinner /> : <></>
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

function CheckoutLoadingSpinner() {
    return <Box
        display="flex"
        alignItems="center"
        flexDirection="column"
        marginY={3}
    >
    </Box>
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
    const { confirm, canConfirm, confirmationRequirements } = useCustomCheckout();
    const [agreedToTerms, setAgreedToTerms] = useState(false);
    const handleSubmit = async (args?: { return_url?: string | undefined }): Promise<StripeCustomCheckoutResult> => {
        const termsKey = "TERMS_AND_CONDITIONS";
        try {
            const res = await API.setConsentedAgreements({setUserAgreementsRequest: {
                agreements: [termsKey],
            }});
            return confirm();
        } catch (err) {
            props.setError(JSON.stringify(err))
            throw err;
        }
    }
    return (
        <Box
            display="flex"
            alignItems="center"
            flexDirection="column"
            marginBottom={3}
        >
            <Card
                sx={{
                    maxWidth: "600px",
                    marginTop: 3,
                    marginX: "auto",
                    padding: 3,
                    width: "100%",
                }}
            >
                <PaymentElement
                    options={{
                        fields: {
                            billingDetails: {
                                name: 'auto',
                            }
                        }
                    }}
                />
                
                <Box
                    display="flex"
                    flexDirection="column"
                    alignContent="center"
                >  
                    { canConfirm ? <Divider /> : <></>}
                    <FormGroup>
                        <FormControlLabel
                            label={<Typography>I agree to the MailboxSweep <Link href="/agreement/terms-and-conditions/">Terms and Conditions</Link></Typography>}
                            sx={{ color: "var(--color-black)", paddingTop: 2 }}
                            control={<Checkbox
                                sx={{borderColor: "red"}}
                                checked={agreedToTerms}
                                onChange={(e) => setAgreedToTerms(e.target.checked)}
                                data-testid="checkbox-agree-terms"
                            />}
                        />
                    </FormGroup>
                    <PayButton
                        setError={props.setError}
                        stripeSessionId={props.stripeSessionId}
                        canConfirm={canConfirm && agreedToTerms}
                        confirm={handleSubmit}
                        confirmationRequirements={confirmationRequirements}
                    />
                </Box>
                
            </Card>
        </Box>
    );
}

interface PayButtonProps {
    confirm: (args?: { return_url?: string | undefined } | undefined) => Promise<StripeCustomCheckoutResult>;
    canConfirm: boolean;
    confirmationRequirements: StripeCustomCheckoutConfirmationRequirement[];
    setError: (str: string) => void;
    stripeSessionId: string
}

function PayButton (props: PayButtonProps) {
    const [loading, setLoading] = useState(false);
    const router = useRouter();
    const successUrl = `http://localhost:3000/plan/payment-success?checkoutSessionId=${props.stripeSessionId}`;
  
    const handleClick = () => {
      setLoading(true);
      props.confirm({
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
        data-testid="button-pay"
        loading={loading}
        disabled={!props.canConfirm || loading}
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
  
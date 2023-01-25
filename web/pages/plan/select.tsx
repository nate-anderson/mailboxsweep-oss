import { API } from "../../client";
import { Plan } from "../../client-gen";
import Navbar from "../../components/navbar";
import Container from '@mui/material/Container';
import Box from "@mui/material/Box";
import CssBaseline from "@mui/material/CssBaseline";
import Typography from "@mui/material/Typography";
import Avatar from "@mui/material/Avatar";
import IconCardMembership from '@mui/icons-material/CardMembership';
import Card from "@mui/material/Card";
import { useEffect, useState } from "react";
import Button from "@mui/material/Button";
import IconArrowForward from "@mui/icons-material/ArrowForward";
import currency from "currency.js";

const renderPrice = (price: currency) => {
    const cents = String(price.cents()).padStart(2, '0');
    const commonStyles = {
        marginBottom: 0,
    };
    return <div>
        <Typography
            display="inline"
            sx={{
                ...commonStyles,
                marginRight: 0,
                display: "inline-block",
                fontSize: "1.5em",
                marginY: 1,
                fontWeight: "light",
            }}
        >
        $
        </Typography>
        <Typography
            display="inline"
            sx={{
                ...commonStyles,
                fontSize: "2.5em",
                marginY: 1,
                fontWeight: "light",
            }}
        >
            {price.dollars()}
        </Typography>
        <Typography
            display="inline"
            sx={{
                ...commonStyles,
                fontSize: "2em",
                marginY: 1,
                marginLeft: -0.4,
                fontWeight: "light",
            }}
        >
            .
        </Typography>
        <Typography
            display="inline"
            sx={{
                ...commonStyles,
                display: "inline-block",
                fontSize: "1.5em",
                marginY: 1,
                fontWeight: "light",
            }}
        >
        {cents}
        </Typography>
    </div>
}

const renderPlanCard = (plan: Plan, submit: () => void) => {
    const price = currency(plan.priceInCents, { fromCents: true });
    return (<Card key={plan.key} sx={{
        maxWidth: 500,
        width: "auto",
        marginX: "auto",
        marginY: 2,
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        padding: 3,
    }}>
            <Typography variant="h5" sx={{
                color: "var(--color-accent)",
            }}>
                {plan.displayName}
            </Typography>
            {renderPrice(price)}
            <Typography
                sx={{
                    color: "var(--color-gray)",
                    marginY: 0
                }}
            >
                for 30 days
            </Typography>
            <ul style={{
                maxWidth: "400px",
                color: "var(--color-gray)"
            }}>
                {plan.features.map(feature => {
                    return <li>{feature}</li>
                })}
            </ul>
            <Button
                size="large"
                sx={{
                    maxWidth: "200px",
                    marginTop: 1,
                    marginBottom: 2,
                    textTransform: 'none',
                }}
                variant="contained"
                onClick={submit}
                endIcon={<IconArrowForward />}
            >
                Confirm & Pay
            </Button>
            <small
                style={{
                    color: "var(--color-gray)",
                }}
            >Stripe checkout will open in a new tab.</small>
    </Card>);
};

export default function SelectPlan() {
    const [ plans, setPlans ] = useState<Plan[] | undefined>();
    const [ showSuccess, setShowSuccess ] = useState(false);
    const [ errorMessage, setErrorMessage ] = useState<string | undefined>();

    useEffect(() => {
        API.listPlans()
            .then(res => {
                setPlans(res.plans);
            })
            .catch(err => {
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
    }

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
                <Box sx={{
                    marginTop: 6,
                    marginX: 1,
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center'
                }}>
                    <Avatar sx={{ m: 1, bgcolor: 'var(--color-accent)' }}>
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
                            color: "var(--color-gray)"
                        }}
                    >
                    Choose the best option to get your inbox back to clean.      
                    </Typography>
                </Box>
                {plans?.sort(p => p.priceInCents).map(p => renderPlanCard(p, () => submit(p.key)))}
            </Container>
        </>
    );
}
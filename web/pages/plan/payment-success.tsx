import { useRouter } from 'next/router';
import { useEffect } from 'react';
import { API } from "../../client";
import Container from '@mui/material/Container';
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import CircularProgress from '@mui/material/CircularProgress';

// store the success token and redirect to /plan
export default function PaymentSuccess() {
    const router = useRouter();

    useEffect(() => {
        if(!router.isReady) return;
        const query = router.query;

        const checkoutSessionId = query["checkoutSessionId"] as string;

        API.completePayment(checkoutSessionId)
            .then((res) => {
                router.push("/plan");
            })
            .catch(console.error);

    }, [router.isReady, router.query, router]);

    return (
        <Container component="main">
        <Box sx={{
                marginTop: 30,
                display: 'flex',
                flexDirection: "column",
                alignItems: "center",
            }}>
                <Typography sx={{
                    marginY: 3,
                    fontFamily: "Lobster",
                    fontSize: "3em",
                    color: "var(--color-primary)"
                }}>MailboxSweep</Typography>
                    <Typography sx={{
                    marginY: 2
                }}>
                    Completing checkout...
                </Typography>
                <CircularProgress />
        </Box>
            
        </Container>
    )
}
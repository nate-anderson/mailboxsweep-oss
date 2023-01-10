import { useRouter } from "next/router";
import { useEffect } from "react";
import { API } from "../client";
import Container from '@mui/material/Container';
import CircularProgress from '@mui/material/CircularProgress';
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";


export default function Login() {
    const router = useRouter();

    useEffect(() => {
        if(!router.isReady) return;
        const query = router.query;

        const loginToken = query["token"] as string;
        const signature = query["sig"];

        API.submitLoginToken(loginToken)
            .then((res) => {
                console.log({ loginSuccess: res });
                router.push("/home");
            })
            .catch(console.error);

    }, [router.isReady, router.query]);

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
                    Signing you in...
                </Typography>
                <CircularProgress />
        </Box>
            
        </Container>
    );
}
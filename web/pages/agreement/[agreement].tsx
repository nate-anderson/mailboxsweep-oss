import { CssBaseline } from "@mui/material";
import { useRouter } from "next/router";
import { useEffect, useState } from "react";
import { API } from "../../client";
import { UserAgreement } from "../../client-gen";
import Navbar from "../../components/navbar";
import Container from '@mui/material/Container';
import Box from "@mui/material/Box";
import Avatar from "@mui/material/Avatar";
import Typography from "@mui/material/Typography";
import IconGrading from "@mui/icons-material/Grading";
import Paper from "@mui/material/Paper";
import Button from "../../components/button";

const renderAgreedAt = (agr?: UserAgreement) => {
    return <Box>
        <Typography>
            You consented to this agreement on {agr?.agreedAt}
        </Typography>
    </Box>
}

const renderAgreementForm = (agr?: UserAgreement) => {
    return (
        <Box
            component="form"
            sx={{
                marginY: 2,
                display: 'flex',
                flexDirection: "column",
                alignItems: "center",
            }}
        >
            <Typography>
                I agree to the terms above
            </Typography>
            <Button
                sx={{
                    marginY: 2
                }}
                onClick={() => {
                    if (agr?.key) {
                        API.setConsentedAgreements({
                            agreements: [
                                agr?.key
                            ]
                        })
                        .then(() => {
                            // show success
                        })
                        .catch(() => {
                            // show failure
                        })
                    }
                }}
            >Agree</Button>
        </Box>
    );
}

export default function Agreement() {
    const router = useRouter();

    const [ agreement, setUserAgreement ] = useState<UserAgreement | undefined>();

    useEffect(() => {
        if (!router.isReady) return;

        const query = router.query;
        const { agreement: agreementCode } = router.query;
        
        API.getUserAgreements(agreementCode as string)
            .then((res) => {
                setUserAgreement(res);
                console.log({ res });
            })
            .catch((error) => {
                console.log({ getUserAgreementsError: error });
            })
    }, [router.isReady, router.query]);

    return (
        <>
            <Navbar />

            <Container component="main">
                <CssBaseline />
                <Box sx={{
                    marginTop: 6,
                    marginX: 1,
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center'
                }}>
                    <Avatar sx={{ m: 1, bgcolor: 'var(--color-accent)' }}>
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
                                fontFamily: "mono",
                                fontSize: "12px",
                                lineHeight: "1.75em",
                            }}    
                        >
                            {agreement?.body}
                        </Typography>
                    </Paper>
                    {
                        agreement?.agreedAt
                            ? renderAgreedAt(agreement)
                            : renderAgreementForm(agreement)
                    }
                </Box>
            </Container>
        </>
    );
}
import Navbar from "../../components/navbar";
import CssBaseline from "@mui/material/CssBaseline";
import Container from '@mui/material/Container';
import Box from "@mui/material/Box";
import Avatar from "@mui/material/Avatar";
import Typography from "@mui/material/Typography";
import IconFactCheck from "@mui/icons-material/FactCheck";
import { useEffect, useState } from "react";
import { Plan } from "../../client-gen";
import { API } from "../../client";
import { displayFormat } from "../../dates";
import { parseISO, format } from "date-fns";
import Link from "next/link";
import Button from "@mui/material/Button";
import IconArrowForward from "@mui/icons-material/ArrowForward";
import CircularProgress from "@mui/material/CircularProgress";
import Paper from "@mui/material/Paper";
import Table from '@mui/material/Table';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableRow from '@mui/material/TableRow';
import currency from "currency.js";


const renderPlan = (ready: boolean, plan?: Plan, validUntil?: Date, subscribedAt?: Date) => {
    if (!ready) {
        return <Box sx={{
            marginTop: 30,
            display: 'flex',
            flexDirection: "column",
            alignItems: "center",
        }}>
            <CircularProgress />
    </Box>
    }

    if (plan && validUntil && subscribedAt) {
        const price = currency(plan.priceInCents, { fromCents: true });
        return <Box sx={{
            maxWidth: 600,
            width: "auto",
            marginX: "auto",
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center'
        }}>
            <Paper
                sx={{
                    marginY: 3,
                    backgroundColor: "var(--color-accent)",
                    paddingY: 1,
                    paddingX: 2,
                    borderRadius: 2,
                    color: "white",
                    fontSize: "1.25em",
                }}
            >{plan.displayName}</Paper>

            <TableContainer component={Paper}>
                <Table sx={{
                    minWidth: 300,
                    maxWidth: 600
                }}>
                    <TableRow>
                        <TableCell>Subscribed</TableCell>
                        <TableCell>{format(subscribedAt, displayFormat)}</TableCell>
                    </TableRow>
                    <TableRow>
                        <TableCell>Expires</TableCell>
                        <TableCell>{format(validUntil, displayFormat)}</TableCell>
                    </TableRow>
                    <TableRow>
                        <TableCell>Scans Included</TableCell>
                        <TableCell>{plan.allowedScans}</TableCell>
                    </TableRow>
                    <TableRow>
                        <TableCell>Scans Remaining</TableCell>
                        <TableCell>@TODO unimplemented</TableCell>
                    </TableRow>
                    <TableRow>
                        <TableCell>Price</TableCell>
                        <TableCell>{price.format()}</TableCell>
                    </TableRow>
                    <TableRow>
                        <TableCell>Order Number</TableCell>
                        <TableCell>@TODO unimplemented</TableCell>
                    </TableRow>
                </Table>
            </TableContainer>
        </Box>;
    }

    return <Box sx={{
        maxWidth: 600,
        width: "auto",
        marginX: "auto",
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center'
    }}>
        <Typography
            variant="h6"
            sx={{
                color: "var(--color-accent)",
                marginY: 3,
            }}
        >
            You are not subscribed to a plan.
        </Typography>
        <Link passHref href="/plan/select">
            <Button
                endIcon={<IconArrowForward />}
                size="large"
                variant="contained"
                sx={{
                    marginY: 2,
                }}
            >
                View plans
            </Button>
        </Link>
    </Box>;
}

export default function ShowPlan() {
    const [ plan, setPlan ] = useState<Plan | undefined>();
    const [ validUntil, setValidUntil ] = useState<Date | undefined>();
    const [ subscribedAt, setSubscribedAt ] = useState<Date | undefined>();
    const [ ready, setReady ] = useState(false);

    useEffect(() => {
        API.getPaymentStatus()
            .then(res => {
                console.log({ res });
                if (res.plan != null) {
                    setPlan(res.plan);
                    console.log("plan");
                }

                if (res.subscribedAt != null) {
                    const subscribedAt = parseISO(res.subscribedAt);
                    setSubscribedAt(subscribedAt);
                    console.log("subscribedAt");
                }

                if (res.validUntil != null) {
                    const validUntil = parseISO(res.validUntil);
                    setValidUntil(validUntil);
                    console.log("validUntil");

                }
            })
            .catch(err => {
                console.log(err);
            })
            .finally(() => {
                setReady(true);
            })
    }, []);

    return (
        <>
            <Navbar />
            <Container component="main">
                <CssBaseline />
                <Box sx={{
                    marginTop: 6,
                    maxWidth: 600,
                    width: "auto",
                    marginX: "auto",
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center'
                }}>
                    <Avatar sx={{ m: 1, bgcolor: 'var(--color-accent)' }}>
                        <IconFactCheck />
                    </Avatar>
                    <Typography sx={{
                        color: "var(--color-black)",
                        fontSize: "2em",
                        fontWeight: 100,
                    }}>
                        Your Plan & Payment
                    </Typography>
                    {renderPlan(ready, plan, validUntil, subscribedAt)}
                </Box>
            </Container>
        </>
    );
}
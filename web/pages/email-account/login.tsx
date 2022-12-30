import { API } from "../../client";
import Navbar from "../../components/navbar"
import Container from '@mui/material/Container';
import Box from "@mui/material/Box";
import Button from "../../components/button";
import CssBaseline from "@mui/material/CssBaseline";
import Typography from "@mui/material/Typography";
import Avatar from "@mui/material/Avatar";
import TextField from '@mui/material/TextField';
import Grid from '@mui/material/Grid';
import IconEnvelope from "@mui/icons-material/Email";
import { useEffect, useState } from "react";
import { ApiError } from "../../client-gen";
import Snackbar from "@mui/material/Snackbar";
import Alert from "@mui/material/Alert";
import { parseISO, format } from "date-fns";

const displayFormat = "MMM do, yyyy";

export default function Login() {
    const [ address, changeAddress ] = useState("");
    const [ password, changePassword ] = useState("");
    const [ submitButtonEnabled, changeSubmitButtonEnabled ] = useState(false);
    const [ created, changeCreated ] = useState<Date>();
    const [ expires, changeExpires ] = useState<Date>();
    const [ credentialSet, changeCredentialSet ] = useState(false);
    const [ error, changeError ] = useState<string | undefined>(undefined); 
    const [ showSuccess, changeShowSuccess ] = useState(false);

    const resetFilledInput = () => changePassword("********");

    useEffect(() => {
        API.getInboxCredential()
            .then((res) => {
                changeAddress(res.address); 
                // 2022-12-29T23:15:17-08:00
                const createdDate = parseISO(res.created);
                if (isNaN(createdDate.valueOf())) {
                    console.log(createdDate, res.created);
                    return changeError("Invalid created date.");
                }
                changeCreated(createdDate);

                const expiresDate = parseISO(res.expires);
                if (isNaN(expiresDate.valueOf())) {
                    return changeError("Invalid expires date.");
                }
                changeExpires(expiresDate);
                changeCredentialSet(true);
                resetFilledInput();
            })
            .catch(error => {
                if (error instanceof ApiError) {
                    const e = error as ApiError;
                    if (e.status == 404) {
                        API.getUserData()
                            .then(res => {
                                changeAddress(res.address);
                            })
                            .catch(err => {
                                console.error({ getUserDataError: err });
                            })
                            return;
                    }
                }
                console.log({ getInboxCredentialError: error });
            })
    }, []);

    const submitCredential = () => {
        changeSubmitButtonEnabled(false);
        API.storeInboxCredential({
            password,
        })
        .then(() => {
            changeShowSuccess(true);
            resetFilledInput();
        })
        .catch((err) => {
            if (err instanceof ApiError) {
                const e = err as ApiError;
                if (e.status === 502) {
                    changeError("We couldn't log into your email provider with this password. Please try again.");
                }
            }
        })
    } 

    return (
        <>
            <Navbar />
            <Snackbar
                open={showSuccess}
                anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
                autoHideDuration={5000}
                onClose={() => changeShowSuccess(false)}
            >
                <Alert variant="filled" severity="success">
                Inbox credential updated!
                </Alert>
            </Snackbar>
            <Snackbar
                open={error !== undefined}
                anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
                onClose={() => changeError(undefined)}
                autoHideDuration={10000}
            >
                <Alert severity="error">{error}</Alert>
            </Snackbar>
            <Container component="main">
                <CssBaseline />
                <Box  sx={{
                    marginTop: 6,
                    marginX: 1,
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center'
                }}>
                    <Avatar sx={{ m: 1, bgcolor: 'var(--color-accent)' }}>
                        <IconEnvelope />
                    </Avatar>
                    <Typography
                        align="center"
                        sx={{
                            color: "var(--color-black)",
                            fontSize: "2em",
                            fontWeight: 100,
                        }}
                    >
                        Inbox Credentials
                    </Typography>
                    <Typography
                        align="center"
                        sx={{
                            marginY: 2,
                            color: "var(--color-gray)"
                        }}
                    >
                    Your credentials are stored with enterprise-grade AES-256 encryption and deleted after 30 days.       
                    </Typography>
                    </Box>

                    {credentialSet ?
                    <Box
                        sx={{
                            marginX: 3
                        }}    
                    >
                            <Typography
                            sx={{
                                color: "var(--color-accent)",
                                marginY: 1,
                                fontSize: 14,
                            }}
                            align="left"
                        >
                        Last updated: {format(created!, displayFormat)}  
                        </Typography>
                        <Typography
                        sx={{
                            color: "var(--color-accent)",
                            marginY: 1,
                            fontSize: 14,
                        }}
                        align="left"
                    >
                        Will be deleted at {format(expires!, displayFormat)}
                    </Typography>
                    </Box>
                        : <></>
                    }
                    
                    <Box component="form" sx={{
                        mt: 3,
                        mx: 3,
                    }}>
                        <Grid container spacing={3}>
                            <Grid item xs={12} sm={12}>
                                {/* Read-only */}
                                <TextField
                                    label="Email"
                                    fullWidth
                                    disabled
                                    value={address}
                                ></TextField>
                            </Grid>
                            <Grid item xs={12} sm={12}>
                                <TextField
                                    label="Password"
                                    type="password"
                                    autoComplete="none"
                                    fullWidth
                                    value={password}
                                    onChange={(e) => {
                                        changePassword(e.target.value);
                                        changeSubmitButtonEnabled(e.target.value !== "");
                                    }}
                                ></TextField>
                            </Grid>
                
                            <Grid item xs={12} sm={12} sx={{
                                display: "flex",
                                justifyContent: "right",
                            }}>
                                <Button
                                    disabled={!submitButtonEnabled}
                                    onClick={submitCredential}
                                >
                                    Save
                                </Button>
                            </Grid>
                        </Grid>
                    </Box>
            </Container>
        </>
    )
}
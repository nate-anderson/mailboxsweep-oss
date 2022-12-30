import { API } from "../client";
import Box from "@mui/material/Box";
import Container from '@mui/material/Container';
import TextField from '@mui/material/TextField'
import Button from '../components/button';
import { useState } from "react";
import { useRouter } from 'next/navigation';
import CssBaseline from "@mui/material/CssBaseline";
import Typography from "@mui/material/Typography";
import Modal from "@mui/material/Modal";
import LargeLogo from "../components/large-logo";

const submitLogin = async (address: string, setModalOpen: (open: boolean) => void) => {
    try {
        const res = await API.initiateLogin({
            address,
        })
        console.log({ res });
        setModalOpen(true);
    } catch (error) {
        console.log({ initiateLoginError: error });
    }
    
}

export default function Login() {
    const router = useRouter();
    const [ email, changeEmail ] = useState("");

    const [modalOpen, setModalOpen] = useState(false);

    return (
        <Container component="main">
        <Modal
            open={modalOpen}
            aria-labelledby="login-email-modal-title"
            aria-describedby="login-email-modal-text"
        >
            <Box>
                <Typography id="login-email-modal-title" variant="h6" component="h2">
                Check your email!
                </Typography>
                <Typography id="login-email-modal-text" sx={{ mt: 2 }}>
                We've sent you a link to log into Mailboxsweep. This is for your convenience and security. Please check your inbox to continue!
                </Typography>
            </Box>
        </Modal>
            <CssBaseline />
            <Box sx={{
                marginTop: 30,
                display: 'flex',
                flexDirection: "column",
                alignItems: "center",
            }}>
                <LargeLogo />
                <Typography
                    align="center"
                    sx={{
                        marginY: 2
                    }}
                >
                    Log in or sign up
                </Typography>
                <TextField
                    label="Email"
                    value={email}
                    onChange={(e) => changeEmail(e.target.value)}
                    sx={{
                        marginX: 20,
                        maxWidth: 300,
                        width: "100%"
                    }}
                ></TextField>
                <Button onClick={() => submitLogin(email, () => setModalOpen(true))} sx={{
                    marginY: 3,
                }}>
                    Sign in
                </Button>
            </Box>
        </Container>
    );
}
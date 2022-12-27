import { API } from "../client";
import Box from "@mui/material/Box";
import Container from '@mui/material/Container';
import TextField from '@mui/material/TextField'
import Button from '@mui/material/Button';
import { useState } from "react";
import { useRouter } from 'next/navigation';
import { CssBaseline, Typography } from "@mui/material";
import Modal from "@mui/material/Modal";

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
        <main>
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
                    <Typography sx={{
                        marginY: 3,
                        fontFamily: "Lobster",
                        fontSize: "3em",
                        color: "var(--color-primary)"
                    }}>MailboxSweep</Typography>
                    <Typography sx={{
                        marginY: 2
                    }}>
                        Log in or sign up
                    </Typography>
                    <TextField label="Email" value={email} onChange={(e) => changeEmail(e.target.value)}></TextField>
                    <Button size="large" onClick={() => submitLogin(email, () => setModalOpen(true))} sx={{
                        backgroundColor: "var(--color-accent)",
                        ":hover": {
                            
                        },
                        color: "white",
                        marginY: 3,
                    }}>
                        Sign in
                    </Button>
                </Box>
            </Container>
        </main>
    );
}
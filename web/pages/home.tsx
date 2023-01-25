import CssBaseline from "@mui/material/CssBaseline";
import Container from "@mui/material/Container";
import Navbar from "../components/navbar";
import Alert from "@mui/material/Alert"
import AlertTitle from "@mui/material/AlertTitle";
import Box from "@mui/material/Box";
import Card from "@mui/material/Card";
import CardContent from "@mui/material/CardContent";
import Typography from "@mui/material/Typography";
import { FolderMessageCount } from "../client-gen";
import { API } from "../client";
import { useEffect, useState } from "react";

export default function Home() {
    const [ inboxFolders, setInboxFolders ] = useState<FolderMessageCount[]>([]);
    const [ address, setAddress ] = useState("");
    const [ errorMessage, setErrorMessage ] = useState<string | undefined>();
    const [ allMessagesCount, setAllMessagesCount ] = useState<number>(0);
    const [ unreadMessagesCount, setUnreadMessagesCount ] = useState<number>(0);

    useEffect(() => {
        API.getInboxFolders()
            .then((res) => {
                setInboxFolders(res.folders);
                setAddress(res.address);
                const totalMessageCount = res.folders.reduce((sum, folder) => sum + folder.allMessages, 0);
                const unreadMessageCount = res.folders.reduce((sum, folder) => sum + folder.unreadMessages, 0);

                setAllMessagesCount(totalMessageCount);
                setUnreadMessagesCount(unreadMessageCount);
            })
            .catch((err) => {
                setErrorMessage(err as string);
            })
    }, []);

    return (
        <>
            <Navbar page="Home" />
            <Container component="main">
                <CssBaseline />
                <Box sx={{
                    marginTop: 3,
                    display: 'flex',
                    flexDirection: "column",
                    alignItems: "center",
                }}>
                    <Alert severity="info" variant="filled" sx={{
                        maxWidth: 600,
                        backgroundColor: "var(--color-accent)"
                    }}>
                        <AlertTitle>Inbox Scan In Progress</AlertTitle>
                        We're categorizing your inbox messages now. This requires going through a lot of emails and can take some time. We'll send you a message when we're ready to start cleaning up!
                    </Alert>
                </Box>
                
                <Card
                    sx={{
                        maxWidth: 600,
                        width: "auto",
                        marginTop: 3,
                        marginX: "auto",
                    }}
                >
                    <CardContent>
                        <Typography sx={{
                            color: "var(--color-gray)"
                        }}>
                            Your inbox
                        </Typography>
                        <Typography sx={{
                            fontSize: "1.6em",
                        }}>
                            {address}
                        </Typography>
                        <Typography sx={{
                            marginTop: 1,
                            color: "var(--color-gray)"
                        }}>
                            {allMessagesCount.toLocaleString()} Messages
                        </Typography>
                        <Typography sx={{
                            color: "var(--color-accent)"
                        }}>
                            {unreadMessagesCount.toLocaleString()} Unread
                        </Typography>
                    </CardContent>
                    
                    {/* If scan is complete, link to scan inside the inbox
                    card with a big "Continue ->" button or similar */}
                </Card>
            
                <Card sx={{
                    marginTop: 3,
                    maxWidth: 600,
                    marginX: "auto",
                }}>
                    <CardContent>
                        <Typography sx={{
                            color: "var(--color-gray)"
                        }}>
                            Your plan
                        </Typography>
                        <Typography sx={{
                            fontSize: "1.6em",
                            color: "var(--color-primary)",
                        }}>
                            Mailboxsweep
                        </Typography>
                        <Typography sx={{
                            marginTop: 1,
                            color: "var(--color-gray)"
                        }}>
                            Expires 12/15/2023
                        </Typography>
                        <Typography sx={{
                            color: "var(--color-accent)"
                        }}>
                            120k messages deleted
                        </Typography>
                    </CardContent>
                    
                    {/* If scan is complete, link to scan inside the inbox
                    card with a big "Continue ->" button or similar */}
                </Card>
            </Container>
        </>
    );
}
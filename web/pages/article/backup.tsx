import { Avatar, Box, Button, Container, CssBaseline, Typography } from "@mui/material";
import Navbar from "../../components/navbar";
import PrivatePage from "../../auth";
import IconSave from "@mui/icons-material/Save";
import IconOpenInNew from "@mui/icons-material/OpenInNew"

function ExternalSiteWarning() {
    return <Typography fontSize=".75em" color="var(--color-info)" marginTop={1}>This link will take you to an external website.</Typography>
}

export default function Backup() {
    return (
        <>
            <Navbar page="My Profile" />
            <Container component="main">
                <CssBaseline />
                <Box sx={{
                    maxWidth: 600,
                    width: "auto",
                    marginX: "auto",
                    display: "flex",
                    flexDirection: "column",
                    alignItems: "center",
                    marginTop: 3,
                }}>
                    <Avatar sx={{ m: 2, bgcolor: "var(--color-accent)" }}>
                        <IconSave />
                    </Avatar>
                    <Typography
                        align="center"
                        fontSize="1.75em"
                    >Back up your email inbox</Typography>
                </Box>
                <Box sx={{
                    maxWidth: 600,
                    width: "auto",
                    marginX: "auto",
                }}>
                    <Typography
                        marginY={2}
                    >
                        MailboxSweep is a powerful tool that can delete and modify thousands of messages, at your request, with the click of a button.
                        While you are in complete control of which messages we delete or mark read, it's always good practice to have a backup of important
                        data before making a big change. You should make a current backup of your inbox any time you use MailboxSweep.
                    </Typography>
                    <Typography
                        marginTop={2}
                        marginBottom={4}
                    >
                        Depending on your email provider and preferred email client, there are several ways to do this.
                    </Typography>
                    <Typography fontSize="1.25em" marginY={2} fontWeight="bold">Gmail</Typography>
                    <Typography>Google provides the <b>Takeout</b> tool for users to export their data, including comprehensive backups of your Gmail inbox.
                    It may take several hours to several days for Google to complete your request. When it's complete, you will be able to download the backup to your device.</Typography>
                    <Button
                        size="large"
                        endIcon={<IconOpenInNew />}
                        sx={{
                            backgroundColor: "var(--color-accent)",
                            marginTop: 3,
                            color: "white",
                            ":hover": {
                                backgroundColor: "#1d6db3"
                            }
                        }}
                        href="https://support.google.com/mail/answer/10016932?hl=en"
                        target="_blank"
                    >Learn about Gmail Backup</Button>
                    <ExternalSiteWarning />

                    <Typography fontSize="1.25em" marginY={2} fontWeight="bold">Outlook</Typography>
                    <Typography>
                        Regardless of your email provider, if you access your email using Microsoft Outlook via IMAP, you can create comprehensive local backups
                        of your emails. 
                    </Typography>
                    <Button
                        size="large"
                        endIcon={<IconOpenInNew />}
                        sx={{
                            backgroundColor: "var(--color-accent)",
                            marginTop: 3,
                            color: "white",
                            ":hover": {
                                backgroundColor: "#1d6db3"
                            }
                        }}
                        href="https://support.microsoft.com/en-us/office/back-up-your-email-e5845b0b-1aeb-424f-924c-aa1c33b18833"
                        target="_blank"
                    >Learn about Outlook Backup</Button>
                    <ExternalSiteWarning />

                    <Typography fontSize="1.25em" marginY={2} fontWeight="bold">Other IMAP Services</Typography>
                    <Typography>
                        If you don't use Gmail or Outlook, but your email provider supports IMAP, you can use an email client like Mozilla Thunderbird
                        to export your messages as a backup to your computer.
                    </Typography>
                    <Button
                        size="large"
                        endIcon={<IconOpenInNew />}
                        sx={{
                            backgroundColor: "var(--color-accent)",
                            marginTop: 3,
                            color: "white",
                            ":hover": {
                                backgroundColor: "#1d6db3"
                            }
                        }}
                        href="https://www.ionos.com/help/email/other-email-programs/mozilla-thunderbird-exporting-emails/"
                        target="_blank"
                    >Learn about Backup with Thunderbird</Button>
                    <ExternalSiteWarning />

                    <Typography
                        sx={{
                            marginTop: 1,
                            marginBottom: 4,
                        }}
                    >
                        <b>Note</b>: if you cannot backup your email with any of the above methods, we do not recommend that you use the MailboxSweep tool.
                        It is generally not good practice to make large changes to data that has not been backed up. Further, if your email provider does not
                        support IMAP and you do not use Gmail, MailboxSweep will be unable to access to your inbox.
                    </Typography>
                </Box>
            </Container>
        </>
    )
}
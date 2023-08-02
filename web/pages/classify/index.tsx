import Image from "next/image";
import { useAPIClient } from "../../client";
import { ApiError, Classifier } from "../../client-gen";
import { defaultClassifiers } from "../../lib/classifier";
import FolderTree from "../../public/folder-tree.svg";
import PrivatePage from "../../auth";
import Navbar from "../../components/navbar";
import { Box, Button, Container, CssBaseline, Typography } from "@mui/material";
import IconGear from "@mui/icons-material/Settings";
import IconStart from "@mui/icons-material/PlayArrow";
import { useConstantSnackbar, useSnackbar } from "../../components/snackbar";
import { useRouter } from "next/router";

export default function Classify() {
    const API = useAPIClient();

    const [successSnackbar, showSuccessSnackbar] = useConstantSnackbar(
        "Success!",
        "success",
        5000
    );
    const [errorSnackbar, showErrorSnackbar] = useSnackbar("error", 5000);
    const router = useRouter();

    const submit = () => {
        API.startClassify({
          folders: ["INBOX"],
          classifiers: defaultClassifiers,
        })
          .then((res) => {
            showSuccessSnackbar(true);
            router.push("/home");
          })
          .catch((err) => {
            if (err instanceof ApiError) {
              const apiErr = err as ApiError;
              showErrorSnackbar(err.message);
            }
          });
      };

    return <PrivatePage>
        <>
            <Navbar />
            <Container component="main">
                <CssBaseline />
                <Box
                    sx={{
                        marginTop: 3,
                        display: "flex",
                        flexDirection: "column",
                        alignItems: "center",
                    }}
                >
                    <Typography
                    align="center"
                    sx={{
                        color: "var(--color-accent)",
                        fontSize: "2em",
                        fontWeight: 100,
                    }}
                    >
                    Let's organize your inbox.
                    </Typography>
                    <Typography
                    align="center"
                    sx={{
                        color: "var(--color-black)",
                        marginTop: 2,
                        marginBottom: 2,
                        maxWidth: 600,
                        marginX: 2,
                    }}
                    >
                    We'll put all your messages into groups based on who sent them, when
                    they were sent, and what's inside. We recommend using these rules, but you
                    can organize with custom rules.<br /><br />

                    By default, your classified messages will be grouped like this:
                    </Typography>
                </Box>
                <Box
                    sx={{
                        marginTop: 3,
                        display: "flex",
                        flexDirection: "column",
                        alignItems: "center",
                        marginBottom: 6,
                    }}
                >
                    <Image
                        src={FolderTree}
                        alt="MailboxSweep Logo"
                        priority
                        style={{
                            width: "80vw",
                            height: "auto",
                            maxWidth: "300px",
                            marginBottom: "20px",
                            
                        }}
                    />
                    <Box
                        display="flex"
                        gap="6px"
                        marginTop={3}
                    >
                        <Button variant="outlined" color="info" endIcon={<IconGear />} href="/classify/configure">Customize</Button>
                        <Button variant="contained" color="info" endIcon={<IconStart />} onClick={submit}>Start</Button>
                    </Box>
                </Box>
            </Container>
        </>
    </PrivatePage>;
}
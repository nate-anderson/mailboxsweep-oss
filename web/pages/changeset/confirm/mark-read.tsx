import Container from "@mui/material/Container";
import CssBaseline from "@mui/material/CssBaseline";
import { useAPIClient } from "../../../client";
import Navbar from "../../../components/navbar";
import { ChangeSet, UpdateOperation } from "../../../client-gen";
import { useEffect, useState } from "react";
import { useConstantSnackbar } from "../../../components/snackbar";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Avatar from "@mui/material/Avatar";
import IconCheckBox from "@mui/icons-material/CheckBox";
import ConfirmBar from "../../../components/changeset/confirm-bar";
import PrivatePage from "../../../auth";
import { useRouter } from "next/router";
import { CircularProgress } from "@mui/material";

export default function ConfirmMarkRead() {
    const API = useAPIClient();
    const [ markReadChangeset, setMarkReadChangeset ] = useState<ChangeSet | undefined>();
    const [ selectedCount, setSelectedCount ] = useState<number>(0);
    const [ error, setError ] = useState<string | undefined>();
    const router = useRouter();

    const [ errorSnackbar, showErrorSnackbar ] = useConstantSnackbar('Something went wrong when loading your changeset. No changes have been made to your inbox. Please try again.', 'warning');
    const [ successSnackbar, showSuccessSnackbar ] = useConstantSnackbar('We are marking your selected messages read!', 'success');
    const [ isConfirmed, setIsConfirmed ] = useState(false);

    const [ submitIsLoading, setSubmitIsLoading ] = useState(false);
    const [ changesetIsLoading, setChangesetIsLoading ] = useState(true);

    useEffect(() => {
        API.getChangeset({
            operation: UpdateOperation.MARK_READ,
        })
        .then((res) => {
            setMarkReadChangeset(res.changeSet);
            setSelectedCount(res.size);
        })
        .catch((err) => {
            setError(err);
            showErrorSnackbar(true);
        })
        .finally(() => {
            setChangesetIsLoading(false);
        });
    }, [API, showErrorSnackbar])
    
    const submit = () => {
        setSubmitIsLoading(true);
        API.commitChangeset({
            commitRequest: {
                confirm: isConfirmed,
                operation: UpdateOperation.MARK_READ,
            }
        })
        .then((res) => {
            showSuccessSnackbar(true);
            setTimeout(() => {
                router.push("/home");
            }, 5000);
        })
        .catch((err) => {
            setError(err as string);
        })
        .finally(() => {
            setSubmitIsLoading(false);
        })
    }

    return <PrivatePage>
        <>
            <Navbar />
            <Container component="main">
                <CssBaseline />
                {errorSnackbar}
                {successSnackbar}
                <Box
                    sx={{
                        marginTop: 4,
                        display: "flex",
                        flexDirection: "column",
                        alignItems: "center",
                    }}
                >
                    {changesetIsLoading
                        ? <CircularProgress sx={{ marginTop: 6 }} />
                        : <>
                            <Avatar sx={{ m: 1, bgcolor: "var(--color-accent)" }}>
                                <IconCheckBox />
                            </Avatar>
                            <Typography
                                align="center"
                                sx={{
                                    color: "var(--color-black)",
                                    fontSize: "1.75em",
                                    fontWeight: 100,
                                }}
                            >
                                Marking <b style={{color: "var(--color-accent)"}}>{selectedCount.toLocaleString()}</b> messages read
                            </Typography>
                            <Typography
                                align="center"
                                sx={{
                                    marginY: 2,
                                    color: "var(--color-gray)",
                                    maxWidth: 600,
                                }}
                            >
                                These messages will be marked as read and remain in your inbox, no longer being counted in your unread messages. This will not delete any information, <b style={{color: "var(--color-red)"}}>but it cannot be undone.</b>
                            </Typography>
                        </>
                        }
                    <ConfirmBar
                        count={selectedCount}
                        deleteMode={false}
                        isLoading={submitIsLoading}
                        confirmButtonEnabled={isConfirmed}
                        onCheckEnabled={setIsConfirmed}
                        label="I'm sure I want to make this change"
                        onSubmit={submit}
                    />
                </Box>
            </Container>
        </>
    </PrivatePage>
}
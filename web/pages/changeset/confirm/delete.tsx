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

export default function ConfirmDelete() {
    const api = useAPIClient();
    const [ deleteChangeset, setDeleteChangeset ] = useState<ChangeSet | undefined>();
    const [ selectedCount, setSelectedCount ] = useState<number>(0);
    const [ error, setError ] = useState<string | undefined>();

    const [ errorSnackbar, showErrorSnackbar ] = useConstantSnackbar('Something went wrong when loading your changeset. No changes have been made to your inbox. Please try again.', 'warning');
    const [ successSnackbar, showSuccessSnackbar ] = useConstantSnackbar('Messages successfully deleted!', 'success');
    const [ isConfirmed, setIsConfirmed ] = useState(false);


    useEffect(() => {
        api.getChangeset(UpdateOperation.DELETE)
        .then((res) => {
            setDeleteChangeset(res.changeSet);
            setSelectedCount(res.size);
        })
        .catch((err) => {
            setError(err);
            showErrorSnackbar(true);
        });
    }, []);

    const submit = () => {
        api.commitChangeset({
            confirm: isConfirmed,
            operation: UpdateOperation.DELETE,
        })
        .then((res) => {
            showSuccessSnackbar(true);
        })
        .catch((err) => {
            setError(err as string);
        })
    }

    return <>
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
                    Deleting <b style={{color: "var(--color-accent)"}}>{selectedCount}</b> messages
                </Typography>
                <Typography
                    align="center"
                    sx={{
                        marginY: 2,
                        color: "var(--color-gray)",
                        maxWidth: 600,
                    }}
                >
                    These messages will be permanently deleted from your inbox. <b style={{color: "var(--color-danger)"}}>This cannot be undone.</b>
                </Typography>
                <ConfirmBar
                    confirmButtonEnabled={isConfirmed}
                    onCheckEnabled={setIsConfirmed}
                    label="I'm sure I want to make this change"
                    onSubmit={submit}
                />
            </Box>
        </Container>
    </>
}
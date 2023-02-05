import Container from "@mui/material/Container";
import CssBaseline from "@mui/material/CssBaseline";
import Navbar from "../../components/navbar";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Treeview from "@mui/lab/TreeView";
import TreeItem from "@mui/lab/TreeItem";
import { useEffect, useState } from "react";
import { API } from "../../client";
import { useSnackbar } from "../../components/snackbar";
import { ApiError, Classification } from "../../client-gen";
import IconExpandMore from "@mui/icons-material/ExpandMore";
import IconChevronRight from "@mui/icons-material/ChevronRight";

const displayTypeKey = (typeKey?: string): string => {
    switch (typeKey) {
        case "SENDER_DOMAIN":
            return "Sender Domain";
        case "SENDER_USERNAME":
            return "Sender Address";
        case "SEND_DATE":
            return "Send Date";
        case undefined:
            return "";
        default:
            return typeKey
                // tokenize
                .split("_")
                // lowercase all letters
                .map(word => word.toLowerCase())
                // capitalize first letter
                .map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(" ");
    }
}

type ClassificationWithId = Classification & {
    id: string
};

const renderListItem = (root: Classification | undefined) => {
    if (!root) {
        return <></>;
    }

    const topLevelClassifications = new Array<ClassificationWithId>();
    for (const [key, value] of Object.entries(root.children)) {
        topLevelClassifications.push({
            ...value,
            id: key,
        });
    }

    return topLevelClassifications.map(cl => (
        <TreeItem
            sx={{
                marginY: 1,
            }}
            nodeId={cl.id} label={displayTypeKey(cl.typeKey) + " " + cl.label}
        >
            {renderListItem(cl)}
        </TreeItem>
        )
    )
}

export default function Edit() {
    const [ classificationRoot, setClassificationRoot ] = useState<Classification | undefined>();

    const [ errorSnackbar, setError ] = useSnackbar("error");
    
    useEffect(() => {
        API.getClassificationSummary()
            .then((data) => {
                setClassificationRoot(data.classification)
            })
            .catch((err) => {
                if (err instanceof ApiError) {
                    const apiError = err as ApiError;
                    setError(apiError.message);
                }
            })
    }, []);

    return <>
        <Navbar />
        <Container>
            <CssBaseline />
            {errorSnackbar}
            <Box sx={{
                    marginTop: 4,
                    maxWidth: 600,
                    width: "auto",
                    marginX: "auto",
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'center'
                }}>
                <Typography sx={{
                    color: "var(--color-black)",
                    fontSize: "2em",
                    fontWeight: 100,
                }}>
                Select Messages
                </Typography>
                <Treeview
                    defaultCollapseIcon={<IconExpandMore />}
                    defaultExpandIcon={<IconChevronRight />}
                    sx={{ flexGrow: 1, maxWidth: 400, overflowY: 'auto' }}
                >
                    {renderListItem(classificationRoot)}
                </Treeview>
            </Box>
        </Container>
    </>;
}

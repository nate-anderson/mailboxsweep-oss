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
import SpeedDial from "@mui/material/SpeedDial";
import SpeedDialAction from "@mui/material/SpeedDialAction";
import Button from "../../components/button";
import Chip from "@mui/material/Chip";
import IconArrowForward from "@mui/icons-material/ArrowForward";
import IconArrowBackward from "@mui/icons-material/ArrowBack";
import IconDeleteForever from "@mui/icons-material/DeleteForever";
import IconMarkRead from "@mui/icons-material/MarkEmailRead";
import IconCheck from "@mui/icons-material/Check";

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

const renderChildList = (
    root: Classification | undefined,
    appendSelectedLeaf: (k: string) => void,
    removeSelectedLeaf: (k: string) => void,
    border: boolean = false,
    ) => {
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

    return topLevelClassifications
        .sort((a, b) => Number(a.count) > Number(b.count) ? -1 : 1)
        .map(cl => (
        <TreeItem
            sx={{
                marginY: 1,
                padding: 2,
                paddingRight: border ? 1 : 0,
                border: border ? "1px solid lightgray" : "none",
                borderRadius: "6px",
            }}
            nodeId={cl.id} label={
                <Box
                    display="flex"
                    justifyContent="space-between"
                >
                    <Typography>{cl.label}</Typography>
                    <Chip
                        label={cl.count} color="primary"
                        size="small"
                    />
                </Box>
            }
        >
            {renderChildList(cl, appendSelectedLeaf, removeSelectedLeaf)}
        </TreeItem>
        )
    )
}

export default function Edit() {
    const [ classificationRoot, setClassificationRoot ] = useState<Classification | undefined>();

    const [ errorSnackbar, setError ] = useSnackbar("error");

    const [ requestPage, setRequestPage ] = useState(0);

    const [ selectedLeaves, setSelectedLeaves ] = useState(new Set<string>());

    const addLeaf = (leaf: string) => setSelectedLeaves(s => s.add(leaf));
    const removeLeaf = (leaf: string) => setSelectedLeaves(s => {
        s.delete(leaf);
        return s;
    });
    
    useEffect(() => {
        console.log("USE EFFECT")
        API.getClassificationSummary(requestPage)
            .then((data) => {
                setClassificationRoot(data.classification)
            })
            .catch((err) => {
                if (err instanceof ApiError) {
                    const apiError = err as ApiError;
                    setError(apiError.message);
                }
            })
    }, [requestPage]);

    return <>
        <Navbar />
        <Container>
            <CssBaseline />
            {errorSnackbar}
            <Box sx={{
                    marginY: 4,
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
                    marginBottom: 2,
                }}>
                Select Messages
                </Typography>
                <Treeview
                    defaultCollapseIcon={<IconExpandMore />}
                    defaultExpandIcon={<IconChevronRight />}
                    sx={{ flexGrow: 1, overflowY: 'auto', width: "90vw", maxWidth: "600px" }}
                >
                    {renderChildList(classificationRoot, addLeaf, removeLeaf, true)}
                </Treeview>

                <Box
                    display="flex"
                    justifyContent="space-between"
                    sx={{
                        width: "90vw",
                        maxWidth: "600px",
                        marginTop: 1,
                        marginBottom: 8
                    }}
                >
                    <Button
                        startIcon={<IconArrowBackward />}
                        onClick={() => setRequestPage((reqPage) => reqPage - 1)}
                        hidden={requestPage > 0}
                    >
                        Back
                    </Button>
                    <Button
                        endIcon={<IconArrowForward />}
                        onClick={() => setRequestPage((reqPage) => reqPage + 1)}
                    >
                        Next
                    </Button>

                </Box>
            </Box>
        </Container>

        <SpeedDial
            ariaLabel="action menu"
            icon={<IconCheck />}
            color="info"
            FabProps={{
                size: "large"
            }}
            sx={{
                position: 'fixed',
                bottom: 20,
                left: 20,
            }}
        >
            <SpeedDialAction
                tooltipTitle="Delete"
                icon={<IconDeleteForever style={{ color: "white" }} />}
                tooltipOpen
                tooltipPlacement="right"
                FabProps={{
                    size: "large",
                    color: "error",
                    sx: {
                        bgcolor: "#ed5c5c",
                        '&:hover': {
                            bgcolor: "#ed5c5c"
                        }
                    }
                }}
            />
            <SpeedDialAction
                tooltipTitle="Mark read"
                icon={<IconMarkRead style={{ color: "white" }} />}
                tooltipOpen
                tooltipPlacement="right"
                FabProps={{
                    size: "large",
                    color: "warning",
                    sx: {
                        bgcolor: "#ff904f",
                        '&:hover': {
                            bgcolor: "#ff904f",
                        }
                    }
                }}
            />
        </SpeedDial>
    </>;
}

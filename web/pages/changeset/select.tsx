import Container from "@mui/material/Container";
import CssBaseline from "@mui/material/CssBaseline";
import Navbar from "../../components/navbar";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Treeview from "@mui/lab/TreeView";
import TreeItem from "@mui/lab/TreeItem";
import { useEffect, useRef, useState } from "react";
import { API } from "../../client";
import { useSnackbar } from "../../components/snackbar";
import { ApiError, Classification } from "../../client-gen";
import IconExpandMore from "@mui/icons-material/ExpandMore";
import IconChevronRight from "@mui/icons-material/ChevronRight";
import BottomNavigation from "@mui/material/BottomNavigation";
import BottomNavigationAction from "@mui/material/BottomNavigationAction";
import Button from "../../components/button";
import Chip from "@mui/material/Chip";
import IconArrowForward from "@mui/icons-material/ArrowForward";
import IconArrowBackward from "@mui/icons-material/ArrowBack";
import IconDeleteForever from "@mui/icons-material/DeleteForever";
import IconMarkRead from "@mui/icons-material/MarkEmailRead";
import Paper from "@mui/material/Paper";
import Pagination from "@mui/material/Pagination";
import Stack from "@mui/material/Stack";
import Switch from "@mui/material/Switch";
import AppBar from "@mui/material/AppBar";
import Toolbar from "@mui/material/Toolbar";

/*
Todo: render tree items differently if they have been selected for
mark-read or deletion.

Store the selected leaf keys in browser storage so they persist
when paginating, and when a user leaves and returns to the site
*/

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

const renderToggleModeSwitch = (isDeleteMode: boolean, setIsDeleteMode: (v: boolean) => void) => {
    return <AppBar position="fixed" color="default" sx={{ top: 'auto', bottom: 0 }}>
    <Toolbar
        sx={{
            display: "flex",
            justifyContent: "space-between",
            backgroundColor: isDeleteMode ? "#ffcccc" : "white"
        }}
    >
        <Typography
            sx={{ marginLeft: 1 }}
        >
            120,849 selected
        </Typography>
        <Stack direction="row" spacing={1} alignItems="center" sx={{
            marginRight: 1,
        }}>
            <IconMarkRead style={{ color: isDeleteMode? undefined : "#1976d2" }}/>
            <Switch
                onChange={(_, v) => setIsDeleteMode(v)}
                value={isDeleteMode}
                color={isDeleteMode ? "error" : "primary" }
            />
            <IconDeleteForever style={{ color: isDeleteMode ? "var(--color-danger)" : undefined }}/>
        </Stack>
    </Toolbar>
</AppBar>
}

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
        .map(cl => {
            const hasChildren = Object.keys(cl.children).length > 0;
            return <TreeItem
                sx={{
                    marginTop: 2,
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
        }
    )
}

export default function Edit() {
    const [ classificationRoot, setClassificationRoot ] = useState<Classification | undefined>();
    const [ errorSnackbar, setError ] = useSnackbar("error");
    const [ requestPage, setRequestPage ] = useState(0);
    const [ totalPages, setTotalPages ] = useState(0);
    const [ selectedLeaves, setSelectedLeaves ] = useState(new Set<string>());
    const [ isDeleteMode, setIsDeleteMode ] = useState(false);

    const addLeaf = (leaf: string) => setSelectedLeaves(s => s.add(leaf));
    const removeLeaf = (leaf: string) => setSelectedLeaves(s => {
        s.delete(leaf);
        return s;
    });
    
    useEffect(() => {
        console.log("USE EFFECT")
        API.getClassificationSummary(requestPage)
            .then((data) => {
                setClassificationRoot(data.classification);
                setTotalPages(data.pageCount);
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
                        marginBottom: 6
                    }}
                >
                    {/* @TODO: Use @mui/material/Pagination here instead of these dumbass buttons */}
                    <Pagination
                        count={totalPages}
                        onChange={(_, page) => setRequestPage(page - 1)}
                        sx={{
                            marginTop: 3,
                        }}
                    />

                </Box>
            </Box>
        </Container>

        {/* 
            @TODO: Replace the speed dial with a two-button navigation bar that allows switching between mark-read mode and delete mode
            It should subtly change colors so the mode is very obvious to the user
            The color of checkmarks should also change
        */}
        {renderToggleModeSwitch(isDeleteMode, setIsDeleteMode)}
        
    </>;
}

import Container from "@mui/material/Container";
import CssBaseline from "@mui/material/CssBaseline";
import Navbar from "../../components/navbar";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { API } from "../../client";
import { useSnackbar } from "../../components/snackbar";
import { ApiError, Classification } from "../../client-gen";
import IconDeleteForever from "@mui/icons-material/DeleteForever";
import IconMarkRead from "@mui/icons-material/MarkEmailRead";
import Pagination from "@mui/material/Pagination";
import Stack from "@mui/material/Stack";
import Switch from "@mui/material/Switch";
import AppBar from "@mui/material/AppBar";
import Toolbar from "@mui/material/Toolbar";
import Tooltip from "@mui/material/Tooltip";
import Leaf, { LeafProps } from "../../components/changeset/leaf";
import { ClassificationTree } from "../../lib/tree";
import Crumbs from "../../components/changeset/crumbs";
import Animation from "../../styles/animations.module.css";

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

function ToggleModeSwitch(props: {
    isDeleteMode: boolean,
    setIsDeleteMode: (v: boolean) => void,
    showDeleteTooltip: boolean,
    showMarkReadTooltip: boolean,
    numSelected: number
}) {
    return <AppBar position="fixed" color="default" sx={{ top: 'auto', bottom: 0 }}>
    <Toolbar
        sx={{
            display: "flex",
            justifyContent: "space-between",
            backgroundColor: props.isDeleteMode ? "#ffd4d4" : "white"
        }}
    >
        <Typography
            sx={{ marginLeft: 1 }}
        >
            {props.numSelected.toLocaleString()} selected
        </Typography>
        <Stack direction="row" spacing={1} alignItems="center" sx={{
            marginRight: 1,
        }}>
            <Tooltip
                title={<Typography sx={{fontSize: "1.3em"}}>Mark messages read</Typography>}
                arrow
                disableFocusListener
                open={props.showMarkReadTooltip}
            >
                <IconMarkRead style={{ color: props.isDeleteMode? undefined : "#1976d2" }}/>
            </Tooltip>
            <Switch
                onChange={(_, v) => props.setIsDeleteMode(v)}
                value={props.isDeleteMode}
                color={props.isDeleteMode ? "error" : "primary" }
            />
            <Tooltip
                title={<Typography sx={{fontSize: "1.3em"}}>Permanently delete messages</Typography>}
                arrow
                disableFocusListener
                open={props.showDeleteTooltip}
            >
                <IconDeleteForever style={{ color: props.isDeleteMode ? "var(--color-danger)" : undefined }}/>
            </Tooltip>
        </Stack>
    </Toolbar>
</AppBar>
}

const classificationToLeafProps = (
    children: Record<string, Classification> | undefined, 
    navigate: (key: string) => void,
    appendSelectedNode: (node: string) => void,
    removeSelectedNode: (node: string) => void,
): LeafProps[] => {
    if (!children) {
        return [];
    }

    return Object
        .entries(children)
        .map(([key, value]) => ({
            label: value.label,
            msgCount: value.count,
            unreadCount: 0,
            onClickExpand: (key: string) => navigate(key),
            id: key,
            childCount: Object.keys(value.children).length,
            isChild: true,
            appendSelectedNode,
            removeSelectedNode,
        }))
        .sort((a, b) => Number(a.msgCount) > Number(b.msgCount) ? -1 : 1);
}

interface ListProps {
    selected: Classification | undefined,
    appendSelectedLeaf: (k: string) => void,
    removeSelectedLeaf: (k: string) => void,
    navigate: (key: string) => void,
    slideBackward: boolean,
    setSlideBackward: (d: boolean) => void
}

function List(props: ListProps) {
    if (!props.selected) {
        return <></>;
    }

    return <div
        className={props.slideBackward ? Animation.slideRight : Animation.slideLeft}
        key={props.selected.label}
    >
        {Object.entries(props.selected.children)
        .map(([key, value]) => ({
            ...value,
            id: key,
        }))
        .sort((a, b) => Number(a.count) > Number(b.count) ? -1 : 1)
        .map(cl => <Leaf
            id={cl.id}
            key={cl.id}
            label={cl.label}
            msgCount={cl.count}
            unreadCount={0}
            childCount={Object.keys(cl.children).length}
            onClickExpand={(node) => {
                props.setSlideBackward(false);
                props.navigate(node);
            }}
            appendSelectedNode={props.appendSelectedLeaf}
            removeSelectedNode={props.removeSelectedLeaf}
        />
        )}
        </div>;
}

export default function Edit() {
    const [ classificationRoot, setClassificationRoot ] = useState<ClassificationTree | undefined>();
    const [ errorSnackbar, setError ] = useSnackbar("error");
    const [ requestPage, setRequestPage ] = useState(0);
    const [ totalPages, setTotalPages ] = useState(0);
    const [ selectedLeaves, setSelectedLeaves ] = useState(new Set<string>());
    const [ selectedCount, setSelectedCount ] = useState(0);
    const [ isDeleteMode, setIsDeleteMode ] = useState(false);
    const [ slideBackward, setSlideBackward ] = useState(false); 

    const [ showDeleteTooltip, setShowDeleteTooltip ] = useState(false);
    const [ showMarkReadTooltip, setShowMarkReadTooltip ] = useState(false);

    const [ currentPath, setCurrentPath ] = useState<string[]>([]);
    const navigateDownward = (key: string | undefined) => {
        if (key) {
            return setCurrentPath(path => [...path, key]);
        }
        return setCurrentPath([]);
    }

    const addLeaf = (leaf: string) => {
        const leaves = setSelectedLeaves(s => s.add(leaf))
    };
    const removeLeaf = (leaf: string) => setSelectedLeaves(s => {
        s.delete(leaf);
        return s;
    });

    const toggleMode = (isDel: boolean) => {
        setIsDeleteMode(isDel);
        if (isDel) {
            setShowMarkReadTooltip(false);
            setShowDeleteTooltip(true);
            setTimeout(() => {
                setShowDeleteTooltip(false);
            }, 5000)
        } else {
            setShowDeleteTooltip(false);
            setShowMarkReadTooltip(true);
            setTimeout(() => {
                setShowMarkReadTooltip(false);
            }, 5000);
        }
    }
    
    useEffect(() => {
        API.getClassificationSummary(requestPage)
            .then((data) => {
                setClassificationRoot(new ClassificationTree(data.classification));
                setTotalPages(data.pageCount);
            })
            .catch((err) => {
                if (err instanceof ApiError) {
                    const apiError = err as ApiError;
                    setError(apiError.message);
                }
            })
    }, [requestPage]);

    const selectedClassification = classificationRoot
        ? classificationRoot.jumpToPath(currentPath)
        : undefined;

    return <>
        <Navbar />
        <Container disableGutters>
            <CssBaseline />
            {errorSnackbar}
            <Box sx={{
                marginTop: 2,
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center'
            }}>
                <Box sx={{
                        maxWidth: 600,
                        width: "100%",
                        display: 'flex',
                        flexDirection: 'column',
                        alignItems: 'center'
                    }}>
                    <Crumbs
                            currentPath={currentPath}
                            onClickBack={() => {
                                setSlideBackward(true);
                                setCurrentPath(currentPath.slice(0, -1))
                            }}
                            title={selectedClassification?.label}
                        /> 
                    </Box>
                    <Box
                        sx={{
                            marginBottom: 2,
                            maxWidth: 600,
                            width: "auto",
                            display: 'flex',
                            flexDirection: 'column',
                            alignItems: 'center'
                        }}
                    >
                        <List
                            selected={selectedClassification}
                            appendSelectedLeaf={addLeaf}
                            removeSelectedLeaf={removeLeaf}
                            navigate={navigateDownward}
                            slideBackward={slideBackward}
                            setSlideBackward={setSlideBackward}
                        />

                    <Box
                        display="flex"
                        justifyContent="center"
                        sx={{
                            width: "90vw",
                            maxWidth: "600px",
                            marginBottom: 6
                        }}
                    >
                        {
                            currentPath.length <= 0
                                ? <Pagination
                                    page={requestPage + 1}
                                    count={totalPages}
                                    onChange={(_, page) => setRequestPage(page - 1)}
                                    sx={{
                                        marginY: 3,
                                    }}
                                />
                                : <></>
                        }
                        
                    </Box>
                </Box>
            </Box>
        </Container>

        <ToggleModeSwitch
            isDeleteMode={isDeleteMode}
            setIsDeleteMode={toggleMode}
            showDeleteTooltip={showDeleteTooltip}
            showMarkReadTooltip={showMarkReadTooltip}
            numSelected={selectedCount}
        />
    </>;
}

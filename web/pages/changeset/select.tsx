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
import Leaf, { LeafProps } from "../../components/changeset/leaf";
import { ClassificationTree } from "../../lib/tree";
import Crumbs from "../../components/changeset/crumbs";
import Animation from "../../styles/animations.module.css";
import Button from "@mui/material/Button";

/*
Todo: render tree items differently if they have been selected for
mark-read or deletion.

Store the selected leaf keys in browser storage so they persist
when paginating, and when a user leaves and returns to the site
*/

function ToggleModeSwitch(props: {
    isDeleteMode: boolean,
    setIsDeleteMode: (v: boolean) => void,
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
        <Button
            variant="contained"
            size="small"
            color={props.isDeleteMode ? "error" : undefined}
            disabled={props.numSelected == 0}
        >
            {
                props.isDeleteMode
                    ? `Delete ${props.numSelected}`
                    : `Mark ${props.numSelected} read`
            }
        </Button>
        <Stack direction="row" alignItems="center">
            <IconMarkRead style={{ color: props.isDeleteMode? undefined : "#1976d2" }}/>
            <Switch
                onChange={(_, v) => props.setIsDeleteMode(v)}
                value={props.isDeleteMode}
                color={props.isDeleteMode ? "error" : "primary" }
                sx={{ marginLeft: 0 }}
            />
            <IconDeleteForever style={{ color: props.isDeleteMode ? "var(--color-danger)" : undefined }}/>
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
                        typeKey={selectedClassification?.typeKey}
                    /> 
                    {
                        currentPath.length == 0
                            ? <Typography
                                align="center"
                                sx={{
                                    marginY: 1,
                                    marginX: 1,
                                    color: "var(--color-gray)",
                                    maxWidth: 600,
                                }}
                            >
                                Select a category to update all messages in that group, or expand the category to make specific selections.       
                            </Typography>
                            : undefined
                    }
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
                                    size="small"
                                    page={requestPage + 1}
                                    count={totalPages}
                                    onChange={(_, page) => setRequestPage(page - 1)}
                                    sx={{
                                        marginY: 3,
                                        // hack to make it fit on small phone screens
                                        marginX: -1,
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
            setIsDeleteMode={setIsDeleteMode}
            numSelected={selectedCount}
        />
    </>;
}

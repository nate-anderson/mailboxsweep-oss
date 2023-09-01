import Container from "@mui/material/Container";
import CssBaseline from "@mui/material/CssBaseline";
import Navbar from "../../components/navbar";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { useEffect, useState } from "react";
import { useAPIClient } from "../../client";
import { useSnackbar } from "../../components/snackbar";
import {
  ApiError,
  ChangeSetRequest,
  Classification,
  UpdateOperation,
} from "../../client-gen";
import IconDeleteForever from "@mui/icons-material/DeleteForever";
import IconMarkRead from "@mui/icons-material/MarkEmailRead";
import Pagination from "@mui/material/Pagination";
import Stack from "@mui/material/Stack";
import Switch from "@mui/material/Switch";
import AppBar from "@mui/material/AppBar";
import Toolbar from "@mui/material/Toolbar";
import Leaf from "../../components/changeset/leaf";
import { ClassificationTree } from "../../lib/tree";
import Crumbs from "../../components/changeset/crumbs";
import Animation from "../../styles/animations.module.css";
import { Selection } from "../../lib/selection";
import debounce from "debounce";
import { useRouter } from "next/router";
import PrivatePage from "../../auth";
import CircularProgress from "@mui/material/CircularProgress";
import LoadingButton from "@mui/lab/LoadingButton";

function ToggleModeSwitch(props: {
  isDeleteMode: boolean;
  setIsDeleteMode: (v: boolean) => void;
  numSelected: number;
  onClickSubmit: (isDeleteMode: boolean) => void;
  isLoading: boolean;
}) {
  return (
    <AppBar position="fixed" color="default" sx={{ top: "auto", bottom: 0 }}>
      <Toolbar
        sx={{
          display: "flex",
          justifyContent: "space-between",
          backgroundColor: props.isDeleteMode ? "#ffd4d4" : "white",
        }}
      >
        <LoadingButton
          variant="contained"
          size="small"
          color={props.isDeleteMode ? "error" : undefined}
          disabled={props.numSelected <= 0}
          onClick={() => props.onClickSubmit(props.isDeleteMode)}
          loading={props.isLoading}
        >
          {props.isDeleteMode
            ? `Delete ${props.numSelected.toLocaleString()}`
            : `Mark ${props.numSelected.toLocaleString()} read`}
        </LoadingButton>
        <Stack direction="row" alignItems="center">
          <IconMarkRead
            style={{ color: props.isDeleteMode ? undefined : "#1976d2" }}
          />
          <Switch
            onChange={(_, v) => props.setIsDeleteMode(v)}
            
            checked={props.isDeleteMode}
            color={props.isDeleteMode ? "error" : "primary"}
            sx={{ marginLeft: 0 }}
          />
          <IconDeleteForever
            style={{
              color: props.isDeleteMode ? "var(--color-danger)" : undefined,
            }}
          />
        </Stack>
      </Toolbar>
    </AppBar>
  );
}

interface ListProps {
  selected: Classification | undefined;
  appendSelectedLeaf: (path: string[]) => void;
  removeSelectedLeaf: (path: string[]) => void;
  navigate: (key: string) => void;
  slideBackward: boolean;
  setSlideBackward: (d: boolean) => void;
  pathPrefix: string[];
  classificationTree: ClassificationTree | undefined;
  selectedNodes: Selection;
}

function List(props: ListProps) {
  if (!props.selected) {
    return <></>;
  }

  return (
    <div
      className={
        props.slideBackward ? Animation.slideRight : Animation.slideLeft
      }
      key={`${props.selected.label}`}
    >
      {Object.entries(props.selected.children)
        .map(([key, value]) => ({
          ...value,
          id: key,
        }))
        .sort((a, b) => (Number(a.count) > Number(b.count) ? -1 : 1))
        .map((cl) => {
          const path = [...props.pathPrefix, cl.id];
          const childLeaves =
            props.classificationTree?.getLeavesOfPath(path) ?? [];
          const isChecked =
            props.selectedNodes.has(path) ||
            (childLeaves.length > 0 && props.selectedNodes.hasAll(childLeaves));
          const isChildChecked =
            props.selectedNodes.hasAny(childLeaves) && !isChecked;
          return (
            <Leaf
              id={cl.id}
              path={[...props.pathPrefix, cl.id]}
              key={`${cl.id}_${isChecked}_${isChildChecked}`}
              label={cl.label}
              msgCount={cl.count}
              unreadCount={cl.unreadCount}
              childCount={Object.keys(cl.children).length}
              onClickExpand={(node) => {
                props.setSlideBackward(false);
                props.navigate(node);
              }}
              appendSelectedNode={props.appendSelectedLeaf}
              removeSelectedNode={props.removeSelectedLeaf}
              isChecked={isChecked}
              isChildChecked={isChildChecked}
              onCheck={(v: boolean) => {}}
              classification={props.classificationTree}
            />
          );
        })}
    </div>
  );
}

function LoadingSpinner() {
  return <Box
    sx={{
      marginTop: "25vh",
      display: "flex",
      flexDirection: "column",
      alignItems: "center",
    }}
  >
    <CircularProgress />
  </Box>
}

export default function Select() {
  const [classificationRoot, setClassificationRoot] = useState<
    ClassificationTree | undefined
  >();
  const [errorSnackbar, setError] = useSnackbar("error");
  const [requestPage, setRequestPage] = useState(0);
  const [totalPages, setTotalPages] = useState(0);
  const [deleteCount, setDeleteCount] = useState(0);
  const [markReadCount, setMarkReadCount] = useState(0);
  const [isDeleteMode, setIsDeleteMode] = useState(false);
  const debouncedSetIsDeleteMode = debounce(setIsDeleteMode, 500);
  const [slideBackward, setSlideBackward] = useState(false);

  const [currentPath, setCurrentPath] = useState<string[]>([]);

  const [deleteSelection, setDeleteSelection] = useState(new Selection([]));
  const [markReadSelection, setMarkReadSelection] = useState(new Selection([]));

  const [isLoadingClassificationSummary, setIsLoadingClassificationSummary] = useState(false);
  const [isLoadingSelection, setIsLoadingSelection] = useState(false);
  const [isLoadingSelectionChange, setIsLoadingSelectionChange] = useState(false);

  const router = useRouter();
  const API = useAPIClient();


  const navigateDownward = (key: string | undefined) => {
    if (key) {
      return setCurrentPath((path) => [...path, key]);
    }
    return setCurrentPath([]);
  };

  const addLeaf = async (path: string[]) => {
    const operation = isDeleteMode
    ? UpdateOperation.DELETE
    : UpdateOperation.MARK_READ;

    setIsLoadingSelectionChange(true);

    // temporarily add this exact path to the delete selection for optimistic UI
    // when the API returns the leaf paths, remove this path and replace it with
    // those
    if (isDeleteMode) {
      setDeleteSelection((sel) => sel.add(path));
    } else {
      setMarkReadSelection((sel) => sel.add(path));
    }

    API.updateChangeSet({
      operation,
      path,
      addOrRemove: ChangeSetRequest.addOrRemove.ADD,
    }).then((res) => {
      if (isDeleteMode) {
        // remove intermediate path
        setDeleteSelection((sel) => sel.remove(path));
        
        setDeleteSelection((sel) => {
          res.updatedPaths.forEach((p) => sel.add(p));
          return sel;
        });
        setDeleteCount((count) => count + res.messageCount);
      } else {
        // remove intermediate path
        setMarkReadSelection((sel) => sel.remove(path));
        
        setMarkReadSelection((sel) => {
          res.updatedPaths.forEach((p) => sel.add(p));
          return sel;
        });
        setMarkReadCount((count) => count + res.messageCount);
      }
    })
    .catch((error)=>{
      console.log({ addLeafError: error });
    })
    .finally(() => {
      setIsLoadingSelectionChange(false);
    });
  };

  const getSelection = (): Selection => {
    if (isDeleteMode) {
      return deleteSelection;
    }
    return markReadSelection;
  };

  const removeLeaf = async (path: string[]) => {
    const operation = isDeleteMode
    ? UpdateOperation.DELETE
    : UpdateOperation.MARK_READ;
    
    setIsLoadingSelectionChange(true);

    // temporarily add this exact path to the exclude set for optimistic UI
    // when the API returns the leaf paths, remove this path and replace it with
    // those
    if (isDeleteMode) {
      setDeleteSelection((sel) => sel.optimisticExclude(path));
    } else {
      setMarkReadSelection((sel) => sel.optimisticExclude(path));
    }

    API.updateChangeSet({
      operation,
      path,
      addOrRemove: ChangeSetRequest.addOrRemove.REMOVE,
    }).then((res) => {
      if (isDeleteMode) {
        setDeleteSelection((sel) => {
          // remove optimistic exclude
          const updated = sel.clearOptimisticExclude(path);
          res.updatedPaths.forEach((p) => updated.remove(p));
          return updated;
        });
        setDeleteCount((count) => count + res.messageCount);
      } else {
        setMarkReadSelection((sel) => {
          //  remove optimistic exclude
          const updated = sel.clearOptimisticExclude(path);
          res.updatedPaths.forEach((p) => updated.remove(p));
          return updated;
        });
        setMarkReadCount((count) => count + res.messageCount);
      }
    })
    .catch((error) => {
      console.log({ removeLeafError: error });
    })
    .finally(() => {
      setIsLoadingSelectionChange(false);
    });
  };

  useEffect(() => {
    if (!router.isReady) return;
    const query = router.query;
    const mode = query["mode"];
    if (typeof mode === "string") {
      if (mode === "delete") {
        setIsDeleteMode(true);
        router.push({
            query: {},
          },
          undefined,
        );
      }
    }

    const updateOperation = isDeleteMode ? UpdateOperation.DELETE : UpdateOperation.MARK_READ;

    setIsLoadingClassificationSummary(true);
    API.getClassificationSummary(requestPage)
      .then((classificationRes) => {
        setClassificationRoot(new ClassificationTree(classificationRes.classification));
        setTotalPages(classificationRes.pageCount);
      })
      .catch((error) => {
        if (error instanceof ApiError) {
          const apiError = error as ApiError;
          setError(apiError.message);
        }
        console.log({ error });
      })
      .finally(() => {
        setIsLoadingClassificationSummary(false);
      });
      
    setIsLoadingSelection(true);
    API.getChangeset(updateOperation).then((changesetRes) => {
      if (isDeleteMode) {
        setDeleteSelection(new Selection(changesetRes.changeSet.paths));
        setDeleteCount(changesetRes.size);
      } else {
        setMarkReadSelection(new Selection(changesetRes.changeSet.paths));
        setMarkReadCount(changesetRes.size);
      }
    })
    .catch((error) => {
      if (error instanceof ApiError) {
        const apiError = error as ApiError;
        // 404 means no selection has been stored 
        if (apiError.status !== 404) {
          setError(apiError.message);
        }
      }
    })
    .finally(() => {
      setIsLoadingSelection(false);
    })
      // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [requestPage, isDeleteMode, router.isReady, router.query, router]);

  const selectedClassification = classificationRoot
    ? classificationRoot.jumpToPath(currentPath)
    : undefined;

  const navigateOnSubmit = async (isDeleteMode: boolean): Promise<void> => {
    if (isDeleteMode) {
      await router.push("/changeset/confirm/delete");
      return;
    } else {
      await router.push("/changeset/confirm/mark-read");
      return;
    }
  }

  return (
    <PrivatePage>
      <>
        <Navbar />
        <Container disableGutters>
          <CssBaseline />
          {errorSnackbar}
          <Box
            sx={{
              marginTop: 2,
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
            }}
          >
            { isLoadingClassificationSummary || isLoadingSelection
              ? <LoadingSpinner />
              :  <>
              <Box
                sx={{
                  maxWidth: 600,
                  width: "100%",
                  display: "flex",
                  flexDirection: "column",
                  alignItems: "center",
                }}
              >
                <Crumbs
                  currentPath={currentPath}
                  onClickBack={() => {
                    setSlideBackward(true);
                    setCurrentPath(currentPath.slice(0, -1));
                  }}
                  title={selectedClassification?.label}
                  typeKey={selectedClassification?.typeKey}
                />
                {currentPath.length == 0 ? (
                  <Typography
                    align="center"
                    sx={{
                      marginY: 1,
                      marginX: 1,
                      color: "var(--color-gray)",
                      maxWidth: 600,
                    }}
                  >
                    Select a category to update all messages in that group, or
                    expand the category to make specific selections.
                  </Typography>
                ) : undefined}
              </Box>
              <Box
                sx={{
                  marginBottom: 2,
                  maxWidth: 600,
                  width: "auto",
                  display: "flex",
                  flexDirection: "column",
                  alignItems: "center",
                }}
              >
                <List
                  selected={selectedClassification}
                  appendSelectedLeaf={addLeaf}
                  removeSelectedLeaf={removeLeaf}
                  navigate={navigateDownward}
                  slideBackward={slideBackward}
                  setSlideBackward={setSlideBackward}
                  pathPrefix={currentPath}
                  classificationTree={classificationRoot}
                  selectedNodes={getSelection()}
                />

                <Box
                  display="flex"
                  justifyContent="center"
                  sx={{
                    width: "90vw",
                    maxWidth: "600px",
                    marginBottom: 6,
                  }}
                >
                  {currentPath.length <= 0 ? (
                    <Pagination
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
                  ) : (
                    <></>
                  )}
                </Box>
              </Box>
            </> }
          </Box>
        </Container>

        <ToggleModeSwitch
          onClickSubmit={navigateOnSubmit}
          isDeleteMode={isDeleteMode}
          setIsDeleteMode={debouncedSetIsDeleteMode}
          numSelected={isDeleteMode ? deleteCount : markReadCount}
          isLoading={isLoadingSelectionChange}
        />
      </>
    </PrivatePage>
  );
}

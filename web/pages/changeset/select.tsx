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
import Button from "@mui/material/Button";
import { Selection } from "../../lib/selection";
import debounce from "debounce";
import { useRouter } from "next/router";

/*
Todo: render tree items differently if they have been selected for
mark-read or deletion.

Store the selected leaf keys in browser storage so they persist
when paginating, and when a user leaves and returns to the site
*/

function ToggleModeSwitch(props: {
  isDeleteMode: boolean;
  setIsDeleteMode: (v: boolean) => void;
  numSelected: number;
  onClickSubmit: (isDeleteMode: boolean) => void;
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
        <Button
          variant="contained"
          size="small"
          color={props.isDeleteMode ? "error" : undefined}
          disabled={props.numSelected <= 0}
          onClick={() => props.onClickSubmit(props.isDeleteMode)}
        >
          {props.isDeleteMode
            ? `Delete ${props.numSelected}`
            : `Mark ${props.numSelected} read`}
        </Button>
        <Stack direction="row" alignItems="center">
          <IconMarkRead
            style={{ color: props.isDeleteMode ? undefined : "#1976d2" }}
          />
          <Switch
            onChange={(_, v) => props.setIsDeleteMode(v)}
            value={props.isDeleteMode}
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

export default function Edit() {
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
  const router = useRouter()

  const navigateDownward = (key: string | undefined) => {
    if (key) {
      return setCurrentPath((path) => [...path, key]);
    }
    return setCurrentPath([]);
  };

  const API = useAPIClient();

  const addLeaf = async (path: string[]) => {
    const res = await API.updateChangeSet({
      operation: isDeleteMode
        ? UpdateOperation.DELETE
        : UpdateOperation.MARK_READ,
      path,
      addOrRemove: ChangeSetRequest.addOrRemove.ADD,
    });
    if (isDeleteMode) {
      setDeleteSelection((sel) => {
        res.updatedPaths.forEach((path) => sel.add(path));
        return sel;
      });
      setDeleteCount((count) => count + res.messageCount);
    } else {
      setMarkReadSelection((sel) => {
        res.updatedPaths.forEach((path) => sel.add(path));
        return sel;
      });
      setMarkReadCount((count) => count + res.messageCount);
    }
  };

  const getSelection = (): Selection => {
    if (isDeleteMode) {
      return deleteSelection;
    }
    return markReadSelection;
  };

  const removeLeaf = async (path: string[]) => {
    const res = await API.updateChangeSet({
      operation: isDeleteMode
        ? UpdateOperation.DELETE
        : UpdateOperation.MARK_READ,
      path,
      addOrRemove: ChangeSetRequest.addOrRemove.REMOVE,
    });

    if (isDeleteMode) {
      setDeleteSelection((sel) => {
        res.updatedPaths.forEach((p) => sel.remove(p));
        return sel;
      });
      setDeleteCount((count) => count + res.messageCount);
    } else {
      setMarkReadSelection((sel) => {
        res.updatedPaths.forEach((p) => sel.remove(p));
        return sel;
      });
      setMarkReadCount((count) => count + res.messageCount);
    }
  };

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
      });

    API.getChangeset(
      isDeleteMode ? UpdateOperation.DELETE : UpdateOperation.MARK_READ
    )
      .then((data) => {
        if (isDeleteMode) {
          setDeleteSelection(new Selection(data.changeSet.paths));
          setDeleteCount(data.size);
        } else {
          setMarkReadSelection(new Selection(data.changeSet.paths));
          setMarkReadCount(data.size);
        }
      })
      .catch((err) => {
        if (err instanceof ApiError) {
          const apiError = err as ApiError;
          setError(apiError.message);
        }
      });
  
      // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [requestPage, isDeleteMode]);

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
        </Box>
      </Container>

      <ToggleModeSwitch
        onClickSubmit={navigateOnSubmit}
        isDeleteMode={isDeleteMode}
        setIsDeleteMode={debouncedSetIsDeleteMode}
        numSelected={isDeleteMode ? deleteCount : markReadCount}
      />
    </>
  );
}

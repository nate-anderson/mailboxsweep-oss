import { useEffect, useState } from "react";
import { useAPIClient } from "../../client";
import { ApiError, ClassifierInfo } from "../../client-gen";
import Navbar from "../../components/navbar";
import Container from "@mui/material/Container";
import CssBaseline from "@mui/material/CssBaseline";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import { SelectedClassifier } from "../../lib/classifier";
import Button from "../../components/button";
import IconButton from "@mui/material/IconButton";
import IconPlus from "@mui/icons-material/Add";
import type {} from "@mui/x-date-pickers/themeAugmentation";
import SelectedClassifierList from "../../components/classifier-inputs/list";
import { useConstantSnackbar, useSnackbar } from "../../components/snackbar";
import { useRouter } from "next/router";
import PrivatePage from "../../auth";
import { defaultClassifiers } from "../../lib/classifier";
import { AddClassifierDialog } from "../../components/classifier-inputs/modal";
import ChipMultiselect from "../../components/chip-multiselect";

const defaultFolders = ["INBOX", "inbox", "Inbox"];

export default function Configure() {
  const API = useAPIClient();
  const [expandedIndex, setExpandedIndex] = useState<number | undefined>();
  const [error, setError] = useState<string | undefined>();
  const [selectedClassifiers, setSelectedClassifiers] = useState<
    SelectedClassifier[]
  >([]);
  const [allClassifiers, setAllClassifiers] = useState<ClassifierInfo[]>([]);
  const [allFolders, setAllFolders] = useState(new Set<string>());
  const [selectedFolders, setSelectedFolders] = useState(new Set<string>());
  const [successSnackbar, showSuccessSnackbar] = useConstantSnackbar(
    "Success!",
    "success",
    5000
  );
  const [errorSnackbar, showErrorSnackbar] = useSnackbar("error", 5000);

  const router = useRouter();
  const [dialogIsOpen, setDialogIsOpen] = useState(false);

  useEffect(() => {
    API.getClassifiers()
      .then((res) => {
        setAllClassifiers(res.classifiers);
        let hydratedClassifiers: SelectedClassifier[];

        try {
          hydratedClassifiers = defaultClassifiers.map((cl) => {
            const classifier = res.classifiers.find(
              (c) => c.typeKey === cl.typeKey
            );
            if (!classifier) {
              throw new Error(
                `classifier with typeKey ${cl.typeKey} was not found in API response`
              );
            }

            return {
              ...cl,
              name: classifier.name,
              description: classifier.description,
              hasInput: classifier.hasInput,
              inputFormat: classifier.inputFormat,
              typeKey: classifier.typeKey,
            };
          });
          setSelectedClassifiers(hydratedClassifiers);
        } catch (error) {
          setError(error as string);
        }
      })
      .catch((err) => {
        setError(err);
      });

    API.getInboxMetadata()
      .then((res) => {
        setAllFolders(new Set(res.folders));
        defaultFolders.forEach(defaultFolder => {
          if (res.folders.includes(defaultFolder)) {
            setSelectedFolders(f => new Set([...f, defaultFolder]));
          }
        })
      })
      .catch((err) => {
        setError(err);
      });
  }, [API]);

  const submit = () => {
    API.startClassify({
      folders: [...selectedFolders],
      classifiers: selectedClassifiers,
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

  return (
    <PrivatePage>
      <>
        <Navbar />
        <Container component="main">
          <AddClassifierDialog
            open={dialogIsOpen}
            onClose={() => setDialogIsOpen(false)}
            availableClassifiers={allClassifiers}
            addClassifier={(cl: ClassifierInfo) => setSelectedClassifiers((current) => ([...current, {
              typeKey: cl.typeKey,
              name: cl.name,
              description: cl.description,
              hasInput: cl.hasInput,
              inputLabel: cl.inputLabel,
              inputFormat: cl.inputFormat,
            }]))}
          />
          <CssBaseline />
          {successSnackbar}
          {errorSnackbar}
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
              Custom organization
            </Typography>
            <Typography
              align="center"
              sx={{
                color: "var(--color-black)",
                marginTop: 2,
                marginBottom: 1,
                maxWidth: 600,
                marginX: 2,
              }}
            >
              Messages will be broken into increasingly specific groups based on the rules you set here. Rules at the top will be applied first,
              and messages in those groups will be further broken down into smaller groups based on the rules further down the list.
            </Typography>
          </Box>
          <Box
            sx={{
              maxWidth: 600,
              width: "auto",
              marginTop: 1,
              marginX: "auto",
            }}
          >
            <Typography
              fontWeight="bold"
              marginLeft="12px"
            >
              Folders to sort:
            </Typography>
            <ChipMultiselect
              items={new Set(allFolders)}
              selectedItems={selectedFolders}
              onAdd={(name: string) => setSelectedFolders(f => new Set([...f, name]))}
              onRemove={(name: string) => setSelectedFolders(f => new Set([...f].filter(x => x !== name)))}
            />
            <Typography
              fontWeight="bold"
              marginLeft="12px"
              marginTop={2}
            >
              Classification filters:
            </Typography>
            <SelectedClassifierList
              setSelectedClassifiers={setSelectedClassifiers}
              classifiers={selectedClassifiers}
              expandedIndex={expandedIndex}
              setExpandedIndex={setExpandedIndex}
            />
            <Box
              display="flex"
              justifyContent="space-between"
              sx={{
                marginBottom: 3,
              }}
            >
              <IconButton color="primary" onClick={() => setDialogIsOpen(true)}>
                <IconPlus />
              </IconButton>
              <Button
                onClick={submit}
              >
                Submit
              </Button>
            </Box>
          </Box>
        </Container>
      </>
    </PrivatePage>
  );
}

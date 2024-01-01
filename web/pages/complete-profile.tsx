import { useAPIClient } from "../client";
import { useEffect, useState } from "react";
import CssBaseline from "@mui/material/CssBaseline";
import Container from "@mui/material/Container";
import Box from "@mui/material/Box";
import Navbar from "../components/navbar";
import IconAccountCircle from "@mui/icons-material/AccountCircle";
import TextField from "@mui/material/TextField";
import Grid from "@mui/material/Grid";
import Button from "../components/button";
import { useSnackbar, useConstantSnackbar } from "../components/snackbar";
import PrivatePage from "../auth";
import PageHeader from "../components/page-header";
import { nextStep } from "../lib/steps";
import { useRouter } from "next/router";

export default function CompleteProfile() {
  const [nameFirst, changeNameFirst] = useState("");
  const [nameLast, changeNameLast] = useState("");
  const [address, changeAddress] = useState("");

  const [saveButtonEnabled, setSaveButtonEnabled] = useState(false);

  const [successSnackbar, setShowSuccess] = useConstantSnackbar(
    "Profile data updated!",
    "success"
  );
  const [errorSnackbar, setError] = useSnackbar("error");

  const API = useAPIClient();
  const router = useRouter();

  useEffect(() => {
    API.getUserData()
      .then((res) => {
        changeAddress(res.address);
        if (res.nameFirst) {
          changeNameFirst(res.nameFirst);
        }
        if (res.nameLast) {
          changeNameLast(res.nameLast);
        }
      })
      .catch((error) => {
        setError("Failed to fetch user data, please try again momentarily.");
      });
  }, [API]);

  const submitProfileData = (nameFirst: string, nameLast: string) => {
    API.updateUserName({
      nameFirst,
      nameLast,
    })
      .then(() => {
        setShowSuccess(true);
        setSaveButtonEnabled(false);
        const nextStepPath = nextStep();
        if (nextStepPath) {
          router.push(nextStepPath);
        }
      })
      .catch((error) => {
        setError("Something went wrong. Please try again later.");
      });
  };

  return (
    <PrivatePage>
      <>
        <Navbar page="My Profile" />
        {successSnackbar}
        {errorSnackbar}
        <Container component="main">
          <CssBaseline />
          <PageHeader
            showStepper={false}
            title="Complete your profile"
            icon={<IconAccountCircle />}
          />
          <Box
            sx={{
              maxWidth: 600,
              width: "auto",
              marginX: "auto",
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
            }}
          >
            <Box
              component="form"
              sx={{
                mt: 3,
                mx: 3,
              }}
            >
              <Grid container spacing={3}>
                <Grid item xs={12} sm={12}>
                  {/* Read-only */}
                  <TextField
                    label="Email"
                    fullWidth
                    disabled
                    value={address}
                  ></TextField>
                </Grid>
                <Grid item xs={12} sm={6}>
                  <TextField
                    label="First Name"
                    required
                    fullWidth
                    value={nameFirst}
                    onChange={(e) => {
                      changeNameFirst(e.target.value);
                      setSaveButtonEnabled(e.target.value !== "");
                    }}
                  ></TextField>
                </Grid>
                <Grid item xs={12} sm={6}>
                  <TextField
                    label="Last Name"
                    required
                    fullWidth
                    value={nameLast}
                    onChange={(e) => {
                      changeNameLast(e.target.value);
                      setSaveButtonEnabled(e.target.value !== "");
                    }}
                  ></TextField>
                </Grid>
                <Grid
                  item
                  xs={12}
                  sm={12}
                  sx={{
                    display: "flex",
                    justifyContent: "right",
                  }}
                >
                  <Button
                    disabled={!saveButtonEnabled}
                    onClick={() => submitProfileData(nameFirst, nameLast)}
                  >
                    Save
                  </Button>
                </Grid>
              </Grid>
            </Box>
          </Box>
        </Container>
      </>
    </PrivatePage>
  );
}

import Alert from "@mui/material/Alert";
import AlertTitle from "@mui/material/AlertTitle";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import ActionLink from "../action-link";

export default function BackupPrompt2() {
    return <Box
        sx={{
            marginTop: 3,
            display: "flex",
            flexDirection: "column",
            alignItems: "center",
        }}
    >
        <Alert
          severity="warning"
          variant="standard"
          sx={{
            maxWidth: 600,
          }}
        >
          <AlertTitle>Reminder: back up your inbox</AlertTitle>
          <Typography marginBottom={2} fontSize="1em">
          Before we clean up your inbox, if you haven't recently backed up your messages, now is the time to do so.
          </Typography>
          <ActionLink
            href="/article/backup"
            color="white"
            backgroundColor="#ed6c02"
            text="Learn how"
          />
        </Alert>
    </Box>
}
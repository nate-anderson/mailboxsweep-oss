import Alert from "@mui/material/Alert";
import AlertTitle from "@mui/material/AlertTitle";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import ActionLink from "../action-link";

export default function BackupPrompt1() {
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
            width: "100%",
            "& > .MuiAlert-message": {
              width: "100%",
            },
          }}
        >
          <AlertTitle>Now is a great time to back up your inbox.</AlertTitle>
          <Typography marginBottom={2} fontSize="1em">
            While we're organizing your inbox, we encourage you to back up your messages.
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
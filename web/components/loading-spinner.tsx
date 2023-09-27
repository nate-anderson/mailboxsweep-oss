import Box from "@mui/material/Box";
import CircularProgress from "@mui/material/CircularProgress";

export function LoadingSpinner() {
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
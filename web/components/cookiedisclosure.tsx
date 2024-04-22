import Snackbar from "@mui/material/Snackbar";
import { useEffect, useState } from "react";
import SnackbarContent from '@mui/material/SnackbarContent';
import Stack from '@mui/material/Stack';
import Button from "@mui/material/Button";
import Alert from "@mui/material/Alert";
import Fade from '@mui/material/Fade';
import Paper from '@mui/material/Paper';
import Typography from "@mui/material/Typography";
import Box from "@mui/material/Box";

const cookieDisclosureKey = "cookie_disclosure.dismissed";

export default function CookieDisclosure() {
    const [open, setOpen] = useState(false);
    useEffect(() => {
        const shouldHide = localStorage.getItem(cookieDisclosureKey) === "true";
        setOpen(!shouldHide);
    }, []);
    const closeBanner = () => {
        localStorage.setItem(cookieDisclosureKey, "true");
        setOpen(false);
    };

    return <Fade appear={false} in={open}>
      <Paper
        role="dialog"
        aria-modal="false"
        aria-label="Cookie banner"
        square
        variant="outlined"
        tabIndex={-1}
        sx={{
          position: 'fixed',
          bottom: 0,
          left: 0,
          right: 0,
          m: 0,
          p: 2,
          borderWidth: 0,
          borderTopWidth: 1,
          zIndex: 2,
        }}
      >
        <Stack
          direction={{ xs: 'row', sm: 'row' }}
          justifyContent="space-between"
          gap={1}
        >
          <Box
            sx={{
              flexShrink: 1,
              alignSelf: { xs: 'center', sm: 'center' },
            }}
          >
            <Typography variant="body2" sx={{
                paddingBottom: 0,
            }}>
            MailboxSweep uses cookies to improve user experience.
            </Typography>
          </Box>
          
        <Button
            size="small"
            onClick={closeBanner}
            sx={{
                margin: 0,
                padding: 0,
            }}
        >
            Accept
        </Button>
          
        </Stack>
      </Paper>
    </Fade>
}

/*

*/
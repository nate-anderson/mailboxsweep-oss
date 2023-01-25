import Snackbar from "@mui/material/Snackbar";
import Alert, { AlertColor } from "@mui/material/Alert";
import React, { useState } from "react";

export function useSnackbar(severity: AlertColor | undefined, autoHideDuration?: number): [ JSX.Element, (text: string) => void ] {
    const [ show, setShow ] = useState<boolean>(false);
    const [ displayText, setDisplayText ] = useState<string | undefined>();

    const snackbarElement = <Snackbar
        open={show}
        anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
        autoHideDuration={autoHideDuration ?? 5000}
        onClose={() => setShow(false)}
    >
        <Alert variant="filled" severity={severity}>
        {displayText}
        </Alert>
    </Snackbar>

    const setErrorMessage = (text: string): void => {
        setShow(true);
        setDisplayText(text);
    }

    return [ snackbarElement, setErrorMessage ];
}

export function useConstantSnackbar(displayText: string, severity: AlertColor | undefined, autoHideDuration?: number): [ JSX.Element, (show: boolean) => void ] {
    const [ show, setShow ] = useState<boolean>(false);

    const snackbarElement = <Snackbar
        open={show}
        anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
        autoHideDuration={autoHideDuration ?? 5000}
        onClose={() => setShow(false)}
    >
        <Alert variant="filled" severity={severity}>
        {displayText}
        </Alert>
    </Snackbar>

    return [ snackbarElement, setShow ];
}
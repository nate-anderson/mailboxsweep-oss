import AppBar from "@mui/material/AppBar";
import Toolbar from "@mui/material/Toolbar";
import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import IconArrowBack from "@mui/icons-material/ArrowBack";
import IconCheck from "@mui/icons-material/Check";

interface ConfirmBarProps {
    label: string;
    confirmButtonEnabled: boolean;
    onCheckEnabled: (enabled: boolean) => void;
    onSubmit: () => void;
}

export default function ConfirmBar(props: ConfirmBarProps) {
    return (
        <AppBar position="fixed" color="default" sx={{ top: "auto", bottom: 0 }}>
            <FormControlLabel
                sx={{
                    marginX: "auto",
                    paddingRight: 2,
                    marginTop: 1,
                }}
                label={
                    <Typography
                        fontSize="0.9em"
                    >
                        {props.label}
                    </Typography>
                }
                control={<Checkbox onChange={(_, enabled) => props.onCheckEnabled(enabled)}/>}
            />
            <Toolbar
                sx={{
                    display: "flex",
                    justifyContent: "space-between",
                    marginBottom: 1,
                }}
            >
                <Button
                    variant="contained"
                    startIcon={<IconArrowBack />}
                    sx={{
                        backgroundColor: "var(--color-accent)"
                    }}
                >Back</Button>
                <Button
                    disabled={!props.confirmButtonEnabled}
                    sx={{
                        backgroundColor: "var(--color-accent)"
                    }}
                    variant="contained"
                    endIcon={<IconCheck />}
                    onClick={() => props.onSubmit()}
                >Confirm</Button>
            </Toolbar>
        </AppBar>
    )
}
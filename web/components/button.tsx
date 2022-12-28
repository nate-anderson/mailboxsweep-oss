import { SxProps, Theme } from '@mui/material';
import MUIButton from '@mui/material/Button';

interface ButtonProps {
    children?: string;
    sx?: SxProps<Theme> | undefined;
    onClick: () => any
}

export default function Button(props: ButtonProps) {
    return (
        <MUIButton variant="contained" size="large" onClick={props.onClick} sx={{
            backgroundColor: "var(--color-accent)",
            color: "white",
            ...props.sx,
        }}>
            {props.children}
        </MUIButton>
    );
}
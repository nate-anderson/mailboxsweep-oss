import Typography from "@mui/material/Typography";

interface Props {
    fontSize?: string;
}

export default function LargeLogo(props: Props) {
    return (<Typography sx={{
        marginY: 3,
        fontFamily: "Lobster",
        fontSize: props.fontSize ?? "3.5em",
        color: "var(--color-primary)"
    }}>MailboxSweep</Typography>);
}
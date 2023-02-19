import Toolbar from "@mui/material/Toolbar";
import IconBack from "@mui/icons-material/ArrowBackIos";
import Typography from "@mui/material/Typography";
import IconButton from "@mui/material/IconButton";
import IconChevronRight from "@mui/icons-material/ChevronRight";

interface CrumbsProps {
    onClickBack: () => void;
    currentPath: string[];
    title?: string;
}

function BackHomeBar(props: CrumbsProps) {
    return <Toolbar
        sx={{
            maxWidth: 600,
            width: "100%",
            paddingX: 3,
        }}
        disableGutters
        >
            <IconButton
                color="primary"
                onClick={props.onClickBack}
                sx={{
                    marginRight: 0,
                }}
            ><IconBack /></IconButton>
            <Typography
                fontSize="1.2em"
                color="var(--color-gray)"
            >{props.title}</Typography>
        </Toolbar>
}

export default function Crumbs(props: CrumbsProps) {
    return props.currentPath.length == 0
        ? <Typography sx={{
            color: "var(--color-black)",
            fontSize: "2em",
            fontWeight: 100,
            marginBottom: 2,
            marginTop: 2,
        }}>Select Messages</Typography>
        : <BackHomeBar {...props} />
}
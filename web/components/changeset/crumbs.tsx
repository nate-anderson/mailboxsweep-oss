import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import IconList from "@mui/icons-material/ViewList";
import IconBack from "@mui/icons-material/ArrowBack";
import Link from "@mui/material/Link";

interface CrumbsProp {
    onClickReset: () => void;
    onClickGoUp?: () => void;
    goUpLabel?: string;
}

export default function Crumbs(props: CrumbsProp) {
    return <ButtonGroup variant="outlined">
        <Button
            onClick={props.onClickReset}
            startIcon={<IconList />}
        >
            Overview
        </Button>
        {
            props.onClickGoUp
                ? <Button
                    onClick={props.onClickGoUp}
                    startIcon={<IconBack />}
                >
                    Back
                </Button>
                : undefined
        }
        </ButtonGroup>;
}
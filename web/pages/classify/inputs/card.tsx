import { SelectedClassifier } from "../classifier";
import Card from "@mui/material/Card";
import ListItemButton from "@mui/material/ListItemButton";
import IconButton from "@mui/material/IconButton";
import ExpandLess from "@mui/icons-material/ExpandLess";
import ExpandMore from "@mui/icons-material/ExpandMore";
import Collapse from "@mui/material/Collapse";
import DragHandle from "@mui/icons-material/DragHandle";
import Typography from "@mui/material/Typography";
import { ReactNode } from "react";
import ListItemIcon from "@mui/material/ListItemIcon";
import Checkbox from "@mui/material/Checkbox";

interface Props {
    classifier: SelectedClassifier;
    expanded: boolean;
    toggleExpanded: () => void;
    children?: ReactNode;
    toggleEnabled: () => void;
}

export default function ClassifierCard(
    props: Props
) {
    return <Card
            sx={{
                marginBottom: 1,
            }}
        >
            <ListItemButton
                sx={{
                    paddingY: 2,
                }}
            >
                <DragHandle sx={{
                    color: "#a3a3a3"
                }}/>
                
                <ListItemIcon>
                    <Checkbox
                        checked={props.classifier.enabled}
                        onChange={() => props.toggleEnabled()}
                    />
                </ListItemIcon>
                {props.children}

                <IconButton onClick={() => props.toggleExpanded()}>
                    {props.expanded ? <ExpandLess /> : <ExpandMore />}
                </IconButton>
                
            </ListItemButton>
            <Collapse in={props.expanded}>
            <Typography
                sx={{
                    padding: 2,
                }}
            >
                {props.classifier.description}
            </Typography>
        </Collapse>
    </Card>;
        
};
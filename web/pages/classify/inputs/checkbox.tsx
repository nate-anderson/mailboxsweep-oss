import ClassifierCard from "./card";
import ListItemIcon from "@mui/material/ListItemIcon";
import Checkbox from "@mui/material/Checkbox";
import ListItemText from "@mui/material/ListItemText";
import { SelectedClassifier } from "../classifier";

interface Props {
    classifier: SelectedClassifier;
    index: number;
    toggleEnabled: () => void;
    expanded: boolean;
    toggleExpanded: () => void,
}

export default function CheckboxClassifierCard(props: Props) {
    return <ClassifierCard key={props.index} expanded={props.expanded} toggleExpanded={props.toggleExpanded} classifier={props.classifier} toggleEnabled={props.toggleEnabled}>
        <ListItemText
            sx={{
                paddingY: 2,
            }}
        >{props.classifier.name}</ListItemText>
    </ClassifierCard>
}
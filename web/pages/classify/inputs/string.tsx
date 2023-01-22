import { SelectedClassifier } from "../classifier";
import ClassifierCard from "./card";
import ListItemText from "@mui/material/ListItemText";
import TextField from "@mui/material/TextField";

interface Props {
    classifier: SelectedClassifier;
    index: number;
    toggleEnabled: () => void;
    expanded: boolean;
    toggleExpanded: () => void,
    setInput: (value: string | null) => void;
};

export default function StringClassifierCard(props: Props) {
    let value = props.classifier.initValue ?? "";

    return <ClassifierCard key={props.index} expanded={props.expanded} toggleExpanded={props.toggleExpanded} classifier={props.classifier} toggleEnabled={props.toggleEnabled}>
        <ListItemText
            sx={{
                paddingY: 2,
            }}
        >{props.classifier.name}</ListItemText>
        <TextField value={value} sx={{ maxWidth: "140px", marginRight: 1 }} onChange={(e) => props.setInput(e.target.value)} />
    </ClassifierCard>
}
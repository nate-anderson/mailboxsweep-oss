import { Dispatch, SetStateAction } from "react";
import { SelectedClassifier, dateFormat } from "../classifier";
import ClassifierCard from "./card";
import ListItemIcon from "@mui/material/ListItemIcon";
import Checkbox from "@mui/material/Checkbox";
import ListItemText from "@mui/material/ListItemText";
import { getYear, parse } from "date-fns";
import { MobileDatePicker } from "@mui/x-date-pickers/MobileDatePicker";
import { InputFormat } from "../../../client-gen";
import TextField from "@mui/material/TextField";

interface Props {
    classifier: SelectedClassifier;
    index: number;
    toggleEnabled: () => void;
    expanded: boolean;
    toggleExpanded: () => void,
    setInput: (value: Date | null) => void;
};

export default function DateClassifierCard(props: Props) {
    let value = new Date();
    if (props.classifier.initValue) {
        value = parse(props.classifier.initValue!.toString(), dateFormat, new Date())
    }

    return <ClassifierCard key={props.index} expanded={props.expanded} toggleExpanded={props.toggleExpanded} classifier={props.classifier} toggleEnabled={props.toggleEnabled}>
        <ListItemText
            sx={{
                paddingY: 2,
            }}
        >{props.classifier.name}</ListItemText>
        <MobileDatePicker
            value={value}
            onChange={props.setInput}
            label={props.classifier.inputLabel}
            renderInput={params => <TextField sx={{ maxWidth: "140px", marginRight: 1 }} {...params} />}
        />
    </ClassifierCard>
}
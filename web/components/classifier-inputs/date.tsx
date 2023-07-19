import { SelectedClassifier, dateFormat } from "../../lib/classifier";
import ClassifierCard from "./card";
import ListItemText from "@mui/material/ListItemText";
import { parse } from "date-fns";
import { MobileDatePicker } from "@mui/x-date-pickers/MobileDatePicker";
import TextField from "@mui/material/TextField";
import {
  DraggableProvidedDraggableProps,
  DraggableProvidedDragHandleProps,
} from "react-beautiful-dnd";

interface Props {
  classifier: SelectedClassifier;
  index: number;
  toggleEnabled: () => void;
  expanded: boolean;
  toggleExpanded: () => void;
  setInput: (value: Date | null) => void;
  draggableRef: (element: HTMLElement | null) => void;
  draggableProps: DraggableProvidedDraggableProps;
  dragHandleProps: DraggableProvidedDragHandleProps | null | undefined;
}

export default function DateClassifierCard(props: Props) {
  let value = new Date();
  if (props.classifier.initValue) {
    value = parse(
      props.classifier.initValue!.toString(),
      dateFormat,
      new Date()
    );
  }

  return (
    <ClassifierCard
      key={props.index}
      expanded={props.expanded}
      toggleExpanded={props.toggleExpanded}
      classifier={props.classifier}
      toggleEnabled={props.toggleEnabled}
      draggableRef={props.draggableRef}
      draggableProps={props.draggableProps}
      dragHandleProps={props.dragHandleProps}
    >
      <ListItemText
        sx={{

        }}
      >
        {props.classifier.name}
      </ListItemText>
      <MobileDatePicker
        value={value}
        onChange={props.setInput}
        label={props.classifier.inputLabel}
        inputFormat="MM/dd/yy"
        renderInput={(params: any) => (
          <TextField
            variant="standard"
            sx={{ maxWidth: "80px", marginRight: 0.5 }}
            {...params}
          />
        )}
      />
    </ClassifierCard>
  );
}

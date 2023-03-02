import { SelectedClassifier } from "../../lib/classifier";
import ClassifierCard from "./card";
import ListItemText from "@mui/material/ListItemText";
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
  setInput: (value: string | null) => void;
  draggableRef: (element: HTMLElement | null) => void;
  draggableProps: DraggableProvidedDraggableProps;
  dragHandleProps: DraggableProvidedDragHandleProps | null | undefined;
}

export default function StringClassifierCard(props: Props) {
  let value = props.classifier.initValue ?? "";

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
          paddingY: 2,
        }}
      >
        {props.classifier.name}
      </ListItemText>
      <TextField
        value={value}
        sx={{ maxWidth: "140px", marginRight: 1 }}
        onChange={(e) => props.setInput(e.target.value)}
      />
    </ClassifierCard>
  );
}

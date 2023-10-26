import ClassifierCard from "./card";
import ListItemText from "@mui/material/ListItemText";
import { SelectedClassifier } from "../../lib/classifier";
import {
  DraggableProvidedDraggableProps,
  DraggableProvidedDragHandleProps,
} from "react-beautiful-dnd";

interface Props {
  classifier: SelectedClassifier;
  index: number;
  remove: () => void;
  expanded: boolean;
  toggleExpanded: () => void;
  draggableRef: (element: HTMLElement | null) => void;
  draggableProps: DraggableProvidedDraggableProps;
  dragHandleProps: DraggableProvidedDragHandleProps | null | undefined;
  canRemove: boolean;
}

export default function CheckboxClassifierCard(props: Props) {
  return (
    <ClassifierCard
      key={props.index}
      expanded={props.expanded}
      toggleExpanded={props.toggleExpanded}
      classifier={props.classifier}
      remove={props.remove}
      draggableRef={props.draggableRef}
      draggableProps={props.draggableProps}
      dragHandleProps={props.dragHandleProps}
      canRemove={props.canRemove}
    >
      <ListItemText
        sx={{
          paddingY: 1,
        }}
      >
        {props.classifier.name}
      </ListItemText>
    </ClassifierCard>
  );
}

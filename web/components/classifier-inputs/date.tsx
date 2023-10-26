import { SelectedClassifier, dateFormat } from "../../lib/classifier";
import ClassifierCard from "./card";
import ListItemText from "@mui/material/ListItemText";
import { parse } from "date-fns";
import { Unstable_DateField as DateField } from "@mui/x-date-pickers/DateField";
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
  setInput: (value: Date | null) => void;
  draggableRef: (element: HTMLElement | null) => void;
  draggableProps: DraggableProvidedDraggableProps;
  dragHandleProps: DraggableProvidedDragHandleProps | null | undefined;
  canRemove: boolean;
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
      remove={props.remove}
      draggableRef={props.draggableRef}
      draggableProps={props.draggableProps}
      dragHandleProps={props.dragHandleProps}
      canRemove={props.canRemove}
    >
      <ListItemText
        sx={{

        }}
      >
        {props.classifier.name}
      </ListItemText>
      <DateField
        inputProps={{
          'data-testid': `classifier-date-input-${props.classifier.typeKey}`
        } as React.InputHTMLAttributes<HTMLInputElement>}
        value={value}
        onChange={props.setInput}
        label={props.classifier.inputLabel}
        format="MM/dd/yy"
        
        margin="dense"
        size="small"
        variant="standard"
        sx={{
          width: '100px'
        }}
      />
    </ClassifierCard>
  );
}

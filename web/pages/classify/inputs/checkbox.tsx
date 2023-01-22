import ClassifierCard from "./card";
import ListItemText from "@mui/material/ListItemText";
import { SelectedClassifier } from "../classifier";
import { DraggableProvidedDraggableProps, DraggableProvidedDragHandleProps } from "react-beautiful-dnd";

interface Props {
    classifier: SelectedClassifier;
    index: number;
    toggleEnabled: () => void;
    expanded: boolean;
    toggleExpanded: () => void;
    draggableRef: (element: HTMLElement | null) => void;
    draggableProps: DraggableProvidedDraggableProps;
    dragHandleProps: DraggableProvidedDragHandleProps | null | undefined;
}

export default function CheckboxClassifierCard(props: Props) {
    return <ClassifierCard
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
        >{props.classifier.name}</ListItemText>
    </ClassifierCard>
}
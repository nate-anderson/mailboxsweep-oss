import List from "@mui/material/List";
import CheckboxClassifierCard from "./checkbox";
import DateClassifierCard from "./date";
import StringClassifierCard from "./string";
import {
  DragDropContext,
  Droppable,
  Draggable,
  DropResult,
  ResponderProvided,
  DraggableProvidedDragHandleProps,
  DraggableProvidedDraggableProps,
} from "react-beautiful-dnd";
import { Dispatch, SetStateAction } from "react";
import { dateFormat, reorder, SelectedClassifier } from "../../lib/classifier";
import { format } from "date-fns";
import { InputFormat } from "../../client-gen";
import Box from "@mui/material/Box";

interface Props {
  classifiers: SelectedClassifier[];
  expandedIndex: number | undefined;
  setExpandedIndex: (i: number | undefined) => void;
  setSelectedClassifiers: Dispatch<SetStateAction<SelectedClassifier[]>>;
}

export default function SelectedClassifierList(props: Props) {
  const onDragEnd = (result: DropResult, provided: ResponderProvided) => {
    if (result.reason === "CANCEL") {
      return;
    }

    const i = Number(result.source.index);
    const j = Number(result.destination?.index);
    const updated = reorder(props.classifiers, i, j);
    props.setSelectedClassifiers(updated);
  };

  return (
    <Box
      minHeight={((88 + 4) * props.classifiers.length) + "px"}
      data-testid="classifier-drag-drop-region"
    >
      <DragDropContext onDragEnd={onDragEnd}>
        <Droppable droppableId="classifier-droppable-list">
          {(provided) => (
            <List ref={provided.innerRef} {...provided.droppableProps}>
              {props.classifiers.map((cl, i) => {
                const toggleExpanded = () =>
                  props.setExpandedIndex(
                    props.expandedIndex === i ? undefined : i
                  );

                const remove = () => {
                  props.setSelectedClassifiers((cls) =>
                    cls.filter((cl, j) => j !== i)
                  );
                  props.setExpandedIndex(undefined);
                };

                const setDateInput = (date: Date | null) => {
                  if (!date) {
                    return;
                  }
                  props.setSelectedClassifiers((cls) =>
                    cls.map((ecl, j) =>
                      j === i
                        ? { ...ecl, initValue: format(date, dateFormat) ?? "" }
                        : ecl
                    )
                  );
                };

                const setTextInput = (text: string | null) => {
                  if (text === null) {
                    return;
                  }
                  props.setSelectedClassifiers((cls) => 
                    cls.map((ecl, j) =>
                      j === i
                        ? { ...ecl, initValue: text}
                        : ecl
                    )
                  )
                }

                return (
                  <Draggable key={i} draggableId={i.toString()} index={i}>
                    {(provided, snapshot) =>
                      <ClassifierListItem
                        classifier={cl}
                        index={i}
                        remove={remove}
                        expandedIndex={props.expandedIndex}
                        toggleExpanded={toggleExpanded}
                        draggableRef={provided.innerRef}
                        draggableProps={provided.draggableProps}
                        dragHandleProps={provided.dragHandleProps}
                        setDateInput={setDateInput}
                        setTextInput={setTextInput}
                        selectedLength={props.classifiers.length}
                      />
                    }
                  </Draggable>
                );
              })}
            </List>
          )}
        </Droppable>
      </DragDropContext>
    </Box>
  );
}

function ClassifierListItem(props: {
  classifier: SelectedClassifier,
  index: number,
  remove: () => void,
  expandedIndex: number | undefined,
  toggleExpanded: () => void,
  draggableRef: (element: HTMLElement | null) => void,
  draggableProps: DraggableProvidedDraggableProps,
  dragHandleProps: DraggableProvidedDragHandleProps | null | undefined,
  setDateInput?: (date: Date | null) => void,
  setTextInput?: (text: string | null) => void,
  selectedLength: number,
}) {
  switch (props.classifier.inputFormat) {
    case undefined:
      return (
        <CheckboxClassifierCard
          key={props.index}
          classifier={props.classifier}
          index={props.index}
          draggableRef={props.draggableRef}
          draggableProps={props.draggableProps}
          dragHandleProps={props.dragHandleProps}
          remove={props.remove}
          toggleExpanded={props.toggleExpanded}
          expanded={props.index === props.expandedIndex}
          canRemove={props.selectedLength > 1}
        />
      );
    case InputFormat.Date:
      if (!props.setDateInput) {
        throw new Error(`setDateInput must be set for a date input classifier`);
      }
      return (
        <DateClassifierCard
          key={props.index}
          classifier={props.classifier}
          index={props.index}
          draggableRef={props.draggableRef}
          draggableProps={props.draggableProps}
          dragHandleProps={props.dragHandleProps}
          remove={props.remove}
          toggleExpanded={props.toggleExpanded}
          expanded={props.index === props.expandedIndex}
          setInput={props.setDateInput}
          canRemove={props.selectedLength > 1}
        />
      );
    case InputFormat.String:
      if (!props.setTextInput) {
        throw new Error(`setTextInput must be set for a text input classifier`);
      }
      return (
        <StringClassifierCard
        key={props.index}
          classifier={props.classifier}
          index={props.index}
          draggableRef={props.draggableRef}
          draggableProps={props.draggableProps}
          dragHandleProps={props.dragHandleProps}
          remove={props.remove}
          toggleExpanded={props.toggleExpanded}
          expanded={props.index === props.expandedIndex}
          setInput={props.setTextInput}
          canRemove={props.selectedLength > 1}
        />
      );
  }
};

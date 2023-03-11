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
      minHeight={((88 + 8) * props.classifiers.length) - 8 + "px"}
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

                const toggleEnabled = () => {
                  props.setSelectedClassifiers((cls) =>
                    cls.map((ecl, j) =>
                      j === i ? { ...ecl, enabled: !ecl.enabled } : ecl
                    )
                  );
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

                return (
                  <Draggable key={i} draggableId={i.toString()} index={i}>
                    {(provided, snapshot) =>
                      renderClassifierListItem(
                        cl,
                        i,
                        toggleEnabled,
                        props.expandedIndex,
                        toggleExpanded,
                        provided.innerRef,
                        provided.draggableProps,
                        provided.dragHandleProps,
                        setDateInput
                      )
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

const renderClassifierListItem = (
  classifier: SelectedClassifier,
  index: number,
  toggleEnabled: () => void,
  expandedIndex: number | undefined,
  toggleExpanded: () => void,
  draggableRef: (element: HTMLElement | null) => void,
  draggableProps: DraggableProvidedDraggableProps,
  dragHandleProps: DraggableProvidedDragHandleProps | null | undefined,
  setDateInput?: (date: Date | null) => void,
  setTextInput?: (text: string | null) => void
) => {
  switch (classifier.inputFormat) {
    case undefined:
      return (
        <CheckboxClassifierCard
          classifier={classifier}
          index={index}
          draggableRef={draggableRef}
          draggableProps={draggableProps}
          dragHandleProps={dragHandleProps}
          toggleEnabled={toggleEnabled}
          toggleExpanded={toggleExpanded}
          expanded={index === expandedIndex}
        />
      );
    case InputFormat.DATE:
      if (!setDateInput) {
        throw new Error(`setDateInput must be set for a date input classifier`);
      }
      return (
        <DateClassifierCard
          classifier={classifier}
          index={index}
          draggableRef={draggableRef}
          draggableProps={draggableProps}
          dragHandleProps={dragHandleProps}
          toggleEnabled={toggleEnabled}
          toggleExpanded={toggleExpanded}
          expanded={index === expandedIndex}
          setInput={setDateInput}
        />
      );
    case InputFormat.STRING:
      if (!setTextInput) {
        throw new Error(`setTextInput must be set for a text input classifier`);
      }
      return (
        <StringClassifierCard
          classifier={classifier}
          index={index}
          draggableRef={draggableRef}
          draggableProps={draggableProps}
          dragHandleProps={dragHandleProps}
          toggleEnabled={toggleEnabled}
          toggleExpanded={toggleExpanded}
          expanded={index === expandedIndex}
          setInput={setTextInput}
        />
      );
  }
};

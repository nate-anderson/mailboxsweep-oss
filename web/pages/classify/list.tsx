import List from "@mui/material/List";
import CheckboxClassifierCard from "./inputs/checkbox";
import DateClassifierCard from "./inputs/date";
import StringClassifierCard from "./inputs/string";
import {
    DragDropContext,
    Droppable,
    OnDragEndResponder,
  } from 'react-beautiful-dnd';
import { Dispatch, SetStateAction } from "react";
import { dateFormat, SelectedClassifier } from "./classifier";
import { format } from "date-fns";
import { InputFormat } from "../../client-gen";

interface Props {
    classifiers: SelectedClassifier[],
    expandedIndex: number | undefined,
    setExpandedIndex: (i: number | undefined) => void,
    setSelectedClassifiers: Dispatch<SetStateAction<SelectedClassifier[]>>
}

export default function SelectedClassifierList (props: Props) {

    return <List>
        {props.classifiers.map(
            (cl, i) => {
                const toggleExpanded = () => props.setExpandedIndex(props.expandedIndex === i ? undefined : i);

                const toggleEnabled = () => {
                    props.setSelectedClassifiers(
                        cls => cls.map(
                            (ecl, j) => j === i
                                ? { ...ecl, enabled: !ecl.enabled }
                                : ecl
                        )
                    );
                };

                const setDateInput = (date: Date | null) => {
                    if (!date) {
                        return;
                    }
                    props.setSelectedClassifiers(
                        cls => cls.map(
                            (ecl, j) => j === i
                                ? { ...ecl, initValue: format(date, dateFormat) ?? "" }
                                : ecl
                        )
                    );
                }

                return renderClassifierListItem(
                    cl,
                    i,
                    toggleEnabled,
                    props.expandedIndex,
                    toggleExpanded,
                    setDateInput,
                );
            }
        )}
    </List>;
}

const renderClassifierListItem = (
    classifier: SelectedClassifier,
    index: number,
    toggleEnabled: () => void,
    expandedIndex: number | undefined,
    toggleExpanded: () => void,
    setDateInput?: (date: Date | null) => void,
    setTextInput?: (text: string | null) => void,
) => {
    switch (classifier.inputFormat) {
        case undefined:
            return <CheckboxClassifierCard
                classifier={classifier}
                index={index}
                toggleEnabled={toggleEnabled}
                toggleExpanded={toggleExpanded}
                expanded={index === expandedIndex}
            />
        case InputFormat.DATE:
            if (!setDateInput) {
                throw new Error(`setDateInput must be set for a date input classifier`);
            }
            return <DateClassifierCard
                classifier={classifier}
                index={index}
                toggleEnabled={toggleEnabled}
                toggleExpanded={toggleExpanded}
                expanded={index === expandedIndex}
                setInput={setDateInput}
            />
        case InputFormat.STRING:
            if (!setTextInput) {
                throw new Error(`setTextInput must be set for a text input classifier`);
            }
            return <StringClassifierCard
                classifier={classifier}
                index={index}
                toggleEnabled={toggleEnabled}
                toggleExpanded={toggleExpanded}
                expanded={index === expandedIndex}
                setInput={setTextInput}
            />
    }
}
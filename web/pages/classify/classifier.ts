import { Classifier, InputFormat } from "../../client-gen";

export const dateFormat = "MM/dd/yyyy";

export type SelectedClassifier = Classifier & {
    name: string;
    description: string;
    hasInput: boolean;
    enabled: boolean;
    inputLabel?: string;
    inputFormat?: InputFormat;
};

export const reorder = <T>(
    array: T[],
    from: number,
    to: number
  ): T[] => {
    const updated = [ ...array ];
    const removed = updated.splice(from, 1)[0];
    updated.splice(to, 0, removed);
  
    return updated;
  };
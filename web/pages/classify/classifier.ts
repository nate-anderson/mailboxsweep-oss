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
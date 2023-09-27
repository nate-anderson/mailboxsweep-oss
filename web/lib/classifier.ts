import { Classifier, InputFormat } from "../client-gen";
import { getYear } from "date-fns";

export const dateFormat = "MM-dd-yyyy";

export type SelectedClassifier = Classifier & {
  name: string;
  description: string;
  hasInput: boolean;
  enabled: boolean;
  inputLabel?: string;
  inputFormat?: InputFormat;
  typeKey: string;
};

export const reorder = <T>(array: T[], from: number, to: number): T[] => {
  const updated = [...array];
  const removed = updated.splice(from, 1)[0];
  updated.splice(to, 0, removed);

  return updated;
};

const year = getYear(new Date());

export const defaultClassifiers: Classifier[] = [
  { typeKey: "SENDER_DOMAIN" },
  { typeKey: "SENDER_USERNAME" },
  // default to the first day of three years ago
  { typeKey: "SEND_DATE", initValue: `01-01-${year - 3}` },
];
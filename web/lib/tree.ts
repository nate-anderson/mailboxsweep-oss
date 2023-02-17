import { Classification } from "../client-gen";

export class ClassificationTree {
    readonly classification: Classification;

    constructor(classificationTree: Classification) {
        this.classification = classificationTree;
    }

    public jumpToPath(path: string[]): Classification {
        if (path.length == 0) {
            return this.classification;
        }

        let selected = this.classification;
        for (const step of path) {
            if (selected.children[step]) {
                selected = selected.children[step];
            } else {
                throw new Error(`path ${path} is invalid for this classification tree`)
            }
        }

        return selected;
    }

    public getPathLabels(path: string[]): string[] {
        if (path.length == 0) {
            return [];
        }

        const labels = [];
        let selected = this.classification;
        for (const step of path) {
            if (selected.children[step]) {
                labels.push(selected.children[step].label);
                selected = selected.children[step];
            } else {
                throw new Error(`path ${path} is invalid for this classification tree`)
            }
        }

        return labels;
    }
}
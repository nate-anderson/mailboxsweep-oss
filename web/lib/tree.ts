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
        throw new Error(`path ${path} is invalid for this classification tree`);
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
        throw new Error(`path ${path} is invalid for this classification tree`);
      }
    }

    return labels;
  }

  // return full paths to all leaves under the provided path
  public getLeaves(
    classification: Classification,
    pathPrefix: string[]
  ): string[][] {
    const results: string[][] = [];

    // if there are no children, this is the leaf
    if (Object.keys(classification.children).length == 0) {
      return [pathPrefix];
    }

    for (const [key, child] of Object.entries(classification.children)) {
      results.push(...this.getLeaves(child, [...pathPrefix, key]));
    }

    return results;
  }

  public getLeavesOfPath(path: string[]): string[][] {
    const selected = this.jumpToPath(path);
    return this.getLeaves(selected, path);
  }
}

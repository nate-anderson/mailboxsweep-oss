export class Selection {
  private internal: Set<string> = new Set<string>();

  // for optimistic UI, allow removing a part of the selection tree
  // without actually dropping the leaf nodes from memory
  // this way, if an API call fails, we still have the correct state
  // to roll back to
  private optimisticExcludes: Set<string> = new Set<string>();

  constructor(paths: string[][]) {
    paths.map((p) => JSON.stringify(p)).forEach((p) => this.internal.add(p));
  }

  public add(path: string[]): Selection {
    this.internal.add(JSON.stringify(path));

    // returning a new object means the useState setter will cause a rerender
    const n = new Selection([]);
    n.internal = this.internal;
    n.optimisticExcludes = this.optimisticExcludes;
    return n;
  }

  public remove(path: string[]): Selection {
    this.internal.delete(JSON.stringify(path));
    const n = new Selection([]);
    n.internal = this.internal;
    n.optimisticExcludes = this.optimisticExcludes;
    return n;
  }

  public has(path: string[]): boolean {
    const serialized = JSON.stringify(path);
    // if excluded by prefix, return false
    let excludedByPrefix = false;
    this.optimisticExcludes.forEach((prefix) => {
      if (serialized.startsWith(prefix)) {
        excludedByPrefix = true;
      }
    });
    return !excludedByPrefix && this.internal.has(JSON.stringify(path));
  }

  public hasAny(paths: string[][]): boolean {
    return paths
      .some((p) => this.has(p));
  }

  public hasAll(paths: string[][]): boolean {
    return paths
      .every((p) => this.has(p));
  }

  // treats path as a prefix and excludes any stored leaf paths that begin with that prefix,
  // until clearOptimisticExclude is called on the same path
  public optimisticExclude(path: string[]): Selection {
    const prefix = JSON.stringify(path).slice(0, -1);
    this.optimisticExcludes.add(prefix);
    const n = new Selection([]);
    n.internal = this.internal;
    n.optimisticExcludes = this.optimisticExcludes;
    return n;
  }

  public clearOptimisticExclude(path: string[]): Selection {
    const prefix = JSON.stringify(path).slice(0, -1);
    this.optimisticExcludes.delete(prefix);
    const n = new Selection([]);
    n.internal = this.internal;
    n.optimisticExcludes = this.optimisticExcludes;
    return n;
  }
}

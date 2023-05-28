export class Selection {
  private internal: Set<string> = new Set<string>();

  constructor(paths: string[][]) {
    paths.map((p) => JSON.stringify(p)).forEach((p) => this.internal.add(p));
  }

  public add(path: string[]): Selection {
    this.internal.add(JSON.stringify(path));

    // returning a new object means the useState setter will cause a rerender
    const n = new Selection([]);
    n.internal = this.internal;
    return n;
  }

  public remove(path: string[]): Selection {
    this.internal.delete(JSON.stringify(path));
    const n = new Selection([]);
    n.internal = this.internal;
    return n;
  }

  public has(path: string[]): boolean {
    return this.internal.has(JSON.stringify(path));
  }

  public hasAny(paths: string[][]): boolean {
    return paths
      .map((p) => JSON.stringify(p))
      .some((p) => this.internal.has(p));
  }

  public hasAll(paths: string[][]): boolean {
    return paths
      .map((p) => JSON.stringify(p))
      .every((p) => this.internal.has(p));
  }
}

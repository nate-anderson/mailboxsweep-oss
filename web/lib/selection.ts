export class Selection {

    private internal: Set<string> = new Set<string>();
    private _count: number = 0;

    constructor(paths: string[][]) {
        paths.map(p => JSON.stringify(p)).forEach(p => this.internal.add(p));
    }

    public add(path: string[], n: number): Selection {
        this.internal.add(JSON.stringify(path));
        this._count += n;
        return this;
    }

    public remove(path: string[], n: number): Selection {
        this.internal.delete(JSON.stringify(path));
        this._count += n;
        return this;
    }

    public count(): number {
        return this._count;
    }

    public has(path: string[]): boolean {
        return this.internal.has(JSON.stringify(path));
    }

    public hasAny(paths: string[][]): boolean {
        return paths
            .map(p => JSON.stringify(p))
            .some(p => this.internal.has(p));
    }

}
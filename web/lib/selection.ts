export class Selection {

    private internal: Set<string> = new Set<string>();

    constructor(paths: string[][]) {
        paths.map(p => JSON.stringify(p)).forEach(p => this.internal.add(p));
    }

    public add(path: string[]): Selection {
        this.internal.add(JSON.stringify(path));
        return this;
    }

    public remove(path: string[]): Selection {
        this.internal.delete(JSON.stringify(path));
        return this;
    }

    public has(path: string[]): boolean {
        return this.internal.has(JSON.stringify(path));
    }

    public hasAny(paths: string[][]): boolean {
        return paths
            .map(p => JSON.stringify(p))
            .some(p => this.internal.has(p));
    }

    public hasAll(paths: string[][]): boolean {
        return paths
            .map(p => JSON.stringify(p))
            .every(p => this.internal.has(p));
    }

}
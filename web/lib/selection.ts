export class Selection {

    private internal: Set<string> = new Set<string>();
    private _count: number = 0;

    public add(path: string, n: number): Selection {
        this.internal.add(JSON.stringify(path));
        this._count += n;
        return this;
    }

    public remove(path: string, n: number): Selection {
        this.internal.delete(JSON.stringify(path));
        this._count += n;
        return this;
    }

    public count(): number {
        return this._count;
    }

}
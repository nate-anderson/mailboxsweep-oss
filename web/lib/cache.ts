export async function asyncCache<T>(getter: () => Promise<T>, key: string): Promise<T> {
    console.log("Checking cache key", key);
    const cached = localStorage.getItem(key);
    if (cached) {
        console.log("Returning cached", cached, typeof cached);
        return JSON.parse(cached) as T;
    }

    const fresh = await getter();
    localStorage.setItem(key, JSON.stringify(fresh));
    console.log("Returning fresh");
    return fresh as T;
}

export function invalidate(key: string) {
    localStorage.removeItem(key);
}

export function clearAll() {
    localStorage.clear();
}
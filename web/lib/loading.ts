import { useState } from "react";

interface ILoadingState<ReqT, ResT> {
    invoke: (req: ReqT) => Promise<ResT>;
    isLoading: boolean;
}

export function useLoadingState<ReqT, ResT>(fn: (req: ReqT) => Promise<ResT>): ILoadingState<ReqT, ResT> {
    const [isLoading, setIsLoading] = useState(false);

    const invoke = async (req: ReqT): Promise<ResT> => {
        setIsLoading(true);
        try {
            const res = await fn(req);
            return res;
        } catch (error) {
            throw error;
        } finally {
            setIsLoading(false);
        }
    }
    
    return {invoke, isLoading};
}
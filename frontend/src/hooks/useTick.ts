import { useSyncExternalStore } from "react";

const listeners = new Set<() => void>();
let intervalId: ReturnType<typeof setInterval> | null = null;

function subscribe(callback: () => void) {
    listeners.add(callback);
    if (intervalId === null) {
        intervalId = setInterval(() => {
            listeners.forEach((l) => l());
        }, 1000);
    }
    return () => {
        listeners.delete(callback);
        if (listeners.size === 0 && intervalId !== null) {
            clearInterval(intervalId);
            intervalId = null;
        }
    };
}

const getSnapshot = () => Math.floor(Date.now() / 1000);

// One interval shared by every consumer, started lazily, torn down when nobody's listening
export function useTick() {
    return useSyncExternalStore(subscribe, getSnapshot);
}

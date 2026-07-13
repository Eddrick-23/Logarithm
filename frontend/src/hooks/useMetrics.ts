import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
    fetchTopServiceErrorsStats,
    fetchIngestionGraphMetrics,
    fetchErrorRateMetrics,
    fetchStorageInfoMetrics,
    fetchLogRateStats,
    fetchNatsQueueDepthGraphMetrics,
} from "../api/metricsApi";
import { useCallback, useEffect, useRef, useState } from "react";

const QUERY_KEYS = {
    ingestionGraphMetrics: "ingestionGraphMetrics",
    logRateStats: "logRateStats",
    topServiceErrorsStats: "topServiceErrorsStats",
    errorRateMetrics: "errorRateMetrics",
    storageInfoMetrics: "storageInfoMetrics",
    natsQueueDepthGraphMetrics: "natsQueueDepthGraphMetrics",
};

const STALE_THRESHOLD_MS = 15000; // 15s stale time
const EVENT_MAP = [
    { event: "ingestion-graph-metrics", queryKey: QUERY_KEYS.ingestionGraphMetrics },
    { event: "log-rate-stats", queryKey: QUERY_KEYS.logRateStats },
    { event: "top-service-errors", queryKey: QUERY_KEYS.topServiceErrorsStats },
    { event: "error-rate", queryKey: QUERY_KEYS.errorRateMetrics },
    { event: "storage-info", queryKey: QUERY_KEYS.storageInfoMetrics },
    { event: "nats-queue-depth", queryKey: QUERY_KEYS.natsQueueDepthGraphMetrics },
]; // stores a map which contains event name and TanStack query key

export const useDashboard = () => {
    const queryClient = useQueryClient();
    const eventSourceRef = useRef<EventSource | null>(null);
    const lastMessageRef = useRef<number>(Date.now());
    const [isLoading, setIsLoading] = useState<boolean>(false);
    const [isError, setIsError] = useState<boolean>(false);

    const openStream = useCallback(() => {
        // close existing connection before opening new conneciton
        eventSourceRef.current?.close();
        lastMessageRef.current = Date.now(); // reset watchdog clock on (re)connect

        const eventSource = new EventSource("/api/dashboard/stream");

        for (const { event, queryKey } of EVENT_MAP) {
            eventSource.addEventListener(event, (e) => {
                const data = JSON.parse(e.data);
                queryClient.setQueryData([queryKey], data);
                setIsError(false);
                lastMessageRef.current = Date.now();
            });
        }

        eventSource.onerror = () => {
            if (eventSourceRef.current?.readyState !== EventSource.OPEN) {
                setIsError(true);
            }
        };

        eventSource.onopen = () => {
            lastMessageRef.current = Date.now(); // update watchdog clock
            setIsError(false);
        };

        eventSourceRef.current = eventSource;
    }, [queryClient]);

    const connect = useCallback(async () => {
        setIsLoading(true);
        try {
            // manually fetch all of the metrics to load the dashboard upon connect
            const [
                ingestionGraphMetrics,
                storageInfoMetrics,
                logRateStats,
                topServiceErrorsStats,
                errorRateMetrics,
                natsQueueDepthGraphMetrics,
            ] = await Promise.all([
                fetchIngestionGraphMetrics(),
                fetchStorageInfoMetrics(),
                fetchLogRateStats(),
                fetchTopServiceErrorsStats(),
                fetchErrorRateMetrics(),
                fetchNatsQueueDepthGraphMetrics(),
            ]);

            queryClient.setQueryData([QUERY_KEYS.ingestionGraphMetrics], ingestionGraphMetrics);
            queryClient.setQueryData([QUERY_KEYS.storageInfoMetrics], storageInfoMetrics);
            queryClient.setQueryData([QUERY_KEYS.logRateStats], logRateStats);
            queryClient.setQueryData([QUERY_KEYS.topServiceErrorsStats], topServiceErrorsStats);
            queryClient.setQueryData([QUERY_KEYS.errorRateMetrics], errorRateMetrics);
            queryClient.setQueryData([QUERY_KEYS.natsQueueDepthGraphMetrics], natsQueueDepthGraphMetrics);

            setIsError(false);
        } catch {
            setIsError(true);
        } finally {
            setIsLoading(false);
        }
        openStream();
    }, [queryClient, openStream]);

    useEffect(() => {
        // open connection on mount, close on dismount
        connect();

        // watchdog: if no message received within threshold, assume connection is dead
        const interval = setInterval(() => {
            if (Date.now() - lastMessageRef.current > STALE_THRESHOLD_MS) {
                setIsError(true);
            }
        }, 5000);

        return () => {
            eventSourceRef.current?.close();
            clearInterval(interval);
        };
    }, [connect]);

    return { isLoading, isError, refetch: connect };
};

export const useIngestionGraphMetrics = () => {
    return useQuery({
        queryKey: [QUERY_KEYS.ingestionGraphMetrics],
        queryFn: fetchIngestionGraphMetrics,
        staleTime: Infinity, // SSE keeps it fresh, no need for TanStack Query to refetch
        refetchInterval: false,
        enabled: false, // never auto-fetch since connect() seeds the data manually
    });
};

export const useLogRateStats = () => {
    return useQuery({
        queryKey: [QUERY_KEYS.logRateStats],
        queryFn: fetchLogRateStats,
        staleTime: Infinity,
        refetchInterval: false,
        enabled: false, // never auto-fetch since connect() seeds the data manually
    });
};

export const useTopServiceErrorsStats = () => {
    return useQuery({
        queryKey: [QUERY_KEYS.topServiceErrorsStats],
        queryFn: fetchTopServiceErrorsStats,
        staleTime: Infinity,
        refetchInterval: false,
        enabled: false, // never auto-fetch since connect() seeds the data manually
    });
};

export const useErrorRateMetrics = () => {
    return useQuery({
        queryKey: [QUERY_KEYS.errorRateMetrics],
        queryFn: fetchErrorRateMetrics,
        staleTime: Infinity,
        refetchInterval: false,
        enabled: false, // never auto-fetch since connect() seeds the data manually
    });
};

export const useStorageInfoMetrics = () => {
    return useQuery({
        queryKey: [QUERY_KEYS.storageInfoMetrics],
        queryFn: fetchStorageInfoMetrics,
        staleTime: Infinity,
        refetchInterval: false,
        enabled: false, // never auto-fetch since connect() seeds the data manually
    });
};

export const useNatsQueueDepthGraphMetrics = () => {
    return useQuery({
        queryKey: [QUERY_KEYS.natsQueueDepthGraphMetrics],
        queryFn: fetchNatsQueueDepthGraphMetrics,
        staleTime: Infinity,
        refetchInterval: false,
        enabled: false, // never auto-fetch since connect() seeds the data manually
    });
};

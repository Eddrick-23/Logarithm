import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
    fetchTopServiceErrorsStats,
    fetchIngestionGraphMetrics,
    fetchErrorRateMetrics,
    fetchStorageInfoMetrics,
    fetchLogRateStats,
} from "../api/metricsApi";
import { useCallback, useEffect, useRef, useState } from "react";
import type { IngestionGraphData } from "../types/Metric";

const QUERY_KEYS = {
    ingestionGraphMetrics: "ingestionGraphMetrics",
    logRateStats: "logRateStats",
    topServiceErrorsStats: "topServiceErrorsStats",
    errorRateMetrics: "errorRateMetrics",
    storageInfoMetrics: "storageInfoMetrics",
    dashboardConnectionError: "dashboardConnectionError",
};

const MAX_POINTS = 60;
const STALE_THRESHOLD_MS = 15000; // 15s stale time
const EVENT_MAP = [
    { event: "ingestion-graph-metrics", queryKey: QUERY_KEYS.ingestionGraphMetrics },
    { event: "log-rate-stats", queryKey: QUERY_KEYS.logRateStats },
    { event: "top-service-errors", queryKey: QUERY_KEYS.topServiceErrorsStats },
    { event: "error-rate", queryKey: QUERY_KEYS.errorRateMetrics },
    { event: "storage-info", queryKey: QUERY_KEYS.storageInfoMetrics },
]; // stores a map which contains event name and TanStack query key

export const useDashboard = () => {
    const queryClient = useQueryClient();
    const eventSourceRef = useRef<EventSource | null>(null);
    const lastMessageRef = useRef<number>(Date.now());
    const [isLoading, setIsLoading] = useState<boolean>(false);

    const { data: connectionError } = useQuery({
        queryKey: [QUERY_KEYS.dashboardConnectionError],
        queryFn: () => false,
        initialData: false,
        staleTime: Infinity,
    });

    const openStream = useCallback(() => {
        // close existing connection before opening new conneciton
        eventSourceRef.current?.close();
        lastMessageRef.current = Date.now(); // reset watchdog clock on (re)connect

        const eventSource = new EventSource("/api/dashboard/stream");

        for (const { event, queryKey } of EVENT_MAP) {
            eventSource.addEventListener(event, (e) => {
                if (event === "ingestion-graph-metrics") {
                    const liveDelta: IngestionGraphData = JSON.parse(e.data);

                    queryClient.setQueryData([queryKey], (oldData: IngestionGraphData | undefined) => {
                        if (!oldData) return liveDelta;

                        const nextTimestamps = [...oldData.timestamps, ...liveDelta.timestamps].slice(-MAX_POINTS);
                        const nextMetrics: Record<string, { logsCount: number }[]> = {};
                        const allServices = Array.from(
                            new Set([...Object.keys(oldData.metrics), ...Object.keys(liveDelta.metrics)]),
                        );

                        allServices.forEach((service) => {
                            const oldVals = oldData.metrics[service] || [];
                            const deltaVals =
                                liveDelta.metrics[service] ?? Array(liveDelta.timestamps.length).fill({ logsCount: 0 });
                            nextMetrics[service] = [...oldVals, ...deltaVals].slice(-MAX_POINTS);
                        });

                        return { timestamps: nextTimestamps, metrics: nextMetrics };
                    });
                } else {
                    const data = JSON.parse(e.data);
                    queryClient.setQueryData([queryKey], data);
                }

                queryClient.setQueryData([QUERY_KEYS.dashboardConnectionError], false);
                lastMessageRef.current = Date.now();
            });
        }

        eventSource.onerror = () => {
            if (eventSourceRef.current?.readyState !== EventSource.OPEN) {
                queryClient.setQueryData([QUERY_KEYS.dashboardConnectionError], true);
            }
        };

        eventSource.onopen = () => {
            lastMessageRef.current = Date.now(); // update watchdog clock
            queryClient.setQueryData([QUERY_KEYS.dashboardConnectionError], false);
        };

        eventSourceRef.current = eventSource;
    }, [queryClient]);

    const connect = useCallback(async () => {
        try {
            queryClient.setQueryData([QUERY_KEYS.dashboardConnectionError], false);
        } catch {
            queryClient.setQueryData([QUERY_KEYS.dashboardConnectionError], true);
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
                queryClient.setQueryData([QUERY_KEYS.dashboardConnectionError], true);
            }
        }, 5000);

        return () => {
            eventSourceRef.current?.close();
            clearInterval(interval);
        };
    }, [connect]);

    return { isLoading, isError: connectionError, refetch: connect };
};

export const useIngestionGraphMetrics = () => {
    return useQuery({
        queryKey: [QUERY_KEYS.ingestionGraphMetrics],
        queryFn: fetchIngestionGraphMetrics,
        staleTime: Infinity, // SSE keeps it fresh, no need for TanStack Query to refetch
        refetchInterval: false,
    });
};

export const useLogRateStats = () => {
    return useQuery({
        queryKey: [QUERY_KEYS.logRateStats],
        queryFn: fetchLogRateStats,
        staleTime: Infinity,
        refetchInterval: false,
    });
};

export const useTopServiceErrorsStats = () => {
    return useQuery({
        queryKey: [QUERY_KEYS.topServiceErrorsStats],
        queryFn: fetchTopServiceErrorsStats,
        staleTime: Infinity,
        refetchInterval: false,
    });
};

export const useErrorRateMetrics = () => {
    return useQuery({
        queryKey: [QUERY_KEYS.errorRateMetrics],
        queryFn: fetchErrorRateMetrics,
        staleTime: Infinity,
        refetchInterval: false,
    });
};

export const useStorageInfoMetrics = () => {
    return useQuery({
        queryKey: [QUERY_KEYS.storageInfoMetrics],
        queryFn: fetchStorageInfoMetrics,
        staleTime: Infinity,
        refetchInterval: false,
    });
};

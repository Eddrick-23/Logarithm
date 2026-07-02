import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
    fetchTopServiceErrorsStats,
    fetchIngestionMetrics,
    fetchErrorRateMetrics,
    fetchStorageInfoMetrics,
    fetchLogRateStats,
} from "../api/metricsApi";
import { useCallback, useEffect, useRef } from "react";
import type { IngestionGraphData } from "../types/Metric";

const MAX_POINTS = 60;
const STALE_THRESHOLD_MS = 15000; // 15s stale time
const EVENT_MAP = [
    { event: "ingestion-graph-metrics", queryKey: "ingestionMetrics" },
    { event: "log-rate-stats", queryKey: "logRateStats" },
    { event: "top-service-errors", queryKey: "topServiceErrorsStats" },
    { event: "error-rate", queryKey: "errorRateMetrics" },
    { event: "storage-info", queryKey: "storageInfoMetrics" },
]; // stores a map which contains event name and TanStack query key

export const useIngestionMetrics = () => {
    const queryClient = useQueryClient();
    const eventSourceRef = useRef<EventSource | null>(null);
    const lastMessageRef = useRef<number>(Date.now());

    const query = useQuery({
        queryKey: ["ingestionMetrics"],
        queryFn: fetchIngestionMetrics,
        staleTime: Infinity, // SSE keeps it fresh, no need for TanStack Query to refetch
        refetchInterval: false,
    });

    const { data: connectionError } = useQuery({
        queryKey: ["ingestionMetricsConnectionError"],
        queryFn: () => false,
        initialData: false,
        staleTime: Infinity,
    });

    const openStream = useCallback(() => {
        // close existing connection before opening new conneciton
        eventSourceRef.current?.close();
        lastMessageRef.current = Date.now(); // reset watchdog clock on (re)connect

        const eventSource = new EventSource("/api/ingestion-metrics/stream");

        for (const { event, queryKey } of EVENT_MAP) {
            if (event === "ingestion-graph-metrics") {
                eventSource.addEventListener(event, (e) => {
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

                    queryClient.setQueryData(["ingestionMetricsConnectionError"], false);
                    lastMessageRef.current = Date.now();
                });
            } else {
                eventSource.addEventListener(event, (e) => {
                    const data = JSON.parse(e.data);
                    queryClient.setQueryData([queryKey], data);
                    queryClient.setQueryData(["ingestionMetricsConnectionError"], false);
                    lastMessageRef.current = Date.now();
                });
            }
        }

        eventSource.onerror = () => {
            if (eventSourceRef.current?.readyState !== EventSource.OPEN) {
                queryClient.setQueryData(["ingestionMetricsConnectionError"], true);
            }
        };

        eventSource.onopen = () => {
            lastMessageRef.current = Date.now(); // update watchdog clock
            queryClient.setQueryData(["ingestionMetricsConnectionError"], false);
        };

        eventSourceRef.current = eventSource;
    }, [queryClient]);

    const connect = useCallback(async () => {
        // do REST fetch to pre load the data then (re)open SSE for live connection
        try {
            const data = await fetchIngestionMetrics();
            queryClient.setQueryData(["ingestionMetrics"], data);
            queryClient.setQueryData(["ingestionMetricsConnectionError"], false);
        } catch {
            queryClient.setQueryData(["ingestionMetricsConnectionError"], true);
        }
        openStream();
    }, [queryClient, openStream]);

    useEffect(() => {
        // open connection on mount, close on dismount
        connect();

        // watchdog: if no message received within threshold, assume connection is dead
        const interval = setInterval(() => {
            if (Date.now() - lastMessageRef.current > STALE_THRESHOLD_MS) {
                queryClient.setQueryData(["ingestionMetricsConnectionError"], true);
            }
        }, 5000);

        return () => {
            eventSourceRef.current?.close();
            clearInterval(interval);
        };
    }, [connect]);

    return {
        ...query,
        isError: query.isError || connectionError,
        refetch: connect,
    };
};

export const useLogRateStats = () => {
    return useQuery({
        queryKey: ["logRateStats"],
        queryFn: fetchLogRateStats,
        staleTime: Infinity,
        refetchInterval: false,
    });
};

export const useTopServiceErrorsStats = () => {
    return useQuery({
        queryKey: ["topServiceErrorsStats"],
        queryFn: fetchTopServiceErrorsStats,
        staleTime: Infinity,
        refetchInterval: false,
    });
};

export const useErrorRateMetrics = () => {
    return useQuery({
        queryKey: ["errorRateMetrics"],
        queryFn: fetchErrorRateMetrics,
        staleTime: Infinity,
        refetchInterval: false,
    });
};

export const useStorageInfoMetrics = () => {
    return useQuery({
        queryKey: ["storageInfoMetrics"],
        queryFn: fetchStorageInfoMetrics,
        staleTime: Infinity,
        refetchInterval: false,
    });
};

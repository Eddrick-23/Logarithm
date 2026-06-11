import { useQuery } from "@tanstack/react-query";
import { fetchMetrics } from "../api/metricsApi";

export const useMetrics = () => {
    return useQuery({
        queryKey: ["metrics"],
        queryFn: fetchMetrics,
        staleTime: 1000 * 60, // 1 minute
        refetchInterval: 1000 * 5, // refetch every 5 seconds
    });
};

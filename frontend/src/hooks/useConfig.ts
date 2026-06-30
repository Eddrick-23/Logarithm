import { useQuery } from "@tanstack/react-query";
import { fetchConfig } from "../api/configApi";

export const useConfig = () => {
    return useQuery({
        queryKey: ["config"],
        queryFn: fetchConfig,
        staleTime: Infinity, // staleTime is infinity since config will only be changed when Logarithm restarts
    });
};

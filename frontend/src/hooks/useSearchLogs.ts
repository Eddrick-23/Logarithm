import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { searchLogs, type SearchLogsParams } from "../api/logsApi";

export const useSearchLogs = (params: SearchLogsParams) => {
    return useQuery({
        queryKey: ["logs-search", params],
        queryFn: () => searchLogs(params),
        placeholderData: keepPreviousData,
    });
};

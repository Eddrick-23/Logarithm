import type { MRT_ColumnFiltersState, MRT_PaginationState, MRT_SortingState } from "material-react-table";
import { capitaliseFirstLetter } from "../utils/utils";
import type { LogRecord } from "../types/Log";
import axios from "axios";
import dayjs from "dayjs";

export type LogApiResponse = {
    data: Array<LogRecord>;
    meta: {
        totalRowCount: number;
    };
};

export interface SearchLogsParams {
    columnFilters: MRT_ColumnFiltersState;
    pagination: MRT_PaginationState;
    sorting: MRT_SortingState;
}

// helper to get a column filter value by id
function getFilter(filters: MRT_ColumnFiltersState, id: string): string {
    return (filters.find((f) => f.id === id)?.value as string) ?? "";
}

export const searchLogs = async ({ columnFilters, pagination, sorting }: SearchLogsParams): Promise<LogApiResponse> => {
    const sortField = capitaliseFirstLetter(sorting[0]?.id) ?? "Timestamp";
    const descending = sorting[0]?.desc ?? true;

    // default time range: from Unix epoch time to current datetime
    const timeFilter = columnFilters.find((f) => f.id === "startTime")?.value as
        | [dayjs.Dayjs | null, dayjs.Dayjs | null]
        | undefined;
    const endTime = timeFilter?.[1] ? timeFilter[1].toDate() : new Date();
    const startTime = timeFilter?.[0] ? timeFilter[0].toDate() : new Date(0);

    const fmt = (d: Date) => d.toISOString().slice(0, 19); // Example: "2006-01-02T15:04:05"

    const response = await axios.get<LogApiResponse>("/api/search", {
        params: {
            startTime: fmt(startTime),
            endTime: fmt(endTime),
            body: getFilter(columnFilters, "body"),
            serviceName: getFilter(columnFilters, "serviceName"),
            severityNumber: getFilter(columnFilters, "severityNumber"),
            severityText: getFilter(columnFilters, "severityText"),
            traceId: getFilter(columnFilters, "traceId"),
            spanId: getFilter(columnFilters, "spanId"),
            orderBy: sortField,
            descending: descending,
            limit: pagination.pageSize,
            offset: pagination.pageIndex * pagination.pageSize,
        },
    });

    return response.data;
};

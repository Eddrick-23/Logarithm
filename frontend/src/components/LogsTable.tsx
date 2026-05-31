import { useMemo, useState } from "react";
import {
    MaterialReactTable,
    useMaterialReactTable,
    type MRT_ColumnDef,
    type MRT_ColumnFiltersState,
    type MRT_PaginationState,
    type MRT_SortingState,
} from "material-react-table";
import { IconButton, Tooltip } from "@mui/material";
import RefreshIcon from "@mui/icons-material/Refresh";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import type { KeyValue, LogRecord } from "../types/LogRecord";
import { LocalizationProvider } from "@mui/x-date-pickers/LocalizationProvider";
import { AdapterDayjs } from "@mui/x-date-pickers/AdapterDayjs";
import axios from "axios";
import { capitaliseFirstLetter } from "../utils/utils";
import dayjs from "dayjs";

type UserApiResponse = {
    data: Array<LogRecord>;
    meta: {
        totalRowCount: number;
    };
};

// helper to get a column filter value by id
function getFilter(filters: MRT_ColumnFiltersState, id: string): string {
    return (filters.find((f) => f.id === id)?.value as string) ?? "";
}

export default function EnhancedTable() {
    // manage our own state for stuff we want to pass to the API
    const [columnFilters, setColumnFilters] = useState<MRT_ColumnFiltersState>([]);
    const [sorting, setSorting] = useState<MRT_SortingState>([]);
    const [pagination, setPagination] = useState<MRT_PaginationState>({
        pageIndex: 0,
        pageSize: 10,
    });

    // consider storing this code in a custom hook (i.e useFetchUsers)
    const {
        data: { data = [], meta } = {}, // your data and api response will probably be different
        isError,
        isRefetching,
        isLoading,
        refetch,
    } = useQuery<UserApiResponse>({
        queryKey: [
            "logs-search",
            {
                columnFilters, // refetch when columnFilters changes
                pagination, // refetch when pagination changes
                sorting, // refetch when sorting changes
            },
        ],
        queryFn: async () => {
            // derive params from table state
            const sortField = capitaliseFirstLetter(sorting[0]?.id) ?? "Timestamp";
            const descending = sorting[0]?.desc ?? true;

            // default time range: from Unix epoch time to current datetime
            const timeFilter = columnFilters.find((f) => f.id === "startTime")?.value as
                | [dayjs.Dayjs | null, dayjs.Dayjs | null]
                | undefined;
            const endTime = timeFilter?.[1] ? timeFilter[1].toDate() : new Date();
            const startTime = timeFilter?.[0] ? timeFilter[0].toDate() : new Date(0);

            const fmt = (d: Date) => d.toISOString().slice(0, 19); // Example: "2006-01-02T15:04:05"

            const response = await axios.get<UserApiResponse>("/api/data", {
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
        },
        placeholderData: keepPreviousData, // don't go to 0 rows when refetching or paginating to next page
    });

    const columns = useMemo<MRT_ColumnDef<LogRecord>[]>(
        //column definitions...
        () => [
            {
                accessorKey: "traceId",
                header: "Trace ID",
                enableSorting: false,
            },
            {
                accessorKey: "spanId",
                header: "Span ID",
                enableSorting: false,
            },
            {
                accessorKey: "severityText",
                header: "Severity Text",
                enableSorting: false,
            },
            {
                accessorKey: "severityNumber",
                header: "Severity #",
                enableSorting: false,
            },
            {
                accessorKey: "serviceName",
                header: "Service Name",
                enableSorting: true,
            },
            {
                accessorKey: "body",
                header: "Body",
                enableSorting: false,
            },
            {
                accessorFn: (row) => new Date(row.timestamp),
                id: "startTime",
                header: "Time",
                filterVariant: "datetime-range",
                muiFilterDateTimePickerProps: ({ rangeFilterIndex }: { rangeFilterIndex: number }) => ({
                    label: rangeFilterIndex === 0 ? "Start" : "End",
                }),
                Cell: ({ cell }) =>
                    `${cell.getValue<Date>().toLocaleDateString()} ${cell.getValue<Date>().toLocaleTimeString()}`,
            },
            {
                accessorKey: "logAttributes",
                header: "Log Attributes",
                enableSorting: false,
                enableColumnFilter: false,
                Cell: ({ cell }) => (
                    <div style={{ display: "flex", flexWrap: "wrap", gap: 4 }}>
                        {cell.getValue<KeyValue[]>()?.map((attr, i) => (
                            <span key={`${i}-${attr.key}`}>
                                {attr.key}: {attr.value}
                            </span>
                        ))}
                    </div>
                ),
            },
            {
                accessorKey: "resourceAttributes",
                header: "Resource Attributes",
                enableSorting: false,
                enableColumnFilter: false,
                Cell: ({ cell }) => (
                    <div style={{ display: "flex", flexWrap: "wrap", gap: 4 }}>
                        {cell.getValue<KeyValue[]>()?.map((attr, i) => (
                            <span key={`${i}-${attr.key}`}>
                                {attr.key}: {attr.value}
                            </span>
                        ))}
                    </div>
                ),
            },
        ],
        [],
    );

    const table = useMaterialReactTable({
        columns,
        data,
        initialState: {
            showColumnFilters: true,
            columnFilters: [
                {
                    id: "startTime",
                    value: [new Date(0), null], // [earliest possible date, no end limit]
                },
            ],
        },
        manualFiltering: true, // turn off built-in client-side filtering
        manualPagination: true, // turn off built-in client-side pagination
        manualSorting: true, // turn off built-in client-side sorting
        muiFilterTextFieldProps: {
            variant: "filled",
        },
        muiToolbarAlertBannerProps: isError
            ? {
                  color: "error",
                  children: "Error loading data",
              }
            : undefined,
        onColumnFiltersChange: setColumnFilters,
        enableGlobalFilter: false,
        onPaginationChange: setPagination,
        onSortingChange: setSorting,
        renderTopToolbarCustomActions: () => (
            <Tooltip arrow title="Refresh Data">
                <IconButton onClick={() => refetch()}>
                    <RefreshIcon />
                </IconButton>
            </Tooltip>
        ),
        rowCount: meta?.totalRowCount ?? 0,
        state: {
            columnFilters,
            isLoading,
            pagination,
            showAlertBanner: isError,
            showProgressBars: isRefetching,
            sorting,
        },
    });

    return (
        <LocalizationProvider dateAdapter={AdapterDayjs}>
            <MaterialReactTable table={table} />
        </LocalizationProvider>
    );
}

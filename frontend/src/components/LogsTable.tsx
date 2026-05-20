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
import type { LogRecord } from "../types/LogRecord";
import { LocalizationProvider } from "@mui/x-date-pickers/LocalizationProvider";
import { AdapterDayjs } from "@mui/x-date-pickers/AdapterDayjs";
import axios from "axios";

type UserApiResponse = {
    data: Array<LogRecord>;
    meta: {
        totalRowCount: number;
    };
};

export default function EnhancedTable() {
    // manage our own state for stuff we want to pass to the API
    const [columnFilters, setColumnFilters] = useState<MRT_ColumnFiltersState>([]);
    const [globalFilter, setGlobalFilter] = useState("");
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
                globalFilter, // refetch when globalFilter changes
                pagination, // refetch when pagination changes
                sorting, // refetch when sorting changes
            },
        ],
        queryFn: async () => {
            try {
                const response = await axios.get("/api/data");
                console.log("data:", response.data);
                return response.data as UserApiResponse;
            } catch (error) {
                console.error("Axios request failed:", error);
                throw error; // Let React Query know it failed so it can set isError: true
            }

            // TODO: add filters
            // read our state and pass it to the API as query params
            // fetchURL.searchParams.set("start", `${pagination.pageIndex * pagination.pageSize}`);
            // fetchURL.searchParams.set("size", `${pagination.pageSize}`);
            // fetchURL.searchParams.set("filters", JSON.stringify(columnFilters ?? []));
            // fetchURL.searchParams.set("globalFilter", globalFilter ?? "");
            // fetchURL.searchParams.set("sorting", JSON.stringify(sorting ?? []));
        },
        placeholderData: keepPreviousData, // don't go to 0 rows when refetching or paginating to next page
    });

    const columns = useMemo<MRT_ColumnDef<LogRecord>[]>(
        //column definitions...
        () => [
            {
                accessorKey: "traceId",
                header: "Trace ID",
            },
            {
                accessorKey: "spanId",
                header: "Span ID",
            },
            {
                accessorKey: "severityText",
                header: "Severity Text",
            },
            {
                accessorKey: "severityNumber",
                header: "Severity #",
            },
            {
                accessorKey: "body",
                header: "body",
            },
            {
                accessorFn: (row) => new Date(row.timestamp),
                id: "timestamp",
                header: "Time",
                Cell: ({ cell }) => new Date(cell.getValue<Date>()).toLocaleString(),
                filterFn: "greaterThan",
                filterVariant: "date",
                enableGlobalFilter: false,
            },
        ],
        [],
    );

    const table = useMaterialReactTable({
        columns,
        data,
        initialState: { showColumnFilters: true },
        manualFiltering: true, // turn off built-in client-side filtering
        manualPagination: true, // turn off built-in client-side pagination
        manualSorting: true, // turn off built-in client-side sorting
        muiToolbarAlertBannerProps: isError
            ? {
                  color: "error",
                  children: "Error loading data",
              }
            : undefined,
        onColumnFiltersChange: setColumnFilters,
        onGlobalFilterChange: setGlobalFilter,
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
            globalFilter,
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

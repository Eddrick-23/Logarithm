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
import type { KeyValue, LogRecord } from "../types/Log";
import { LocalizationProvider } from "@mui/x-date-pickers/LocalizationProvider";
import { AdapterDayjs } from "@mui/x-date-pickers/AdapterDayjs";
import { useSearchLogs } from "../hooks/useSearchLogs";

export default function LogsTable() {
    const [columnFilters, setColumnFilters] = useState<MRT_ColumnFiltersState>([]);
    const [sorting, setSorting] = useState<MRT_SortingState>([]);
    const [pagination, setPagination] = useState<MRT_PaginationState>({
        pageIndex: 0,
        pageSize: 10,
    });
    const {
        data: { data: logs = [], meta } = {},
        isLoading,
        isError,
        isRefetching,
        refetch,
    } = useSearchLogs({
        columnFilters,
        pagination,
        sorting,
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
        data: logs,
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
                  severity: "error",
                  children: "Error loading data.",
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
        muiTableHeadCellProps: {
            sx: {
                color: "primary.main",
                fontSize: 12,
                textTransform: "uppercase",
                letterSpacing: "0.08em",
            },
        },
    });

    return (
        <LocalizationProvider dateAdapter={AdapterDayjs}>
            <MaterialReactTable table={table} />
        </LocalizationProvider>
    );
}

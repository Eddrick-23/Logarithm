import { useMemo, useState } from "react";
import {
    MaterialReactTable,
    useMaterialReactTable,
    type MRT_ColumnDef,
    type MRT_ColumnFiltersState,
    type MRT_PaginationState,
    type MRT_SortingState,
} from "material-react-table";
import { IconButton, InputAdornment, Stack, Tooltip } from "@mui/material";
import RefreshIcon from "@mui/icons-material/Refresh";
import type { KeyValue, LogRecord, LogType } from "../types/Log";
import { SEVERITY_NORMALISE_MAP } from "../utils/severity";
import { LocalizationProvider } from "@mui/x-date-pickers/LocalizationProvider";
import { AdapterDayjs } from "@mui/x-date-pickers/AdapterDayjs";
import { useSearchLogs } from "../hooks/useSearchLogs";
import SearchIcon from "@mui/icons-material/Search";
import SeverityPill from "./SeverityPill";

function normaliseSeverity(value: string): LogType {
    const lower = value.toLowerCase();
    return SEVERITY_NORMALISE_MAP[lower] ?? "info";
}

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
                grow: false, // should not grow in length since traceId length is fixed
                size: 295,
            },
            {
                accessorKey: "spanId",
                header: "Span ID",
                enableSorting: false,
                grow: false, // should not grow in length since spanId length is fixed
                size: 190,
            },
            {
                accessorKey: "severityText",
                header: "Severity Text",
                enableSorting: false,
                grow: false, // should not grow in length since severityText length is fixed
                size: 235,
                Cell: ({ cell }) => <SeverityPill severity={normaliseSeverity(cell.getValue<string>())} />,
            },
            {
                accessorKey: "severityNumber",
                header: "Severity #",
                enableSorting: false,
                grow: false, // should not grow in length since severityNumber length is fixed
                size: 200,
                muiFilterTextFieldProps: {
                    type: "number",
                    slotProps: {
                        input: {
                            startAdornment: (
                                <InputAdornment position="start">
                                    <SearchIcon />
                                </InputAdornment>
                            ),
                            endAdornment: null,
                        },
                    },
                },
            },
            {
                accessorKey: "serviceName",
                header: "Service Name",
                enableSorting: true,
                grow: false, // should not grow in length since serviceName length is fixed
                size: 240,
            },
            {
                accessorKey: "body",
                header: "Body",
                enableSorting: false,
                size: 240,
            },
            {
                accessorFn: (row) => new Date(row.timestamp),
                id: "startTime",
                header: "Time",
                filterVariant: "datetime-range",
                grow: false, // should not grow in length since time length is fixed
                size: 635,
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
                size: 250,
                Cell: ({ cell }) => (
                    <Stack direction="column" spacing={1}>
                        {cell.getValue<KeyValue[]>()?.map((attr, i) => (
                            <span key={`${i}-${attr.key}`}>
                                {attr.key}: {attr.value}
                            </span>
                        ))}
                    </Stack>
                ),
            },
            {
                accessorKey: "resourceAttributes",
                header: "Resource Attributes",
                enableSorting: false,
                enableColumnFilter: false,
                size: 250,
                Cell: ({ cell }) => (
                    <Stack direction="column" spacing={1}>
                        {cell.getValue<KeyValue[]>()?.map((attr, i) => (
                            <span key={`${i}-${attr.key}`}>
                                {attr.key}: {attr.value}
                            </span>
                        ))}
                    </Stack>
                ),
            },
        ],
        [],
    );

    const table = useMaterialReactTable({
        columns,
        data: logs,
        layoutMode: "grid",
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
        muiFilterTextFieldProps: ({ column }) => ({
            variant: "filled",
            placeholder: column.columnDef.header,
            slotProps: {
                input: {
                    startAdornment: (
                        <InputAdornment position="start">
                            <SearchIcon />
                        </InputAdornment>
                    ),
                },
            },
        }),
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

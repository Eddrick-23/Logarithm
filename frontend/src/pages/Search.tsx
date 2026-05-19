import { useQuery } from "@tanstack/react-query";
import EnhancedTable, { type Column } from "../components/EnhancedTable";
import type { LogRecord } from "../types/LogRecord";
import axios from "axios";
import Skeleton from "@mui/material/Skeleton";

const logColumns: Column<LogRecord>[] = [
    { id: "traceId", label: "Trace ID" },
    { id: "spanId", label: "Span ID" },
    { id: "severityText", label: "Severity" },
    { id: "severityNumber", label: "Severity #" },
    { id: "body", label: "Body" },
    {
        id: "timestamp",
        label: "Time",
        render: (value) => (value instanceof Date ? value.toLocaleString() : String(value)),
    },
];

export default function Search() {
    const { data, isLoading, isError } = useQuery({
        queryKey: ["search"],
        queryFn: async () => {
            const response = await axios.get("/api/data");
            return response.data.map((item: any) => ({
                ...item,
            })) as LogRecord[];
        },
    });

    const handleDelete = (ids: string[]) => {
        console.log("Delete rows:", ids);
        // TODO: call API and update state
    };

    if (isLoading) {
        return <Skeleton variant="rectangular" width="100%" height={100} />;
    }

    if (isError || !data) {
        return <div>Failed to load logs.</div>;
    }

    return (
        <EnhancedTable<LogRecord>
            title="Logs"
            rows={data}
            columns={logColumns}
            getRowId={(row) => row.traceId}
            onDelete={handleDelete}
            defaultRowsPerPage={10}
        />
    );
}
